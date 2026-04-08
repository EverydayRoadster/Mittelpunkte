package methods

import (
	"fmt"
	"math"
)

// ReliefCenterOfGravity calculates the point of balance on a 3D surface model.
type ReliefCenterOfGravity struct {
	Resolution float64 // Grid resolution in meters
}

func (m ReliefCenterOfGravity) Name() string { return "ReliefCenterOfGravity" }

func (m ReliefCenterOfGravity) Calculate(areas []Area) Point {
	if len(areas) == 0 {
		return Point{}
	}

	res := m.Resolution
	gridPoints := GenerateGridPoints(areas, res, m.Name())
	if len(gridPoints) == 0 {
		return Point{}
	}

	// Fetch elevations (batched)
	elevations, err := FetchElevations(gridPoints, m.Name())
	if err != nil {
		fmt.Printf("\nWarning: Could not fetch elevations: %v. Falling back to 2D CenterOfGravity.\n", err)
		return CenterOfGravity{Resolution: res}.Calculate(areas)
	}

	// Calculate weighted center of gravity in 3D Cartesian coordinates
	var xSum, ySum, zSum, sumWeight, sumElev float64
	
	minLat := math.MaxFloat64
	for _, a := range areas {
		for _, part := range a.Parts {
			for _, p := range part {
				if p.Lat < minLat { minLat = p.Lat }
			}
		}
	}
	
	// Map points to their elevations for easy lookup
	type key struct{ i, j int }
	elevMap := make(map[key]float64)
	
	// Re-estimate steps to re-index
	center := Point{Lat: gridPoints[0].Lat, Lon: gridPoints[0].Lon}
	pOffsetLat := Point{Lat: center.Lat + 0.1, Lon: center.Lon}
	pOffsetLon := Point{Lat: center.Lat, Lon: center.Lon + 0.1}
	mPerDegLat := center.DistanceTo(pOffsetLat) * 10.0
	mPerDegLon := center.DistanceTo(pOffsetLon) * 10.0
	stepLat := res / mPerDegLat
	stepLon := res / mPerDegLon

	for idx, p := range gridPoints {
		if idx%1000 == 0 {
			UpdateProgress(m.Name()+" (Map)", idx, len(gridPoints))
		}
		i := int((p.Lat - minLat) / stepLat + 0.5)
		j := int(p.Lon / stepLon + 0.5) // Simplified key
		elevMap[key{i, j}] = elevations[idx]
	}
	UpdateProgress(m.Name()+" (Map)", len(gridPoints), len(gridPoints))

	for idx, p := range gridPoints {
		if idx%1000 == 0 {
			UpdateProgress(m.Name()+" (Weight)", idx, len(gridPoints))
		}
		i := int((p.Lat - minLat) / stepLat + 0.5)
		j := int(p.Lon / stepLon + 0.5)
		
		z := elevations[idx]
		
		var dzdx, dzdy float64
		if zNext, ok := elevMap[key{i + 1, j}]; ok {
			dzdx = (zNext - z) / res
		}
		if zNext, ok := elevMap[key{i, j + 1}]; ok {
			dzdy = (zNext - z) / res
		}
		
		phi := p.Lat * math.Pi / 180
		lambda := p.Lon * math.Pi / 180
		
		// Weight each point by both cos(phi) (meridian convergence) 
		// and the surface area factor sqrt(1 + (dz/dx)^2 + (dz/dy)^2).
		reliefWeight := math.Sqrt(1 + dzdx*dzdx + dzdy*dzdy)
		weight := math.Cos(phi) * reliefWeight
		
		xSum += weight * math.Cos(phi) * math.Cos(lambda)
		ySum += weight * math.Cos(phi) * math.Sin(lambda)
		zSum += weight * math.Sin(phi)
		
		sumWeight += weight
		sumElev += z * reliefWeight // Average elevation weighted by surface area
	}
	UpdateProgress(m.Name()+" (Weight)", len(gridPoints), len(gridPoints))

	x := xSum / sumWeight
	y := ySum / sumWeight
	z := zSum / sumWeight

	lon := math.Atan2(y, x) * 180 / math.Pi
	hyp := math.Sqrt(x*x + y*y)
	lat := math.Atan2(z, hyp) * 180 / math.Pi

	return Point{
		Lat:       lat,
		Lon:       lon,
		Elevation: sumElev / sumWeight,
		Method:    m.Name(),
	}
}

func (m ReliefCenterOfGravity) SVG(areas []Area, p Point, t SVGTransformer) string {
	cx, cy := t.Project(p)
	// Draw a star shape
	return fmt.Sprintf(`<path d="M %f,%f l 2,-5 l 2,5 l 5,0 l -4,3 l 2,5 l -5,-3 l -5,3 l 2,-5 l -4,-3 z" fill="yellow" stroke="orange" stroke-width="1" />`,
		cx, cy)
}

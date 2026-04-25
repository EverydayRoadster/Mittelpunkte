package methods

import (
	"fmt"
	"math"
	"strings"
)

// ReliefCenterOfGravity calculates the point of balance on a 3D surface model.
type ReliefCenterOfGravity struct {
	Resolution float64 // Grid resolution in meters
}

func (m ReliefCenterOfGravity) Name() string { return "ReliefCenterOfGravity" }

func (m ReliefCenterOfGravity) Calculate(areas []Area) []Point {
	if len(areas) == 0 {
		return nil
	}

	res := m.Resolution
	gridPoints := GenerateGridPoints(areas, res, m.Name())
	if len(gridPoints) == 0 {
		return nil
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

	return []Point{{
		Lat:       lat,
		Lon:       lon,
		Elevation: sumElev / sumWeight,
		Method:    m.Name(),
	}}
}

func (m ReliefCenterOfGravity) SVG(areas []Area, points []Point, t SVGTransformer) string {
	if len(points) == 0 {
		return ""
	}
	p := points[0]
	cx, cy := t.Project(p)

	var sb strings.Builder

	// Draw distorted gravitational waves
	for i := 1; i <= 5; i++ {
		baseDist := float64(i) * 10000
		var polyPoints []string
		for angle := 0; angle <= 360; angle += 10 {
			rad := float64(angle) * math.Pi / 180
			latShift := (baseDist * math.Cos(rad)) / 111320.0
			lonShift := (baseDist * math.Sin(rad)) / (111320.0 * math.Cos(p.Lat*math.Pi/180))

			samplePoint := Point{Lat: p.Lat + latShift, Lon: p.Lon + lonShift}
			
			// Try to find nearby elevation in cache
			// We use a slightly fuzzy lookup for the wave distortion
			elev := p.Elevation // Fallback to center elevation
			key := getCacheKey(samplePoint)
			if e, ok := elevationCache[key]; ok {
				elev = e
			}

			// Distort: higher elevation "pushes" the wave out (simulating more mass/influence)
			// Scale distortion: 1000m = ~15% push
			distort := 1.0 + (elev / 7000.0) 
			
			x, y := t.Project(Point{
				Lat: p.Lat + latShift*distort,
				Lon: p.Lon + lonShift*distort,
			})
			polyPoints = append(polyPoints, fmt.Sprintf("%.2f,%.2f", x, y))
		}
		sb.WriteString(fmt.Sprintf(`<polygon points="%s" fill="none" stroke="orange" stroke-width="0.8" stroke-opacity="%.2f" stroke-dasharray="4,2" />`,
			strings.Join(polyPoints, " "), 0.7-float64(i)*0.1))
	}

	// Draw elevation-colored grid points
	res := m.Resolution
	gridPoints := GenerateGridPoints(areas, res, "")
	for i, gp := range gridPoints {
		if len(gridPoints) > 600 && i%(len(gridPoints)/600) != 0 {
			continue
		}
		x, y := t.Project(gp)
		elev := elevationCache[getCacheKey(gp)]
		
		// Color gradient: Green (low) -> Yellow -> Red (high)
		// Normalized to 0-2000m for color scaling
		norm := math.Max(0, math.Min(1, elev/2000.0))
		r := int(255 * norm)
		g := int(255 * (1 - norm))
		if norm > 0.5 {
			g = int(255 * (1 - norm) * 2)
		} else {
			r = int(255 * norm * 2)
		}
		
		sb.WriteString(fmt.Sprintf(`<circle cx="%.2f" cy="%.2f" r="0.9" fill="rgb(%d,%d,50)" fill-opacity="0.3" />`, 
			x, y, r, g))
	}

	// Draw a star shape for the center
	sb.WriteString(fmt.Sprintf(`<path d="M %f,%f l 2,-5 l 2,5 l 5,0 l -4,3 l 2,5 l -5,-3 l -5,3 l 2,-5 l -4,-3 z" fill="yellow" stroke="orange" stroke-width="1" />`,
		cx, cy))
	return sb.String()
}

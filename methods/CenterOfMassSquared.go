package methods

import (
	"fmt"
	"math"
)

// CenterOfMassSquared calculates the geographic center using the method 
// proposed by Peter Rogerson (2015) in "A New Method for Finding Geographic Centers".
// It finds the point that minimizes the sum of squared great-circle distances
// to all points in the region by calculating the 3D Cartesian mean of points 
// on the sphere and projecting it back to the surface.
type CenterOfMassSquared struct {
	Resolution float64
}

func (m CenterOfMassSquared) Name() string { return "CenterOfMassSquared" }

func (m CenterOfMassSquared) Calculate(areas []Area) Point {
	res := m.Resolution
	gridPoints := GenerateGridPoints(areas, res, m.Name())
	if len(gridPoints) == 0 {
		return Point{Method: m.Name()}
	}

	// 1. Convert to 3D Cartesian coordinates and average them.
	// We weight each point by cos(lat) to account for the grid point density
	// distortion in our GenerateGridPoints (which uses a constant degree step).
	// This ensures each grid point represents its true physical area.
	var xSum, ySum, zSum, weightSum float64
	for i, p := range gridPoints {
		if i%1000 == 0 {
			UpdateProgress(m.Name()+" (Avg)", i, len(gridPoints))
		}
		phi := p.Lat * math.Pi / 180
		lambda := p.Lon * math.Pi / 180
		
		// Weight is proportional to the area each grid point represents
		// A = (R * dLat) * (R * cos(lat) * dLon)
		weight := math.Cos(phi)
		
		xSum += weight * math.Cos(phi) * math.Cos(lambda)
		ySum += weight * math.Cos(phi) * math.Sin(lambda)
		zSum += weight * math.Sin(phi)
		weightSum += weight
	}
	UpdateProgress(m.Name()+" (Avg)", len(gridPoints), len(gridPoints))

	x := xSum / weightSum
	y := ySum / weightSum
	z := zSum / weightSum

	// 2. Convert back to Lat/Lon
	lon := math.Atan2(y, x) * 180 / math.Pi
	hyp := math.Sqrt(x*x + y*y)
	lat := math.Atan2(z, hyp) * 180 / math.Pi

	// Calculate average elevation
	var sumElev float64
	for i, p := range gridPoints {
		if i%1000 == 0 {
			UpdateProgress(m.Name()+" (Elev)", i, len(gridPoints))
		}
		sumElev += p.Elevation
	}
	UpdateProgress(m.Name()+" (Elev)", len(gridPoints), len(gridPoints))

	return Point{
		Lat:       lat,
		Lon:       lon,
		Elevation: sumElev / float64(len(gridPoints)),
		Method:    m.Name(),
	}
}

func (m CenterOfMassSquared) SVG(areas []Area, p Point, t SVGTransformer) string {
	cx, cy := t.Project(p)
	return fmt.Sprintf(`<circle cx="%.2f" cy="%.2f" r="5" fill="none" stroke="magenta" stroke-width="2" />`+
		`<circle cx="%.2f" cy="%.2f" r="1" fill="magenta" />`, cx, cy, cx, cy)
}

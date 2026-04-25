package methods

import (
	"fmt"
	"math"
	"strings"
)

// CenterOfGravity calculates the geographic center using the method 
// proposed by Peter Rogerson (2015) in "A New Method for Finding Geographic Centers".
// It finds the point that minimizes the sum of squared great-circle distances
// to all points in the region by calculating the 3D Cartesian mean of points 
// on the sphere and projecting it back to the surface.
// This is a grid-based approximation of the area's centroid.
type CenterOfGravity struct {
	Resolution float64 // Optional: Resolution in meters. If 0, uses 30m grid (clamped in main).
}

func (m CenterOfGravity) Name() string { return "CenterOfGravity" }

func (m CenterOfGravity) Calculate(areas []Area) []Point {
	if len(areas) == 0 {
		return nil
	}

	res := m.Resolution
	gridPoints := GenerateGridPoints(areas, res, m.Name())
	if len(gridPoints) == 0 {
		return []Point{{Method: m.Name()}}
	}

	var xSum, ySum, zSum, weightSum float64
	for i, p := range gridPoints {
		if i%1000 == 0 {
			UpdateProgress(m.Name()+" (Avg)", i, len(gridPoints))
		}
		phi := p.Lat * math.Pi / 180
		lambda := p.Lon * math.Pi / 180
		
		// Weight each point by cos(phi) to account for varying grid point density
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

	lon := math.Atan2(y, x) * 180 / math.Pi
	hyp := math.Sqrt(x*x + y*y)
	lat := math.Atan2(z, hyp) * 180 / math.Pi

	return []Point{{
		Lat:    lat,
		Lon:    lon,
		Method: m.Name(),
	}}
}

func (m CenterOfGravity) SVG(areas []Area, points []Point, t SVGTransformer) string {
	if len(points) == 0 {
		return ""
	}
	p := points[0]
	cx, cy := t.Project(p)

	res := m.Resolution
	gridPoints := GenerateGridPoints(areas, res, "")
	if len(gridPoints) == 0 {
		return ""
	}

	var sb strings.Builder

	// Draw gravitational waves
	// We'll draw 5 concentric circles representing equal distances
	for i := 1; i <= 5; i++ {
		// Use a representative distance (e.g., 5km, 10km...) based on area size could be better,
		// but 10km steps are often reasonable for countries.
		// Let's try to scale it to the area.
		dist := float64(i) * 10000 // 10km, 20km...
		rSVG := t.ProjectRadius(dist, p)
		sb.WriteString(fmt.Sprintf(`<circle cx="%.2f" cy="%.2f" r="%.2f" fill="none" stroke="blue" stroke-width="0.5" stroke-opacity="%.2f" stroke-dasharray="5,5" />`,
			cx, cy, rSVG, 0.5-float64(i)*0.08))
	}

	for i, gp := range gridPoints {
		// Limit to ~800 dots for performance and visual clarity
		if len(gridPoints) > 800 && i%(len(gridPoints)/800) != 0 {
			continue
		}
		x, y := t.Project(gp)
		sb.WriteString(fmt.Sprintf(`<circle cx="%.2f" cy="%.2f" r="0.8" fill="blue" fill-opacity="0.15" />`, x, y))
	}
	return sb.String()
}

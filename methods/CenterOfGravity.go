package methods

import (
	"fmt"
	"strings"
)

// CenterOfGravity calculates the point of balance using a grid-based approximation.
type CenterOfGravity struct {
	Resolution float64 // Optional: Resolution in meters. If 0, uses 200x200 grid.
}

func (m CenterOfGravity) Name() string { return "CenterOfGravity" }

func (m CenterOfGravity) Calculate(areas []Area) Point {
	if len(areas) == 0 {
		return Point{}
	}

	res := m.Resolution
	if res <= 0 {
		res = 100.0 // Default
	}

	gridPoints := GenerateGridPoints(areas, res)
	if len(gridPoints) == 0 {
		return Point{Method: m.Name()}
	}

	var sumLat, sumLon float64
	for _, p := range gridPoints {
		sumLat += p.Lat
		sumLon += p.Lon
	}

	return Point{
		Lat:    sumLat / float64(len(gridPoints)),
		Lon:    sumLon / float64(len(gridPoints)),
		Method: m.Name(),
	}
}

func (m CenterOfGravity) SVG(areas []Area, p Point, t SVGTransformer) string {
	res := m.Resolution
	if res <= 0 {
		res = 100.0 // Larger resolution for visualization to avoid huge SVGs
	}
	gridPoints := GenerateGridPoints(areas, res)
	if len(gridPoints) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, gp := range gridPoints {
		// Limit to ~1000 dots for performance
		if len(gridPoints) > 1000 && i%(len(gridPoints)/1000) != 0 { continue }
		x, y := t.Project(gp)
		sb.WriteString(fmt.Sprintf(`<circle cx="%.2f" cy="%.2f" r="1" fill="blue" fill-opacity="0.3" />`, x, y))
	}
	return sb.String()
}

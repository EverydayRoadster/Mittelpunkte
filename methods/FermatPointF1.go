package methods

import (
	"fmt"
	"math"
)

// FermatPointF1 calculates the point where the sum of distances to all 
// other equally sized subareas (grid points) is the lowest.
type FermatPointF1 struct {
	Resolution float64
}

func (m FermatPointF1) Name() string { return "FermatPointF1" }

func (m FermatPointF1) Calculate(areas []Area) Point {
	res := m.Resolution
	if res <= 0 {
		res = 30.0 // Default 30m
	}

	gridPoints := GenerateGridPoints(areas, res)
	if len(gridPoints) == 0 {
		return Point{}
	}

	// For each grid point, calculate the sum of distances to all OTHER grid points.
	// This is an O(N^2) operation.
	var minSum float64 = math.MaxFloat64
	var bestPoint Point

	for i, p1 := range gridPoints {
		var currentSum float64
		for j, p2 := range gridPoints {
			if i == j {
				continue
			}
			// Use simple Euclidean distance for performance in the nested loop
			// (Relative differences are similar to Haversine on this small scale)
			dLat := p1.Lat - p2.Lat
			dLon := p1.Lon - p2.Lon
			dist := math.Sqrt(dLat*dLat + dLon*dLon)
			currentSum += dist
		}

		if currentSum < minSum {
			minSum = currentSum
			bestPoint = p1
		}
	}

	bestPoint.Method = m.Name()
	return bestPoint
}

func (m FermatPointF1) SVG(areas []Area, p Point, t SVGTransformer) string {
	cx, cy := t.Project(p)
	return fmt.Sprintf(`<circle cx="%.2f" cy="%.2f" r="5" fill="none" stroke="cyan" stroke-width="2" />`+
		`<circle cx="%.2f" cy="%.2f" r="1" fill="cyan" />`, cx, cy, cx, cy)
}

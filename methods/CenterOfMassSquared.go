package methods

import (
	"fmt"
	"math"
)

// CenterOfMassSquared calculates the point where the sum of squared distances 
// to all other equally sized subareas (grid points) is the lowest.
// Based on the 2015 article "A New Method for Finding Geographic Centers".
type CenterOfMassSquared struct {
	Resolution float64
}

func (m CenterOfMassSquared) Name() string { return "CenterOfMassSquared" }

func (m CenterOfMassSquared) Calculate(areas []Area) Point {
	res := m.Resolution
	if res <= 0 {
		res = 30.0 // Default 30m
	}

	gridPoints := GenerateGridPoints(areas, res)
	if len(gridPoints) == 0 {
		return Point{}
	}

	// For each grid point, calculate the sum of SQUARED distances to all OTHER grid points.
	var minSum float64 = math.MaxFloat64
	var bestPoint Point

	for i, p1 := range gridPoints {
		var currentSum float64
		for j, p2 := range gridPoints {
			if i == j {
				continue
			}
			// Sum of squared distances: (x1-x2)^2 + (y1-y2)^2
			dLat := p1.Lat - p2.Lat
			dLon := p1.Lon - p2.Lon
			distSq := dLat*dLat + dLon*dLon
			currentSum += distSq
		}

		if currentSum < minSum {
			minSum = currentSum
			bestPoint = p1
		}
	}

	bestPoint.Method = m.Name()
	return bestPoint
}

func (m CenterOfMassSquared) SVG(areas []Area, p Point, t SVGTransformer) string {
	cx, cy := t.Project(p)
	return fmt.Sprintf(`<circle cx="%.2f" cy="%.2f" r="5" fill="none" stroke="magenta" stroke-width="2" />`+
		`<circle cx="%.2f" cy="%.2f" r="1" fill="magenta" />`, cx, cy, cx, cy)
}

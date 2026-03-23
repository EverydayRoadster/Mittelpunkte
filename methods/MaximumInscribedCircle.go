package methods

import (
	"math"
)

// MaximumInscribedCircle calculates the point furthest from the boundary.
type MaximumInscribedCircle struct {
	Resolution float64
}

func (m MaximumInscribedCircle) Name() string { return "MaximumInscribedCircle" }

func (m MaximumInscribedCircle) Calculate(areas []Area) Point {
	res := m.Resolution
	if res <= 0 {
		res = 30.0 // Default 30m
	}

	// Initial grid search
	gridPoints := GenerateGridPoints(areas, res)
	if len(gridPoints) == 0 {
		return Point{}
	}

	var maxDist float64 = -math.MaxFloat64
	var bestPoint Point

	for _, p := range gridPoints {
		dist := DistanceToBoundary(p, areas)
		if dist > maxDist {
			maxDist = dist
			bestPoint = p
		}
	}

	// Refinement: Recursive search around the best point
	// We use a smaller and smaller grid around the current best point.
	currRes := res / 2.0
	for iter := 0; iter < 5; iter++ { // 5 iterations of refinement
		foundBetter := false
		// Search in a 3x3 grid around bestPoint
		for i := -1; i <= 1; i++ {
			for j := -1; j <= 1; j++ {
				if i == 0 && j == 0 {
					continue
				}
				
				// Calculate lat/lon offsets based on meters
				center := bestPoint
				pOffsetLat := Point{Lat: center.Lat + 0.01, Lon: center.Lon}
				pOffsetLon := Point{Lat: center.Lat, Lon: center.Lon + 0.01}
				mPerDegLat := center.DistanceTo(pOffsetLat) * 100.0
				mPerDegLon := center.DistanceTo(pOffsetLon) * 100.0
				
				testPoint := Point{
					Lat: bestPoint.Lat + float64(i)*currRes/mPerDegLat,
					Lon: bestPoint.Lon + float64(j)*currRes/mPerDegLon,
				}

				// Verify point is inside at least one area
				inside := false
				for _, a := range areas {
					if IsPointInPolygon(testPoint.Lat, testPoint.Lon, a.Points) {
						inside = true
						break
					}
				}

				if inside {
					dist := DistanceToBoundary(testPoint, areas)
					if dist > maxDist {
						maxDist = dist
						bestPoint = testPoint
						foundBetter = true
					}
				}
			}
		}
		if !foundBetter {
			// If no better point found in this grid, we still shrink the grid
			// to search more finely around the current best point.
		}
		currRes /= 2.0
	}

	// Set elevation to average of border points to be consistent.
	sumElev := 0.0
	numPoints := 0
	for _, a := range areas {
		for _, p := range a.Points {
			sumElev += p.Elevation
			numPoints++
		}
	}
	if numPoints > 0 {
		bestPoint.Elevation = sumElev / float64(numPoints)
	}

	bestPoint.Method = m.Name()
	return bestPoint
}

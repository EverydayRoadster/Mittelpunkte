package methods

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// ElevationQuartiles calculates middle points based on elevation quartiles (Q1, Q2, Q3).
// For each quartile, it finds all points within 0.5m of that elevation and averages them.
type ElevationQuartiles struct {
	Resolution float64 // Grid resolution in meters
}

func (m ElevationQuartiles) Name() string { return "ElevationQuartiles" }

func (m ElevationQuartiles) Calculate(areas []Area) []Point {
	if len(areas) == 0 {
		return nil
	}

	res := m.Resolution
	gridPoints := GenerateGridPoints(areas, res, m.Name())
	if len(gridPoints) == 0 {
		return nil
	}

	// Fetch elevations
	elevations, err := FetchElevations(gridPoints, m.Name())
	if err != nil {
		fmt.Printf("\nWarning: Could not fetch elevations for ElevationQuartiles: %v\n", err)
		return nil
	}

	// Filter out points with no elevation data if API returned invalid results (though opentopodata usually returns 0 or null)
	// We'll assume all points in elevations correspond to gridPoints.

	// Sort elevations to find quartiles
	sortedElevs := make([]float64, len(elevations))
	copy(sortedElevs, elevations)
	sort.Float64s(sortedElevs)

	n := len(sortedElevs)
	if n < 4 {
		return nil
	}

	getQuartile := func(q float64) float64 {
		pos := q * float64(n-1)
		index := int(pos)
		fraction := pos - float64(index)
		if index+1 < n {
			return sortedElevs[index] + fraction*(sortedElevs[index+1]-sortedElevs[index])
		}
		return sortedElevs[index]
	}

	q1 := getQuartile(0.25)
	q2 := getQuartile(0.50)
	q3 := getQuartile(0.75)

	quartiles := []struct {
		val  float64
		name string
	}{
		{q1, "Q1"},
		{q2, "Q2/Median"},
		{q3, "Q3"},
	}

	var results []Point
	tolerance := 0.5

	for _, q := range quartiles {
		var sumLat, sumLon float64
		var count int

		for i, elev := range elevations {
			if math.Abs(elev-q.val) <= tolerance {
				sumLat += gridPoints[i].Lat
				sumLon += gridPoints[i].Lon
				count++
			}
		}

		if count > 0 {
			results = append(results, Point{
				Lat:       sumLat / float64(count),
				Lon:       sumLon / float64(count),
				Elevation: q.val,
				Method:    fmt.Sprintf("%s (%s: %.1fm)", m.Name(), q.name, q.val),
			})
		} else {
			// Fallback: find the single closest point if none within tolerance
			minDiff := math.MaxFloat64
			var bestIdx int
			for i, elev := range elevations {
				diff := math.Abs(elev - q.val)
				if diff < minDiff {
					minDiff = diff
					bestIdx = i
				}
			}
			results = append(results, Point{
				Lat:       gridPoints[bestIdx].Lat,
				Lon:       gridPoints[bestIdx].Lon,
				Elevation: q.val,
				Method:    fmt.Sprintf("%s (%s: %.1fm) [closest]", m.Name(), q.name, q.val),
			})
		}
	}

	return results
}

func (m ElevationQuartiles) SVG(areas []Area, points []Point, t SVGTransformer) string {
	if len(points) == 0 {
		return ""
	}
	
	res := m.Resolution
	gridPoints := GenerateGridPoints(areas, res, "")
	if len(gridPoints) == 0 {
		return ""
	}

	elevations, err := FetchElevations(gridPoints, "")
	if err != nil {
		return ""
	}

	// Recalculate quartiles to identify contributing points
	sortedElevs := make([]float64, len(elevations))
	copy(sortedElevs, elevations)
	sort.Float64s(sortedElevs)

	n := len(sortedElevs)
	if n < 4 {
		return ""
	}

	getQuartile := func(q float64) float64 {
		pos := q * float64(n-1)
		index := int(pos)
		fraction := pos - float64(index)
		if index+1 < n {
			return sortedElevs[index] + fraction*(sortedElevs[index+1]-sortedElevs[index])
		}
		return sortedElevs[index]
	}

	q1 := getQuartile(0.25)
	q2 := getQuartile(0.50)
	q3 := getQuartile(0.75)

	var sb strings.Builder
	// We use a slightly larger tolerance for the visualization (dots) 
	// to show the "line" or area of the quartile more clearly in the illustration.
	vizTolerance := 1.0

	for i, elev := range elevations {
		color := ""
		if math.Abs(elev-q1) <= vizTolerance {
			color = "red"
		} else if math.Abs(elev-q2) <= vizTolerance {
			color = "blue"
		} else if math.Abs(elev-q3) <= vizTolerance {
			color = "green"
		}

		if color != "" {
			x, y := t.Project(gridPoints[i])
			sb.WriteString(fmt.Sprintf(`<circle cx="%.2f" cy="%.2f" r="2" fill="%s" fill-opacity="0.3" />`, x, y, color))
		}
	}
	
	return sb.String()
}

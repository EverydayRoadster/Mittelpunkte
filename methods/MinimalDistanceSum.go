package methods

import (
	"math"
)

// MinimalDistanceSum calculates the point that minimizes the sum of distances to all border points.
// This is also known as the Geometric Median of the boundary points.
type MinimalDistanceSum struct{}

func (m MinimalDistanceSum) Name() string { return "MinimalDistanceSum" }

func (m MinimalDistanceSum) Calculate(areas []Area) Point {
	var points []Point
	sumElev := 0.0
	for _, a := range areas {
		points = append(points, a.Points...)
		for _, p := range a.Points {
			sumElev += p.Elevation
		}
	}

	if len(points) == 0 {
		return Point{}
	}

	avgElev := sumElev / float64(len(points))

	// Initial guess: Centroid (arithmetic mean)
	var curr Point
	for _, p := range points {
		curr.Lat += p.Lat
		curr.Lon += p.Lon
	}
	curr.Lat /= float64(len(points))
	curr.Lon /= float64(len(points))

	// Weiszfeld's algorithm
	const iterations = 100
	const epsilon = 1e-10

	for i := 0; i < iterations; i++ {
		var nextLat, nextLon, totalWeight float64
		foundExact := false

		for _, p := range points {
			d := dist(curr, p)
			if d < epsilon {
				// If current guess is exactly on a border point, we handle it
				foundExact = true
				break
			}
			weight := 1.0 / d
			nextLat += p.Lat * weight
			nextLon += p.Lon * weight
			totalWeight += weight
		}

		if foundExact {
			// In most cases for geographic areas, the median won't be exactly on a point.
			// If it is, Weiszfeld's algorithm requires a more complex update,
			// but for this "simple approach", we can stop or slightly nudge.
			break
		}

		next := Point{
			Lat: nextLat / totalWeight,
			Lon: nextLon / totalWeight,
		}

		// Check for convergence
		if dist(curr, next) < epsilon {
			curr = next
			break
		}
		curr = next
	}

	curr.Elevation = avgElev
	curr.Method = m.Name()
	return curr
}

func dist(p1, p2 Point) float64 {
	dLat := p1.Lat - p2.Lat
	dLon := p1.Lon - p2.Lon
	return math.Sqrt(dLat*dLat + dLon*dLon)
}

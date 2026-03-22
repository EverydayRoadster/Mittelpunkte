package methods

import (
	"math"
)

// MinimalDistanceSumEqualSpaced calculates the point that minimizes the sum of distances 
// to points equally spaced every 10 meters along the border.
type MinimalDistanceSumEqualSpaced struct{}

func (m MinimalDistanceSumEqualSpaced) Name() string { return "MinimalDistanceSumEqualSpaced" }

func (m MinimalDistanceSumEqualSpaced) Calculate(areas []Area) Point {
	const spacing = 10.0 // 10 meters spacing

	var sampledPoints []Point
	sumElev := 0.0
	pointCountBoundary := 0

	for _, a := range areas {
		if len(a.Points) < 2 {
			continue
		}
		
		// For each segment in the area's polygon
		for i := 0; i < len(a.Points); i++ {
			p1 := a.Points[i]
			// Boundary is typically a closed loop in Shapefiles (p[0] == p[last])
			// But for resampling segments, we handle all segments.
			nextIdx := (i + 1) % len(a.Points)
			p2 := a.Points[nextIdx]

			segmentDist := p1.DistanceTo(p2)
			if segmentDist == 0 {
				continue
			}

			// Number of points to sample in this segment
			samples := int(segmentDist / spacing)
			for s := 0; s < samples; s++ {
				fraction := float64(s) * spacing / segmentDist
				
				// Interpolate Lat/Lon linearly (fine for small areas)
				lat := p1.Lat + (p2.Lat-p1.Lat)*fraction
				lon := p1.Lon + (p2.Lon-p1.Lon)*fraction
				elev := p1.Elevation + (p2.Elevation-p1.Elevation)*fraction
				
				sampledPoints = append(sampledPoints, Point{Lat: lat, Lon: lon, Elevation: elev})
			}
			
			sumElev += p1.Elevation
			pointCountBoundary++
		}
	}

	if len(sampledPoints) == 0 {
		return Point{}
	}

	avgElev := sumElev / float64(pointCountBoundary)

	// Weiszfeld's algorithm
	// Initial guess: Centroid (arithmetic mean of sampled points)
	var curr Point
	for _, p := range sampledPoints {
		curr.Lat += p.Lat
		curr.Lon += p.Lon
	}
	curr.Lat /= float64(len(sampledPoints))
	curr.Lon /= float64(len(sampledPoints))

	const iterations = 100
	const epsilon = 1e-10

	for i := 0; i < iterations; i++ {
		var nextLat, nextLon, totalWeight float64
		foundExact := false

		for _, p := range sampledPoints {
			// We use DistanceTo for weighting (actual meters)
			d := curr.DistanceTo(p)
			if d < epsilon {
				foundExact = true
				break
			}
			weight := 1.0 / d
			nextLat += p.Lat * weight
			nextLon += p.Lon * weight
			totalWeight += weight
		}

		if foundExact {
			break
		}

		next := Point{
			Lat: nextLat / totalWeight,
			Lon: nextLon / totalWeight,
		}

		// Check for convergence (in degrees is fine here)
		distDegrees := math.Sqrt(math.Pow(curr.Lat-next.Lat, 2) + math.Pow(curr.Lon-next.Lon, 2))
		if distDegrees < 1e-12 {
			curr = next
			break
		}
		curr = next
	}

	curr.Elevation = avgElev
	curr.Method = m.Name()
	return curr
}

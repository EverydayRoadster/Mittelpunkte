package methods

import (
	"fmt"
	"math"
	"strings"
)

// MinimalDistanceSum calculates the point that minimizes the sum of distances to all border points.
// This is also known as the Geometric Median of the boundary points.
// This version accounts for the Earth's curvature by using great-circle distances.
type MinimalDistanceSum struct{}

func (m MinimalDistanceSum) Name() string { return "MinimalDistanceSum" }

func (m MinimalDistanceSum) Calculate(areas []Area) Point {
	var points []Point
	sumElev := 0.0
	for _, a := range areas {
		for _, part := range a.Parts {
			points = append(points, part...)
			for _, p := range part {
				sumElev += p.Elevation
			}
		}
	}

	if len(points) == 0 {
		return Point{}
	}

	avgElev := sumElev / float64(len(points))

	// Initial guess: 3D Centroid
	var xSum, ySum, zSum, weightSum float64
	for _, p := range points {
		phi := p.Lat * math.Pi / 180
		lambda := p.Lon * math.Pi / 180
		w := math.Cos(phi) // Weight to account for meridian convergence
		xSum += w * math.Cos(phi) * math.Cos(lambda)
		ySum += w * math.Cos(phi) * math.Sin(lambda)
		zSum += w * math.Sin(phi)
		weightSum += w
	}
	
	curr := Point{
		Lat: math.Atan2(zSum/weightSum, math.Sqrt(math.Pow(xSum/weightSum, 2)+math.Pow(ySum/weightSum, 2))) * 180 / math.Pi,
		Lon: math.Atan2(ySum/weightSum, xSum/weightSum) * 180 / math.Pi,
	}

	// Weiszfeld's algorithm using Great Circle distances
	const iterations = 100
	const epsilon = 1e-10

	for i := 0; i < iterations; i++ {
		var nextX, nextY, nextZ, totalWeight float64
		foundExact := false

		for _, p := range points {
			// Use Haversine distance
			d := curr.DistanceTo(p)
			if d < 1.0 { // 1 meter threshold for "exact"
				foundExact = true
				break
			}
			
			// Weight is 1/d, further adjusted by cos(lat) for boundary point density
			phiP := p.Lat * math.Pi / 180
			lambdaP := p.Lon * math.Pi / 180
			w := math.Cos(phiP) / d
			
			nextX += w * math.Cos(phiP) * math.Cos(lambdaP)
			nextY += w * math.Cos(phiP) * math.Sin(lambdaP)
			nextZ += w * math.Sin(phiP)
			totalWeight += w
		}

		if foundExact || totalWeight == 0 {
			break
		}

		next := Point{
			Lat: math.Atan2(nextZ/totalWeight, math.Sqrt(math.Pow(nextX/totalWeight, 2)+math.Pow(nextY/totalWeight, 2))) * 180 / math.Pi,
			Lon: math.Atan2(nextY/totalWeight, nextX/totalWeight) * 180 / math.Pi,
		}

		// Check for convergence (approx 1mm)
		if curr.DistanceTo(next) < 0.001 {
			curr = next
			break
		}
		curr = next
	}

	curr.Elevation = avgElev
	curr.Method = m.Name()
	return curr
}

func (m MinimalDistanceSum) SVG(areas []Area, p Point, t SVGTransformer) string {
	var sb strings.Builder
	cx, cy := t.Project(p)
	for _, a := range areas {
		for _, part := range a.Parts {
			for i, pt := range part {
				if i%5 != 0 { continue }
				px, py := t.Project(pt)
				sb.WriteString(fmt.Sprintf(`<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="grey" stroke-width="0.5" stroke-opacity="0.3" />`, cx, cy, px, py))
			}
		}
	}
	return sb.String()
}

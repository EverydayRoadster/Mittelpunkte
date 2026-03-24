package methods

import (
	"fmt"
	"math"
	"strings"
)

// MinimalDistanceSumEqualSpaced calculates the point that minimizes the sum of distances
// but uses equally spaced points along the boundary instead of just the vertices.
type MinimalDistanceSumEqualSpaced struct {
	Spacing float64 // Spacing in meters
}

func (m MinimalDistanceSumEqualSpaced) Name() string { return "MinimalDistanceSumEqualSpaced" }

func (m MinimalDistanceSumEqualSpaced) Calculate(areas []Area) Point {
	spacing := m.Spacing
	if spacing <= 0 {
		spacing = 50.0 // Default 50m spacing
	}

	var points []Point
	for _, a := range areas {
		for _, part := range a.Parts {
			if len(part) < 2 {
				points = append(points, part...)
				continue
			}

			for i := 0; i < len(part); i++ {
				p1 := part[i]
				nextIdx := (i + 1) % len(part)
				p2 := part[nextIdx]

				d := p1.DistanceTo(p2)
				points = append(points, p1)

				if d > spacing {
					steps := int(d / spacing)
					for s := 1; s <= steps; s++ {
						frac := float64(s) / (d / spacing)
						points = append(points, Point{
							Lat:       p1.Lat + (p2.Lat-p1.Lat)*frac,
							Lon:       p1.Lon + (p2.Lon-p1.Lon)*frac,
							Elevation: p1.Elevation + (p2.Elevation-p1.Elevation)*frac,
						})
					}
				}
			}
		}
	}

	// Now apply the same logic as MinimalDistanceSum
	if len(points) == 0 {
		return Point{}
	}

	var curr Point
	for _, p := range points {
		curr.Lat += p.Lat
		curr.Lon += p.Lon
	}
	curr.Lat /= float64(len(points))
	curr.Lon /= float64(len(points))

	// Weiszfeld's algorithm
	const iterations = 50
	const epsilon = 1e-10

	for i := 0; i < iterations; i++ {
		var nextLat, nextLon, totalWeight float64
		for _, p := range points {
			d := math.Sqrt(math.Pow(curr.Lat-p.Lat, 2) + math.Pow(curr.Lon-p.Lon, 2))
			if d < epsilon {
				continue
			}
			weight := 1.0 / d
			nextLat += p.Lat * weight
			nextLon += p.Lon * weight
			totalWeight += weight
		}
		if totalWeight == 0 {
			break
		}
		next := Point{Lat: nextLat / totalWeight, Lon: nextLon / totalWeight}
		if math.Sqrt(math.Pow(curr.Lat-next.Lat, 2)+math.Pow(curr.Lon-next.Lon, 2)) < epsilon {
			break
		}
		curr = next
	}

	curr.Method = m.Name()
	return curr
}

func (m MinimalDistanceSumEqualSpaced) SVG(areas []Area, p Point, t SVGTransformer) string {
	var sb strings.Builder
	cx, cy := t.Project(p)
	
	// Collect points for visualization
	var points []Point
	for _, a := range areas {
		for _, part := range a.Parts {
			for i := 0; i < len(part); i++ {
				p1 := part[i]
				p2 := part[(i+1)%len(part)]
				d := p1.DistanceTo(p2)
				points = append(points, p1)
				if d > 200 { // Use 200m for visualization
					steps := int(d / 200)
					for s := 1; s <= steps; s++ {
						frac := float64(s) / (d / 200)
						points = append(points, Point{
							Lat: p1.Lat + (p2.Lat-p1.Lat)*frac,
							Lon: p1.Lon + (p2.Lon-p1.Lon)*frac,
						})
					}
				}
			}
		}
	}

	for i, pt := range points {
		if i%10 != 0 { continue }
		px, py := t.Project(pt)
		sb.WriteString(fmt.Sprintf(`<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="green" stroke-width="0.3" stroke-opacity="0.2" />`, cx, cy, px, py))
	}
	return sb.String()
}

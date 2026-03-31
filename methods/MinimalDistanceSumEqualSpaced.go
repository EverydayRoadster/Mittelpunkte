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

	// Now apply the same logic as MinimalDistanceSum (Geometric Median)
	if len(points) == 0 {
		return Point{}
	}

	// Initial guess: 3D Centroid
	var xSum, ySum, zSum, weightSum float64
	for i, p := range points {
		if i%1000 == 0 {
			UpdateProgress(m.Name()+" (Init)", i, len(points))
		}
		phi := p.Lat * math.Pi / 180
		lambda := p.Lon * math.Pi / 180
		w := math.Cos(phi) // Weight to account for meridian convergence
		xSum += w * math.Cos(phi) * math.Cos(lambda)
		ySum += w * math.Cos(phi) * math.Sin(lambda)
		zSum += w * math.Sin(phi)
		weightSum += w
	}
	UpdateProgress(m.Name()+" (Init)", len(points), len(points))
	
	curr := Point{
		Lat: math.Atan2(zSum/weightSum, math.Sqrt(math.Pow(xSum/weightSum, 2)+math.Pow(ySum/weightSum, 2))) * 180 / math.Pi,
		Lon: math.Atan2(ySum/weightSum, xSum/weightSum) * 180 / math.Pi,
	}

	// Weiszfeld's algorithm using Great Circle distances
	const iterations = 50
	const epsilon = 1e-10

	for i := 0; i < iterations; i++ {
		UpdateProgress(m.Name()+" (Iter)", i, iterations)
		var nextX, nextY, nextZ, totalWeight float64
		foundExact := false

		for _, p := range points {
			d := curr.DistanceTo(p)
			if d < 1.0 { // 1 meter threshold
				foundExact = true
				break
			}
			
			// For EqualSpaced, we don't need the extra cos(phi) density weight 
			// because points are already equally distributed in 3D space.
			// However, we still need it if our generation was done in Lat/Lon steps.
			// Actually DistanceTo and spacing are in meters, so we are good.
			// But the Density of points along a parallel at high lat is higher 
			// if we use degree steps. Here we use spacing in meters, so it's uniform.
			w := 1.0 / d
			
			phiP := p.Lat * math.Pi / 180
			lambdaP := p.Lon * math.Pi / 180
			
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

		if curr.DistanceTo(next) < 0.001 {
			curr = next
			break
		}
		curr = next
	}
	UpdateProgress(m.Name()+" (Iter)", iterations, iterations)

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
				if d > 200 { 
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

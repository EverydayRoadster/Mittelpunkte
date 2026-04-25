package methods

import (
	"fmt"
	"math"
	"strings"
)

// FermatPointF1 calculates the point that minimizes the sum of great-circle distances 
// to all grid points inside the area. This is the Geometric Median of the area.
type FermatPointF1 struct {
	Resolution float64
}

func (m FermatPointF1) Name() string { return "FermatPointF1" }

func (m FermatPointF1) Calculate(areas []Area) []Point {
	res := m.Resolution
	gridPoints := GenerateGridPoints(areas, res, m.Name())
	if len(gridPoints) == 0 {
		return nil
	}

	// This is the same problem as MinimalDistanceSum but for the whole area instead of just boundary.
	// We use the same robust Weiszfeld algorithm in 3D.
	
	var xSum, ySum, zSum, weightSum float64
	for i, p := range gridPoints {
		if i%1000 == 0 {
			UpdateProgress(m.Name()+" (Init)", i, len(gridPoints))
		}
		phi := p.Lat * math.Pi / 180
		lambda := p.Lon * math.Pi / 180
		w := math.Cos(phi) // Weight to account for meridian convergence
		xSum += w * math.Cos(phi) * math.Cos(lambda)
		ySum += w * math.Cos(phi) * math.Sin(lambda)
		zSum += w * math.Sin(phi)
		weightSum += w
	}
	UpdateProgress(m.Name()+" (Init)", len(gridPoints), len(gridPoints))
	
	curr := Point{
		Lat: math.Atan2(zSum/weightSum, math.Sqrt(math.Pow(xSum/weightSum, 2)+math.Pow(ySum/weightSum, 2))) * 180 / math.Pi,
		Lon: math.Atan2(ySum/weightSum, xSum/weightSum) * 180 / math.Pi,
	}

	const iterations = 50
	for i := 0; i < iterations; i++ {
		UpdateProgress(m.Name()+" (Iter)", i, iterations)
		var nextX, nextY, nextZ, totalWeight float64
		foundExact := false

		for _, p := range gridPoints {
			d := curr.DistanceTo(p)
			if d < 1.0 { 
				foundExact = true
				break
			}
			
			phiP := p.Lat * math.Pi / 180
			lambdaP := p.Lon * math.Pi / 180
			// Weight each point by cos(phiP) for density AND 1/d for median
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

		if curr.DistanceTo(next) < 0.001 {
			curr = next
			break
		}
		curr = next
	}
	UpdateProgress(m.Name()+" (Iter)", iterations, iterations)

	curr.Method = m.Name()
	return []Point{curr}
}

func (m FermatPointF1) SVG(areas []Area, points []Point, t SVGTransformer) string {
	if len(points) == 0 {
		return ""
	}
	p := points[0]
	cx, cy := t.Project(p)

	var sb strings.Builder

	// Use a sparse grid for visualization
	res := m.Resolution
	if res == 0 {
		res = 3000
	}
	gridPoints := GenerateGridPoints(areas, res*5, "") // 5x coarser for visual lines

	for _, gp := range gridPoints {
		gx, gy := t.Project(gp)
		// Draw a small box (representing area mass)
		sb.WriteString(fmt.Sprintf(`<rect x="%.2f" y="%.2f" width="2" height="2" fill="cyan" fill-opacity="0.4" />`,
			gx-1, gy-1))
		// Draw a line connecting to the Fermat point
		sb.WriteString(fmt.Sprintf(`<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="cyan" stroke-width="0.3" stroke-opacity="0.2" />`,
			gx, gy, cx, cy))
	}

	sb.WriteString(fmt.Sprintf(`<circle cx="%.2f" cy="%.2f" r="5" fill="none" stroke="cyan" stroke-width="2" />`+
		`<circle cx="%.2f" cy="%.2f" r="1" fill="cyan" />`, cx, cy, cx, cy))
	return sb.String()
}

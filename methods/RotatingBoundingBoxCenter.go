package methods

import (
	"fmt"
	"math"
	"strings"
)

// RotatingBoundingBoxCenter calculates the average center of bounding boxes at 1-degree intervals.
// This version uses a local projection to avoid Lat/Lon rotation distortion.
type RotatingBoundingBoxCenter struct{}

func (m RotatingBoundingBoxCenter) Name() string { return "RotatingBoundingBoxCenter" }

func (m RotatingBoundingBoxCenter) Calculate(areas []Area) []Point {
	var points []Point
	var sumLat, sumLon, sumElev float64
	for _, a := range areas {
		for _, part := range a.Parts {
			points = append(points, part...)
			for _, p := range part {
				sumLat += p.Lat
				sumLon += p.Lon
				sumElev += p.Elevation
			}
		}
	}

	if len(points) == 0 {
		return nil
	}

	ref := Point{Lat: sumLat / float64(len(points)), Lon: sumLon / float64(len(points))}
	
	const R = 6371000.0
	const rad = math.Pi / 180.0
	toLocal := func(p Point) (float64, float64) {
		y := (p.Lat - ref.Lat) * rad * R
		x := (p.Lon - ref.Lon) * rad * R * math.Cos(ref.Lat*rad)
		return x, y
	}
	fromLocal := func(x, y float64) Point {
		lat := ref.Lat + (y / R / rad)
		lon := ref.Lon + (x / R / rad / math.Cos(ref.Lat*rad))
		return Point{Lat: lat, Lon: lon}
	}

	localPoints := make([][2]float64, len(points))
	for i, p := range points {
		if i%1000 == 0 {
			UpdateProgress(m.Name()+" (Init)", i, len(points))
		}
		px, py := toLocal(p)
		localPoints[i] = [2]float64{px, py}
	}
	UpdateProgress(m.Name()+" (Init)", len(points), len(points))

	var resX, resY float64
	for degree := 0; degree < 360; degree++ {
		UpdateProgress(m.Name()+" (Iter)", degree, 360)
		mx, my := m.rotatedCenterLocal(localPoints, degree)
		resX += mx
		resY += my
	}
	UpdateProgress(m.Name()+" (Iter)", 360, 360)

	res := fromLocal(resX/360.0, resY/360.0)
	res.Elevation = sumElev / float64(len(points))
	res.Method = m.Name()
	return []Point{res}
}

func (m RotatingBoundingBoxCenter) rotatedCenterLocal(points [][2]float64, degree int) (float64, float64) {
	angle := float64(degree) * math.Pi / 180.0
	cosR := math.Cos(angle)
	sinR := math.Sin(angle)

	minX, maxX := math.MaxFloat64, -math.MaxFloat64
	minY, maxY := math.MaxFloat64, -math.MaxFloat64

	for _, p := range points {
		rx := p[0]*cosR - p[1]*sinR
		ry := p[0]*sinR + p[1]*cosR

		if rx < minX { minX = rx }
		if rx > maxX { maxX = rx }
		if ry < minY { minY = ry }
		if ry > maxY { maxY = ry }
	}

	midX := (minX + maxX) / 2.0
	midY := (minY + maxY) / 2.0

	origX := midX*cosR + midY*sinR
	origY := -midX*sinR + midY*cosR
	return origX, origY
}

func (m RotatingBoundingBoxCenter) SVG(areas []Area, points []Point, t SVGTransformer) string {
	if len(points) == 0 {
		return ""
	}
	// p := points[0] // Unused in this method's SVG implementation, but available
	
	var allBoundaryPoints []Point
	var sumLat, sumLon float64
	for _, a := range areas {
		for _, part := range a.Parts {
			allBoundaryPoints = append(allBoundaryPoints, part...)
			for _, pt := range part {
				sumLat += pt.Lat
				sumLon += pt.Lon
			}
		}
	}
	if len(allBoundaryPoints) == 0 {
		return ""
	}
	
	ref := Point{Lat: sumLat / float64(len(allBoundaryPoints)), Lon: sumLon / float64(len(allBoundaryPoints))}
	const R = 6371000.0
	const rad = math.Pi / 180.0
	toLocal := func(p Point) (float64, float64) {
		y := (p.Lat - ref.Lat) * rad * R
		x := (p.Lon - ref.Lon) * rad * R * math.Cos(ref.Lat*rad)
		return x, y
	}
	fromLocal := func(x, y float64) Point {
		lat := ref.Lat + (y / R / rad)
		lon := ref.Lon + (x / R / rad / math.Cos(ref.Lat*rad))
		return Point{Lat: lat, Lon: lon}
	}

	var sb strings.Builder
	// Draw boxes for some angles
	angles := []int{0, 45, 90, 135}
	colors := []string{"green", "olive", "lime", "teal"}
	for idx, degree := range angles {
		angle := float64(degree) * math.Pi / 180.0
		cosR := math.Cos(angle)
		sinR := math.Sin(angle)

		minX, maxX := math.MaxFloat64, -math.MaxFloat64
		minY, maxY := math.MaxFloat64, -math.MaxFloat64
		for _, pt := range allBoundaryPoints {
			px, py := toLocal(pt)
			rx := px*cosR - py*sinR
			ry := px*sinR + py*cosR
			if rx < minX { minX = rx }
			if rx > maxX { maxX = rx }
			if ry < minY { minY = ry }
			if ry > maxY { maxY = ry }
		}

		// Box corners in rotated frame
		corners := [][2]float64{{minX, minY}, {maxX, minY}, {maxX, maxY}, {minX, maxY}}
		var polyPoints []string
		for _, c := range corners {
			ox := c[0]*cosR + c[1]*sinR
			oy := -c[0]*sinR + c[1]*cosR
			orig := fromLocal(ox, oy)
			sx, sy := t.Project(orig)
			polyPoints = append(polyPoints, fmt.Sprintf("%.2f,%.2f", sx, sy))
		}
		sb.WriteString(fmt.Sprintf(`<polygon points="%s" fill="none" stroke="%s" stroke-width="1" stroke-dasharray="2,2" />`,
			strings.Join(polyPoints, " "), colors[idx]))
	}
	return sb.String()
}

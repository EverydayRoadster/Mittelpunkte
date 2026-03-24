package methods

import (
	"fmt"
	"math"
	"strings"
)

// RotatingBoundingBoxCenter calculates the average center of bounding boxes at 1-degree intervals.
type RotatingBoundingBoxCenter struct{}

func (m RotatingBoundingBoxCenter) Name() string { return "RotatingBoundingBoxCenter" }

func (m RotatingBoundingBoxCenter) Calculate(areas []Area) Point {
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

	var sumLat, sumLon float64

	for degree := 0; degree < 360; degree++ {
		origLat, origLon := m.rotatedCenter(points, degree)
		sumLon += origLon
		sumLat += origLat
	}

	return Point{
		Lat:       sumLat / 360.0,
		Lon:       sumLon / 360.0,
		Elevation: avgElev,
		Method:    m.Name(),
	}
}

func (m RotatingBoundingBoxCenter) rotatedCenter(points []Point, degree int) (float64, float64) {
	rad := float64(degree) * math.Pi / 180.0
	cosR := math.Cos(rad)
	sinR := math.Sin(rad)

	minX, maxX := math.MaxFloat64, -math.MaxFloat64
	minY, maxY := math.MaxFloat64, -math.MaxFloat64

	for _, p := range points {
		rx := p.Lon*cosR - p.Lat*sinR
		ry := p.Lon*sinR + p.Lat*cosR

		if rx < minX { minX = rx }
		if rx > maxX { maxX = rx }
		if ry < minY { minY = ry }
		if ry > maxY { maxY = ry }
	}

	midX := (minX + maxX) / 2.0
	midY := (minY + maxY) / 2.0

	origLon := midX*cosR + midY*sinR
	origLat := -midX*sinR + midY*cosR
	return origLat, origLon
}

func (m RotatingBoundingBoxCenter) SVG(areas []Area, p Point, t SVGTransformer) string {
	var points []Point
	for _, a := range areas {
		for _, part := range a.Parts {
			points = append(points, part...)
		}
	}
	if len(points) == 0 {
		return ""
	}

	var sb strings.Builder
	// Draw boxes for some angles
	angles := []int{0, 45, 90, 135}
	colors := []string{"green", "olive", "lime", "teal"}
	for idx, degree := range angles {
		rad := float64(degree) * math.Pi / 180.0
		cosR := math.Cos(rad)
		sinR := math.Sin(rad)

		minX, maxX := math.MaxFloat64, -math.MaxFloat64
		minY, maxY := math.MaxFloat64, -math.MaxFloat64
		for _, pt := range points {
			rx := pt.Lon*cosR - pt.Lat*sinR
			ry := pt.Lon*sinR + pt.Lat*cosR
			if rx < minX { minX = rx }
			if rx > maxX { maxX = rx }
			if ry < minY { minY = ry }
			if ry > maxY { maxY = ry }
		}

		// Box corners in rotated frame
		corners := [][2]float64{{minX, minY}, {maxX, minY}, {maxX, maxY}, {minX, maxY}}
		var polyPoints []string
		for _, c := range corners {
			origLon := c[0]*cosR + c[1]*sinR
			origLat := -c[0]*sinR + c[1]*cosR
			sx, sy := t.Project(Point{Lat: origLat, Lon: origLon})
			polyPoints = append(polyPoints, fmt.Sprintf("%.2f,%.2f", sx, sy))
		}
		sb.WriteString(fmt.Sprintf(`<polygon points="%s" fill="none" stroke="%s" stroke-width="1" stroke-dasharray="2,2" />`,
			strings.Join(polyPoints, " "), colors[idx]))
	}
	return sb.String()
}

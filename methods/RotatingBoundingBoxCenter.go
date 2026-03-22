package methods

import (
	"math"
)

// RotatingBoundingBoxCenter calculates the average center of bounding boxes at 1-degree intervals.
type RotatingBoundingBoxCenter struct{}

func (m RotatingBoundingBoxCenter) Name() string { return "RotatingBoundingBoxCenter" }

func (m RotatingBoundingBoxCenter) Calculate(areas []Area) Point {
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

	var sumLat, sumLon float64

	for degree := 0; degree < 360; degree++ {
		rad := float64(degree) * math.Pi / 180.0
		cosR := math.Cos(rad)
		sinR := math.Sin(rad)

		// Rotate all points and find bounding box
		minX, maxX := math.MaxFloat64, -math.MaxFloat64
		minY, maxY := math.MaxFloat64, -math.MaxFloat64

		for _, p := range points {
			// Standard rotation
			rx := p.Lon*cosR - p.Lat*sinR
			ry := p.Lon*sinR + p.Lat*cosR

			if rx < minX {
				minX = rx
			}
			if rx > maxX {
				maxX = rx
			}
			if ry < minY {
				minY = ry
			}
			if ry > maxY {
				maxY = ry
			}
		}

		// Center in rotated frame
		midX := (minX + maxX) / 2.0
		midY := (minY + maxY) / 2.0

		// Rotate center back (using -rad)
		// x = x'cos(-rad) - y'sin(-rad) = x'cos(rad) + y'sin(rad)
		// y = x'sin(-rad) + y'cos(-rad) = -x'sin(rad) + y'cos(rad)
		origLon := midX*cosR + midY*sinR
		origLat := -midX*sinR + midY*cosR

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

package methods

import (
	"fmt"
)

// BoundingBoxCenter calculates the average between maximum and minimum lat and lon.
type BoundingBoxCenter struct{}

func (m BoundingBoxCenter) Name() string { return "BoundingBoxCenter" }

func (m BoundingBoxCenter) Calculate(areas []Area) Point {
	var points []Point
	for _, a := range areas {
		points = append(points, a.Points...)
	}

	if len(points) == 0 {
		return Point{}
	}
	minLat, maxLat := points[0].Lat, points[0].Lat
	minLon, maxLon := points[0].Lon, points[0].Lon
	sumElev := 0.0

	for _, p := range points {
		if p.Lat < minLat {
			minLat = p.Lat
		}
		if p.Lat > maxLat {
			maxLat = p.Lat
		}
		if p.Lon < minLon {
			minLon = p.Lon
		}
		if p.Lon > maxLon {
			maxLon = p.Lon
		}
		sumElev += p.Elevation
	}

	return Point{
		Lat:       (minLat + maxLat) / 2.0,
		Lon:       (minLon + maxLon) / 2.0,
		Elevation: sumElev / float64(len(points)),
		Method:    m.Name(),
	}
}

func (m BoundingBoxCenter) SVG(areas []Area, p Point, t SVGTransformer) string {
	var points []Point
	for _, a := range areas {
		points = append(points, a.Points...)
	}
	if len(points) == 0 {
		return ""
	}
	minLat, maxLat := points[0].Lat, points[0].Lat
	minLon, maxLon := points[0].Lon, points[0].Lon
	for _, p := range points {
		if p.Lat < minLat { minLat = p.Lat }
		if p.Lat > maxLat { maxLat = p.Lat }
		if p.Lon < minLon { minLon = p.Lon }
		if p.Lon > maxLon { maxLon = p.Lon }
	}

	x1, y1 := t.Project(Point{Lat: maxLat, Lon: minLon})
	x2, y2 := t.Project(Point{Lat: minLat, Lon: maxLon})
	
	return fmt.Sprintf(`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="none" stroke="blue" stroke-width="2" stroke-dasharray="5,5" />`,
		x1, y1, x2-x1, y2-y1)
}

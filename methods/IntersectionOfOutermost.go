package methods

import (
	"fmt"
	"math"
)

// IntersectionOfOutermost calculates the intersection of lines between the outermost points of lat and lon.
type IntersectionOfOutermost struct{}

func (m IntersectionOfOutermost) Name() string { return "IntersectionOfOutermost" }

func (m IntersectionOfOutermost) Calculate(areas []Area) Point {
	var points []Point
	for _, a := range areas {
		for _, part := range a.Parts {
			points = append(points, part...)
		}
	}

	if len(points) == 0 {
		return Point{}
	}

	var pMinLat, pMaxLat, pMinLon, pMaxLon Point
	pMinLat, pMaxLat, pMinLon, pMaxLon = points[0], points[0], points[0], points[0]
	var sumLat, sumLon, sumElev float64

	for _, p := range points {
		if p.Lat < pMinLat.Lat { pMinLat = p }
		if p.Lat > pMaxLat.Lat { pMaxLat = p }
		if p.Lon < pMinLon.Lon { pMinLon = p }
		if p.Lon > pMaxLon.Lon { pMaxLon = p }
		sumLat += p.Lat
		sumLon += p.Lon
		sumElev += p.Elevation
	}
	
	avgLat := sumLat / float64(len(points))
	avgLon := sumLon / float64(len(points))
	ref := Point{Lat: avgLat, Lon: avgLon}
	
	// Project to local tangent plane to avoid lat/lon distortion
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

	x1, y1 := toLocal(pMinLat)
	x2, y2 := toLocal(pMaxLat)
	x3, y3 := toLocal(pMinLon)
	x4, y4 := toLocal(pMaxLon)

	denom := (x1-x2)*(y3-y4) - (y1-y2)*(x3-x4)
	if math.Abs(denom) < 1e-9 {
		return BoundingBoxCenter{}.Calculate(areas)
	}

	intersectX := ((x1*y2-y1*x2)*(x3-x4) - (x1-x2)*(x3*y4-y3*x4)) / denom
	intersectY := ((x1*y2-y1*x2)*(y3-y4) - (y1-y2)*(x3*y4-y3*x4)) / denom

	res := fromLocal(intersectX, intersectY)
	res.Elevation = sumElev / float64(len(points))
	res.Method = m.Name()
	return res
}

func (m IntersectionOfOutermost) SVG(areas []Area, p Point, t SVGTransformer) string {
	var points []Point
	for _, a := range areas {
		for _, part := range a.Parts {
			points = append(points, part...)
		}
	}
	if len(points) == 0 {
		return ""
	}
	var pMinLat, pMaxLat, pMinLon, pMaxLon Point
	pMinLat, pMaxLat, pMinLon, pMaxLon = points[0], points[0], points[0], points[0]
	for _, pt := range points {
		if pt.Lat < pMinLat.Lat { pMinLat = pt }
		if pt.Lat > pMaxLat.Lat { pMaxLat = pt }
		if pt.Lon < pMinLon.Lon { pMinLon = pt }
		if pt.Lon > pMaxLon.Lon { pMaxLon = pt }
	}

	x1, y1 := t.Project(pMinLat)
	x2, y2 := t.Project(pMaxLat)
	x3, y3 := t.Project(pMinLon)
	x4, y4 := t.Project(pMaxLon)

	return fmt.Sprintf(`<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="red" stroke-width="2" />`+
		`<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="red" stroke-width="2" />`,
		x1, y1, x2, y2, x3, y3, x4, y4)
}

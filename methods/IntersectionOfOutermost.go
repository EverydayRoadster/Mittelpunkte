package methods

// IntersectionOfOutermost calculates the intersection of lines between the outermost points of lat and lon.
type IntersectionOfOutermost struct{}

func (m IntersectionOfOutermost) Name() string { return "IntersectionOfOutermost" }

func (m IntersectionOfOutermost) Calculate(areas []Area) Point {
	var points []Point
	for _, a := range areas {
		points = append(points, a.Points...)
	}

	if len(points) == 0 {
		return Point{}
	}

	var pMinLat, pMaxLat, pMinLon, pMaxLon Point
	pMinLat = points[0]
	pMaxLat = points[0]
	pMinLon = points[0]
	pMaxLon = points[0]
	sumElev := 0.0

	for _, p := range points {
		if p.Lat < pMinLat.Lat {
			pMinLat = p
		}
		if p.Lat > pMaxLat.Lat {
			pMaxLat = p
		}
		if p.Lon < pMinLon.Lon {
			pMinLon = p
		}
		if p.Lon > pMaxLon.Lon {
			pMaxLon = p
		}
		sumElev += p.Elevation
	}

	// Line 1: pMinLat to pMaxLat
	// Line 2: pMinLon to pMaxLon
	x1, y1 := pMinLat.Lon, pMinLat.Lat
	x2, y2 := pMaxLat.Lon, pMaxLat.Lat
	x3, y3 := pMinLon.Lon, pMinLon.Lat
	x4, y4 := pMaxLon.Lon, pMaxLon.Lat

	denom := (x1-x2)*(y3-y4) - (y1-y2)*(x3-x4)
	if denom == 0 {
		// Parallel lines, fallback to bounding box center for this case
		return BoundingBoxCenter{}.Calculate(areas)
	}

	intersectX := ((x1*y2-y1*x2)*(x3-x4) - (x1-x2)*(x3*y4-y3*x4)) / denom
	intersectY := ((x1*y2-y1*x2)*(y3-y4) - (y1-y2)*(x3*y4-y3*x4)) / denom

	return Point{
		Lat:       intersectY,
		Lon:       intersectX,
		Elevation: sumElev / float64(len(points)),
		Method:    m.Name(),
	}
}

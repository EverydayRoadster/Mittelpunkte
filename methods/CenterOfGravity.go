package methods

import (
	"math"
)

// CenterOfGravity calculates the point of balance using a grid-based approximation.
type CenterOfGravity struct{}

func (m CenterOfGravity) Name() string { return "CenterOfGravity" }

func (m CenterOfGravity) Calculate(areas []Area) Point {
	if len(areas) == 0 {
		return Point{}
	}

	// 1. Determine the global bounding box
	minLat, maxLat := math.MaxFloat64, -math.MaxFloat64
	minLon, maxLon := math.MaxFloat64, -math.MaxFloat64
	sumElevBoundary := 0.0
	pointCountBoundary := 0

	for _, a := range areas {
		for _, p := range a.Points {
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
			sumElevBoundary += p.Elevation
			pointCountBoundary++
		}
	}

	if pointCountBoundary == 0 {
		return Point{}
	}

	// 2. Sample the bounding box with a grid
	const resolution = 200 // 200x200 grid
	stepLat := (maxLat - minLat) / float64(resolution)
	stepLon := (maxLon - minLon) / float64(resolution)

	var sumLat, sumLon float64
	countInside := 0

	for i := 0; i < resolution; i++ {
		lat := minLat + (float64(i)+0.5)*stepLat
		for j := 0; j < resolution; j++ {
			lon := minLon + (float64(j)+0.5)*stepLon

			// Check if (lat, lon) is inside ANY area
			isInsideAny := false
			for _, a := range areas {
				if isPointInPolygon(lat, lon, a.Points) {
					isInsideAny = true
					break
				}
			}

			if isInsideAny {
				sumLat += lat
				sumLon += lon
				countInside++
			}
		}
	}

	// 3. Result
	if countInside == 0 {
		// Fallback to average of boundary points if no grid points are inside
		// (should only happen for extremely thin or degenerate polygons)
		return Point{
			Lat:       (minLat + maxLat) / 2.0,
			Lon:       (minLon + maxLon) / 2.0,
			Elevation: sumElevBoundary / float64(pointCountBoundary),
			Method:    m.Name(),
		}
	}

	return Point{
		Lat:       sumLat / float64(countInside),
		Lon:       sumLon / float64(countInside),
		Elevation: sumElevBoundary / float64(pointCountBoundary), // Elevation remains boundary average
		Method:    m.Name(),
	}
}

// isPointInPolygon implements the ray casting algorithm.
func isPointInPolygon(lat, lon float64, polygon []Point) bool {
	inside := false
	for i, j := 0, len(polygon)-1; i < len(polygon); j, i = i, i+1 {
		if ((polygon[i].Lat > lat) != (polygon[j].Lat > lat)) &&
			(lon < (polygon[j].Lon-polygon[i].Lon)*(lat-polygon[i].Lat)/(polygon[j].Lat-polygon[i].Lat)+polygon[i].Lon) {
			inside = !inside
		}
	}
	return inside
}

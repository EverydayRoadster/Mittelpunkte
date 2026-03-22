package methods

import (
	"math"
)

// CenterOfGravity calculates the point of balance using a grid-based approximation.
type CenterOfGravity struct {
	Resolution float64 // Optional: Resolution in meters. If 0, uses 200x200 grid.
}

func (m CenterOfGravity) Name() string { return "CenterOfGravity" }

func (m CenterOfGravity) Calculate(areas []Area) Point {
	if len(areas) == 0 {
		return Point{}
	}

	var gridPoints []Point
	if m.Resolution > 0 {
		gridPoints = GenerateGridPoints(areas, m.Resolution)
	} else {
		// Default 200x200 if no resolution provided
		minLat, maxLat := math.MaxFloat64, -math.MaxFloat64
		minLon, maxLon := math.MaxFloat64, -math.MaxFloat64
		for _, a := range areas {
			for _, p := range a.Points {
				if p.Lat < minLat { minLat = p.Lat }
				if p.Lat > maxLat { maxLat = p.Lat }
				if p.Lon < minLon { minLon = p.Lon }
				if p.Lon > maxLon { maxLon = p.Lon }
			}
		}
		
		res := 200
		stepLat := (maxLat - minLat) / float64(res)
		stepLon := (maxLon - minLon) / float64(res)
		for i := 0; i < res; i++ {
			lat := minLat + (float64(i)+0.5)*stepLat
			for j := 0; j < res; j++ {
				lon := minLon + (float64(j)+0.5)*stepLon
				inside := false
				for _, a := range areas {
					if IsPointInPolygon(lat, lon, a.Points) {
						inside = true
						break
					}
				}
				if inside {
					gridPoints = append(gridPoints, Point{Lat: lat, Lon: lon})
				}
			}
		}
	}

	if len(gridPoints) == 0 {
		return Point{}
	}

	var sumLat, sumLon float64
	for _, p := range gridPoints {
		sumLat += p.Lat
		sumLon += p.Lon
	}

	return Point{
		Lat:    sumLat / float64(len(gridPoints)),
		Lon:    sumLon / float64(len(gridPoints)),
		Method: m.Name(),
	}
}

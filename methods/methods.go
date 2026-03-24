package methods

import "math"

// Point represents a geographical point with elevation.
type Point struct {
	Lat       float64
	Lon       float64
	Elevation float64
	Method    string
}

// DistanceTo calculates the distance in meters to another point using the Haversine formula.
func (p Point) DistanceTo(other Point) float64 {
	const R = 6371000.0 // Earth radius in meters
	phi1 := p.Lat * math.Pi / 180
	phi2 := other.Lat * math.Pi / 180
	dphi := (other.Lat - p.Lat) * math.Pi / 180
	dlambda := (other.Lon - p.Lon) * math.Pi / 180

	a := math.Sin(dphi/2)*math.Sin(dphi/2) +
		math.Cos(phi1)*math.Cos(phi2)*
			math.Sin(dlambda/2)*math.Sin(dlambda/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

// Area represents a named set of points (e.g., a village boundary).
type Area struct {
	Name   string
	Level  string
	Parts  [][]Point
}

// CalculationMethod is the interface for different middle point calculation methods.
type CalculationMethod interface {
	Name() string
	Calculate(areas []Area) Point
	SVG(areas []Area, p Point, t SVGTransformer) string
}

// SVGTransformer handles projection of Lat/Lon to SVG coordinates.
type SVGTransformer struct {
	MinLat, MaxLat float64
	MinLon, MaxLon float64
	Width, Height  float64
}

// Project converts a geographical point to SVG coordinates.
func (t SVGTransformer) Project(p Point) (float64, float64) {
	if t.MaxLon == t.MinLon || t.MaxLat == t.MinLat {
		return 0, 0
	}
	x := (p.Lon - t.MinLon) / (t.MaxLon - t.MinLon) * t.Width
	y := (1.0 - (p.Lat-t.MinLat)/(t.MaxLat-t.MinLat)) * t.Height
	return x, y
}

// ProjectRadius converts a distance in meters to SVG units (approximate).
func (t SVGTransformer) ProjectRadius(meters float64, center Point) float64 {
	p2 := Point{Lat: center.Lat, Lon: center.Lon + 0.01}
	dist := center.DistanceTo(p2)
	degPerMeter := 0.01 / dist
	lonRange := t.MaxLon - t.MinLon
	return (meters * degPerMeter / lonRange) * t.Width
}


// GenerateGridPoints generates a grid of points inside the provided areas based on resolution in meters.
func GenerateGridPoints(areas []Area, resolution float64) []Point {
	if len(areas) == 0 || resolution <= 0 {
		return nil
	}

	minLat, maxLat := math.MaxFloat64, -math.MaxFloat64
	minLon, maxLon := math.MaxFloat64, -math.MaxFloat64
	for _, a := range areas {
		for _, part := range a.Parts {
			for _, p := range part {
				if p.Lat < minLat { minLat = p.Lat }
				if p.Lat > maxLat { maxLat = p.Lat }
				if p.Lon < minLon { minLon = p.Lon }
				if p.Lon > maxLon { maxLon = p.Lon }
			}
		}
	}

	center := Point{Lat: (minLat + maxLat) / 2.0, Lon: (minLon + maxLon) / 2.0}
	pOffsetLat := Point{Lat: center.Lat + 0.1, Lon: center.Lon}
	pOffsetLon := Point{Lat: center.Lat, Lon: center.Lon + 0.1}
	
	mPerDegLat := center.DistanceTo(pOffsetLat) * 10.0
	mPerDegLon := center.DistanceTo(pOffsetLon) * 10.0
	
	stepLat := resolution / mPerDegLat
	stepLon := resolution / mPerDegLon

	var gridPoints []Point
	latSteps := int((maxLat-minLat)/stepLat) + 1
	lonSteps := int((maxLon-minLon)/stepLon) + 1

	for i := 0; i < latSteps; i++ {
		lat := minLat + (float64(i)+0.5)*stepLat
		for j := 0; j < lonSteps; j++ {
			lon := minLon + (float64(j)+0.5)*stepLon
			
			inside := false
			for _, a := range areas {
				for _, part := range a.Parts {
					if IsPointInPolygon(lat, lon, part) {
						inside = true
						break
					}
				}
				if inside { break }
			}
			if inside {
				gridPoints = append(gridPoints, Point{Lat: lat, Lon: lon})
			}
		}
	}
	return gridPoints
}

// IsPointInPolygon implements the ray casting algorithm.
func IsPointInPolygon(lat, lon float64, polygon []Point) bool {
	inside := false
	for i, j := 0, len(polygon)-1; i < len(polygon); j, i = i, i+1 {
		if ((polygon[i].Lat > lat) != (polygon[j].Lat > lat)) &&
			(lon < (polygon[j].Lon-polygon[i].Lon)*(lat-polygon[i].Lat)/(polygon[j].Lat-polygon[i].Lat)+polygon[i].Lon) {
			inside = !inside
		}
	}
	return inside
}

// DistanceToSegment calculates the minimum distance from a point to a line segment.
func DistanceToSegment(p, v, w Point) float64 {
	dLat := w.Lat - v.Lat
	dLon := w.Lon - v.Lon
	l2 := dLat*dLat + dLon*dLon
	if l2 == 0 {
		return p.DistanceTo(v)
	}
	t := ((p.Lat-v.Lat)*dLat + (p.Lon-v.Lon)*dLon) / l2
	t = math.Max(0, math.Min(1, t))
	projection := Point{
		Lat: v.Lat + t*dLat,
		Lon: v.Lon + t*dLon,
	}
	return p.DistanceTo(projection)
}

// DistanceToBoundary calculates the minimum distance from a point to the boundary of the area.
func DistanceToBoundary(p Point, areas []Area) float64 {
	minDist := math.MaxFloat64
	for _, a := range areas {
		for _, part := range a.Parts {
			for i := 0; i < len(part); i++ {
				j := (i + 1) % len(part)
				d := DistanceToSegment(p, part[i], part[j])
				if d < minDist {
					minDist = d
				}
			}
		}
	}
	return minDist
}

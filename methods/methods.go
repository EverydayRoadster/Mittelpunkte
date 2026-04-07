package methods

import (
	"fmt"
	"math"
	"strings"
)

// UpdateProgress prints a progress bar to the terminal.
func UpdateProgress(method string, current, total int) {
	if total <= 0 {
		return
	}
	width := 40
	percent := float64(current) / float64(total)
	filled := int(percent * float64(width))
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("*", filled) + strings.Repeat(".", width-filled)
	fmt.Printf("\r%-25s [%s] %3d%%", method, bar, int(percent*100))
	if current >= total {
		fmt.Println()
	}
}

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

// ProjectRadiusX converts a distance in meters to SVG units along the X axis.
func (t SVGTransformer) ProjectRadiusX(meters float64, center Point) float64 {
	p2 := Point{Lat: center.Lat, Lon: center.Lon + 0.001}
	dist := center.DistanceTo(p2)
	return (meters / dist) * (0.001 / (t.MaxLon - t.MinLon)) * t.Width
}

// ProjectRadiusY converts a distance in meters to SVG units along the Y axis.
func (t SVGTransformer) ProjectRadiusY(meters float64, center Point) float64 {
	p2 := Point{Lat: center.Lat + 0.001, Lon: center.Lon}
	dist := center.DistanceTo(p2)
	return (meters / dist) * (0.001 / (t.MaxLat - t.MinLat)) * t.Height
}

// ProjectRadius is a legacy alias for ProjectRadiusX.
func (t SVGTransformer) ProjectRadius(meters float64, center Point) float64 {
	return t.ProjectRadiusX(meters, center)
}


// GenerateGridPoints generates a grid of points inside the provided areas based on resolution in meters.
func GenerateGridPoints(areas []Area, resolution float64, methodName string) []Point {
	if len(areas) == 0 || resolution <= 0 {
		return nil
	}

	minLat, maxLat, minLon, maxLon := getBoundingBox(areas)
	stepLat, stepLon := getGridSteps(minLat, maxLat, minLon, maxLon, resolution)

	var gridPoints []Point
	latSteps := int((maxLat-minLat)/stepLat) + 1
	lonSteps := int((maxLon-minLon)/stepLon) + 1

	for i := 0; i < latSteps; i++ {
		if methodName != "" && i%10 == 0 {
			UpdateProgress(methodName+" (Grid)", i, latSteps)
		}
		lat := minLat + (float64(i)+0.5)*stepLat
		for j := 0; j < lonSteps; j++ {
			lon := minLon + (float64(j)+0.5)*stepLon
			
			if isPointInside(lat, lon, areas) {
				gridPoints = append(gridPoints, Point{Lat: lat, Lon: lon})
			}
		}
	}
	if methodName != "" {
		UpdateProgress(methodName+" (Grid)", latSteps, latSteps)
	}
	return gridPoints
}

// CountGridPoints returns the number of points that would be generated inside the areas with the given resolution.
func CountGridPoints(areas []Area, resolution float64) int {
	if len(areas) == 0 || resolution <= 0 {
		return 0
	}

	minLat, maxLat, minLon, maxLon := getBoundingBox(areas)
	stepLat, stepLon := getGridSteps(minLat, maxLat, minLon, maxLon, resolution)

	count := 0
	latSteps := int((maxLat-minLat)/stepLat) + 1
	lonSteps := int((maxLon-minLon)/stepLon) + 1

	for i := 0; i < latSteps; i++ {
		lat := minLat + (float64(i)+0.5)*stepLat
		for j := 0; j < lonSteps; j++ {
			lon := minLon + (float64(j)+0.5)*stepLon
			
			if isPointInside(lat, lon, areas) {
				count++
			}
		}
	}
	return count
}

func getBoundingBox(areas []Area) (minLat, maxLat, minLon, maxLon float64) {
	minLat, maxLat = math.MaxFloat64, -math.MaxFloat64
	minLon, maxLon = math.MaxFloat64, -math.MaxFloat64
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
	return
}

func getGridSteps(minLat, maxLat, minLon, maxLon, resolution float64) (stepLat, stepLon float64) {
	center := Point{Lat: (minLat + maxLat) / 2.0, Lon: (minLon + maxLon) / 2.0}
	pOffsetLat := Point{Lat: center.Lat + 0.1, Lon: center.Lon}
	pOffsetLon := Point{Lat: center.Lat, Lon: center.Lon + 0.1}
	
	mPerDegLat := center.DistanceTo(pOffsetLat) * 10.0
	mPerDegLon := center.DistanceTo(pOffsetLon) * 10.0
	
	stepLat = resolution / mPerDegLat
	stepLon = resolution / mPerDegLon
	return
}

func isPointInside(lat, lon float64, areas []Area) bool {
	inside := false
	for _, a := range areas {
		for _, part := range a.Parts {
			if IsPointInPolygon(lat, lon, part) {
				inside = !inside
			}
		}
	}
	return inside
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
	const rad = math.Pi / 180.0
	cosLat := math.Cos(v.Lat * rad)
	
	dLat := w.Lat - v.Lat
	dLon := (w.Lon - v.Lon) * cosLat
	
	pLat := p.Lat - v.Lat
	pLon := (p.Lon - v.Lon) * cosLat
	
	l2 := dLat*dLat + dLon*dLon
	if l2 == 0 {
		return p.DistanceTo(v)
	}
	
	t := (pLat*dLat + pLon*dLon) / l2
	t = math.Max(0, math.Min(1, t))
	
	projection := Point{
		Lat: v.Lat + t*dLat,
		Lon: v.Lon + t*(w.Lon-v.Lon),
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

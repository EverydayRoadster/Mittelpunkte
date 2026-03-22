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
	Points []Point
}

// CalculationMethod is the interface for different middle point calculation methods.
type CalculationMethod interface {
	Name() string
	Calculate(areas []Area) Point
}

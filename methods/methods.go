package methods

// Point represents a geographical point with elevation.
type Point struct {
	Lat       float64
	Lon       float64
	Elevation float64
	Method    string
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

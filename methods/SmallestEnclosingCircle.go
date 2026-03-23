package methods

import (
	"math/rand"
)

// SmallestEnclosingCircle calculates the center of the smallest circle
// that contains all boundary points of the areas.
// It uses Welzl's algorithm (randomized, O(N)).
type SmallestEnclosingCircle struct{}

func (m SmallestEnclosingCircle) Name() string { return "SmallestEnclosingCircle" }

func (m SmallestEnclosingCircle) Calculate(areas []Area) Point {
	var points []Point
	for _, a := range areas {
		points = append(points, a.Points...)
	}

	if len(points) == 0 {
		return Point{}
	}

	c := welzl(points)

	// Elevation is average of all points
	sumElev := 0.0
	for _, p := range points {
		sumElev += p.Elevation
	}

	return Point{
		Lat:       c.center.Lat,
		Lon:       c.center.Lon,
		Elevation: sumElev / float64(len(points)),
		Method:    m.Name(),
	}
}

type circle struct {
	center Point
	radius float64
}

func welzl(points []Point) circle {
	shuffled := make([]Point, len(points))
	copy(shuffled, points)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	c := circle{center: shuffled[0], radius: 0}

	for i := 1; i < len(shuffled); i++ {
		if shuffled[i].DistanceTo(c.center) > c.radius+1e-7 {
			c = welzlWithPoint(shuffled[:i], shuffled[i])
		}
	}

	return c
}

func welzlWithPoint(points []Point, q Point) circle {
	c := circleFromTwoPoints(points[0], q) // Base case with 2 points

	// For small i, we might need a 1-point base case
	if len(points) == 0 {
		return circle{center: q, radius: 0}
	}
	
	c = circleFromTwoPoints(points[0], q)

	for i := 0; i < len(points); i++ {
		if points[i].DistanceTo(c.center) > c.radius+1e-7 {
			c = welzlWithTwoPoints(points[:i], points[i], q)
		}
	}
	return c
}

func welzlWithTwoPoints(points []Point, q1, q2 Point) circle {
	c := circleFromTwoPoints(q1, q2)

	for i := 0; i < len(points); i++ {
		if points[i].DistanceTo(c.center) > c.radius+1e-7 {
			c = circleFromThreePoints(q1, q2, points[i])
		}
	}
	return c
}


func circleFromTwoPoints(p1, p2 Point) circle {
	center := Point{
		Lat: (p1.Lat + p2.Lat) / 2,
		Lon: (p1.Lon + p2.Lon) / 2,
	}
	return circle{center: center, radius: p1.DistanceTo(center)}
}

func circleFromThreePoints(p1, p2, p3 Point) circle {
	// Simple circumcircle of 3 points in 2D (Lat/Lon)
	// This is an approximation for small geographic areas.
	x1, y1 := p1.Lon, p1.Lat
	x2, y2 := p2.Lon, p2.Lat
	x3, y3 := p3.Lon, p3.Lat

	D := 2 * (x1*(y2-y3) + x2*(y3-y1) + x3*(y1-y2))
	if D == 0 {
		// Points are collinear, find the two furthest apart
		d12 := p1.DistanceTo(p2)
		d13 := p1.DistanceTo(p3)
		d23 := p2.DistanceTo(p3)
		if d12 >= d13 && d12 >= d23 {
			return circleFromTwoPoints(p1, p2)
		}
		if d13 >= d12 && d13 >= d23 {
			return circleFromTwoPoints(p1, p3)
		}
		return circleFromTwoPoints(p2, p3)
	}

	Ux := ((x1*x1+y1*y1)*(y2-y3) + (x2*x2+y2*y2)*(y3-y1) + (x3*x3+y3*y3)*(y1-y2)) / D
	Uy := ((x1*x1+y1*y1)*(x3-x2) + (x2*x2+y2*y2)*(x1-x3) + (x3*x3+y3*y3)*(x2-x1)) / D

	center := Point{Lat: Uy, Lon: Ux}
	return circle{center: center, radius: p1.DistanceTo(center)}
}

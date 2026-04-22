package methods

import (
	"fmt"
	"math"
	"math/rand"
)

// SmallestEnclosingCircle calculates the center of the smallest circle
// that contains all boundary points of the areas.
// It uses Welzl's algorithm (randomized, O(N)).
type SmallestEnclosingCircle struct{}

func (m SmallestEnclosingCircle) Name() string { return "SmallestEnclosingCircle" }

func (m SmallestEnclosingCircle) Calculate(areas []Area) []Point {
	var points []Point
	for _, a := range areas {
		for _, part := range a.Parts {
			points = append(points, part...)
		}
	}

	if len(points) == 0 {
		return nil
	}

	c := m.calculateSEC(points)

	// Elevation is average of all points
	sumElev := 0.0
	for _, p := range points {
		sumElev += p.Elevation
	}

	res := c.center
	res.Elevation = sumElev / float64(len(points))
	res.Method = m.Name()
	return []Point{res}
}

func (m SmallestEnclosingCircle) SVG(areas []Area, points []Point, t SVGTransformer) string {
	if len(points) == 0 {
		return ""
	}
	
	var allBoundaryPoints []Point
	for _, a := range areas {
		for _, part := range a.Parts {
			allBoundaryPoints = append(allBoundaryPoints, part...)
		}
	}
	if len(allBoundaryPoints) == 0 {
		return ""
	}

	c := m.calculateSEC(allBoundaryPoints)
	cx, cy := t.Project(c.center)
	rx := t.ProjectRadiusX(c.radius, c.center)
	ry := t.ProjectRadiusY(c.radius, c.center)

	return fmt.Sprintf(`<ellipse cx="%.2f" cy="%.2f" rx="%.2f" ry="%.2f" fill="orange" fill-opacity="0.1" stroke="orange" stroke-width="2" />`, cx, cy, rx, ry)
}

type vec2 struct{ x, y float64 }
type cartCircle struct {
	center vec2
	radSq  float64
}

func (m SmallestEnclosingCircle) calculateSEC(points []Point) circle {
	if len(points) == 0 {
		return circle{}
	}

	// Reference point for local projection (use the average to minimize distortion)
	var avgLat, avgLon float64
	for i, p := range points {
		if i%1000 == 0 {
			UpdateProgress(m.Name()+" (Ref)", i, len(points))
		}
		avgLat += p.Lat
		avgLon += p.Lon
	}
	UpdateProgress(m.Name()+" (Ref)", len(points), len(points))
	ref := Point{Lat: avgLat / float64(len(points)), Lon: avgLon / float64(len(points))}

	const R = 6371000.0
	const rad = math.Pi / 180.0
	toLocal := func(p Point) vec2 {
		y := (p.Lat - ref.Lat) * rad * R
		x := (p.Lon - ref.Lon) * rad * R * math.Cos(ref.Lat*rad)
		return vec2{x, y}
	}
	fromLocal := func(v vec2) Point {
		lat := ref.Lat + (v.y / R / rad)
		lon := ref.Lon + (v.x / R / rad / math.Cos(ref.Lat*rad))
		return Point{Lat: lat, Lon: lon}
	}

	localPoints := make([]vec2, len(points))
	for i, p := range points {
		if i%1000 == 0 {
			UpdateProgress(m.Name()+" (Init)", i, len(points))
		}
		localPoints[i] = toLocal(p)
	}
	UpdateProgress(m.Name()+" (Init)", len(points), len(points))

	// Shuffle for O(N) performance
	shuffled := make([]vec2, len(localPoints))
	copy(shuffled, localPoints)
	rng := rand.New(rand.NewSource(42))
	rng.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	var welzl func(int, []vec2) cartCircle
	welzl = func(n int, bnd []vec2) cartCircle {
		if n == 0 || len(bnd) == 3 {
			return minidiskWithBoundary(bnd)
		}

		p := shuffled[n-1]
		D := welzl(n-1, bnd)

		if distSq(p, D.center) <= D.radSq+1e-7 {
			return D
		}

		// Point p must be on the boundary
		newBnd := make([]vec2, len(bnd)+1)
		copy(newBnd, bnd)
		newBnd[len(bnd)] = p
		return welzl(n-1, newBnd)
	}

	res := welzl(len(shuffled), nil)

	return circle{
		center: fromLocal(res.center),
		radius: math.Sqrt(res.radSq),
	}
}

func distSq(v1, v2 vec2) float64 {
	dx, dy := v1.x-v2.x, v1.y-v2.y
	return dx*dx + dy*dy
}

func minidiskWithBoundary(bnd []vec2) cartCircle {
	switch len(bnd) {
	case 0:
		return cartCircle{vec2{0, 0}, 0}
	case 1:
		return cartCircle{bnd[0], 0}
	case 2:
		p1, p2 := bnd[0], bnd[1]
		center := vec2{(p1.x + p2.x) / 2, (p1.y + p2.y) / 2}
		return cartCircle{center, distSq(p1, center)}
	case 3:
		return circumcircle(bnd[0], bnd[1], bnd[2])
	}
	return cartCircle{}
}

func circumcircle(p1, p2, p3 vec2) cartCircle {
	x1, y1 := p1.x, p1.y
	x2, y2 := p2.x, p2.y
	x3, y3 := p3.x, p3.y
	D := 2 * (x1*(y2-y3) + x2*(y3-y1) + x3*(y1-y2))
	if math.Abs(D) < 1e-7 {
		// Collinear points, SEC is defined by the two furthest apart
		d12 := distSq(p1, p2)
		d13 := distSq(p1, p3)
		d23 := distSq(p2, p3)
		if d12 >= d13 && d12 >= d23 {
			center := vec2{(p1.x + p2.x) / 2, (p1.y + p2.y) / 2}
			return cartCircle{center, distSq(p1, center)}
		}
		if d13 >= d12 && d13 >= d23 {
			center := vec2{(p1.x + p3.x) / 2, (p1.y + p3.y) / 2}
			return cartCircle{center, distSq(p1, center)}
		}
		center := vec2{(p2.x + p3.x) / 2, (p2.y + p3.y) / 2}
		return cartCircle{center, distSq(p2, center)}
	}
	ux := ((x1*x1+y1*y1)*(y2-y3) + (x2*x2+y2*y2)*(y3-y1) + (x3*x3+y3*y3)*(y1-y2)) / D
	uy := ((x1*x1+y1*y1)*(x3-x2) + (x2*x2+y2*y2)*(x1-x3) + (x3*x3+y3*y3)*(x2-x1)) / D
	center := vec2{ux, uy}
	return cartCircle{center, distSq(p1, center)}
}

type circle struct {
	center Point
	radius float64
}

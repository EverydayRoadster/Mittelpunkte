package methods

import (
	"container/heap"
	"fmt"
	"math"
)

// LargestInnerCircle calculates the center of the largest circle
// that can be inscribed within the areas.
// It uses a quadtree-based approach (similar to Mapbox's polylabel).
type LargestInnerCircle struct{}

func (m LargestInnerCircle) Name() string { return "LargestInnerCircle" }

func (m LargestInnerCircle) Calculate(areas []Area) Point {
	minLat, maxLat, minLon, maxLon := getBoundingBox(areas)

	width := maxLon - minLon
	height := maxLat - minLat
	cellSize := math.Max(width, height)
	if cellSize == 0 {
		return Point{}
	}

	// Initial best point: centroid of the bounding box if it's inside, 
	// otherwise just any point inside.
	bestPoint := Point{Lat: minLat + height/2, Lon: minLon + width/2}
	if !isPointInside(bestPoint.Lat, bestPoint.Lon, areas) {
		// Try to find a point inside
		grid := GenerateGridPoints(areas, 1000, "") // Coarse grid
		if len(grid) > 0 {
			bestPoint = grid[0]
		}
	}

	bestDist := 0.0
	if isPointInside(bestPoint.Lat, bestPoint.Lon, areas) {
		bestDist = DistanceToBoundary(bestPoint, areas)
	}

	pq := &cellQueue{}
	heap.Init(pq)

	// Initial cells
	for x := minLon; x < maxLon; x += cellSize {
		for y := minLat; y < maxLat; y += cellSize {
			c := &cell{
				lat:  y + cellSize/2,
				lon:  x + cellSize/2,
				size: cellSize,
			}
			pCenter := Point{Lat: c.lat, Lon: c.lon}
			c.dist = DistanceToBoundary(pCenter, areas)
			if !isPointInside(c.lat, c.lon, areas) {
				c.dist = -c.dist
			}
			
			// Calculate distance from center to corner in meters
			pCorner := Point{Lat: c.lat + cellSize/2, Lon: c.lon + cellSize/2}
			diagDist := pCenter.DistanceTo(pCorner)
			c.maxDist = c.dist + diagDist
			
			heap.Push(pq, c)
		}
	}

	iterations := 0
	for pq.Len() > 0 {
		c := heap.Pop(pq).(*cell)

		if c.dist > bestDist {
			bestPoint = Point{Lat: c.lat, Lon: c.lon}
			bestDist = c.dist
		}

		if c.maxDist-bestDist <= 0.1 { // Resolution limit: 10cm
			continue
		}

		// Split cell
		h := c.size / 2
		cells := []*cell{
			{lat: c.lat - h/2, lon: c.lon - h/2, size: h},
			{lat: c.lat - h/2, lon: c.lon + h/2, size: h},
			{lat: c.lat + h/2, lon: c.lon - h/2, size: h},
			{lat: c.lat + h/2, lon: c.lon + h/2, size: h},
		}

		for _, nc := range cells {
			pCenter := Point{Lat: nc.lat, Lon: nc.lon}
			nc.dist = DistanceToBoundary(pCenter, areas)
			if !isPointInside(nc.lat, nc.lon, areas) {
				nc.dist = -nc.dist
			}
			
			pCorner := Point{Lat: nc.lat + h/2, Lon: nc.lon + h/2}
			diagDist := pCenter.DistanceTo(pCorner)
			nc.maxDist = nc.dist + diagDist
			
			if nc.maxDist > bestDist {
				heap.Push(pq, nc)
			}
		}

		iterations++
		if iterations%1000 == 0 {
			UpdateProgress(m.Name(), iterations, 10000) // Rough estimate
		}
		if iterations > 20000 { // Safety break
			break
		}
	}
	UpdateProgress(m.Name(), 1, 1)

	// Elevation is average of nearest boundary points? Or just average of all points?
	// For consistency with SEC, let's use average of all points if available, 
	// but SEC used boundary points.
	var sumElev float64
	var countElev int
	for _, a := range areas {
		for _, part := range a.Parts {
			for _, p := range part {
				sumElev += p.Elevation
				countElev++
			}
		}
	}

	res := bestPoint
	if countElev > 0 {
		res.Elevation = sumElev / float64(countElev)
	}
	res.Method = m.Name()
	return res
}

func (m LargestInnerCircle) SVG(areas []Area, p Point, t SVGTransformer) string {
	radius := DistanceToBoundary(p, areas)
	cx, cy := t.Project(p)
	rx := t.ProjectRadiusX(radius, p)
	ry := t.ProjectRadiusY(radius, p)

	return fmt.Sprintf(`<ellipse cx="%.2f" cy="%.2f" rx="%.2f" ry="%.2f" fill="blue" fill-opacity="0.1" stroke="blue" stroke-width="2" />`, cx, cy, rx, ry)
}

type cell struct {
	lat, lon float64
	size     float64
	dist     float64 // distance from center to boundary
	maxDist  float64 // upper bound on distance within cell
	index    int     // for heap
}

type cellQueue []*cell

func (pq cellQueue) Len() int           { return len(pq) }
func (pq cellQueue) Less(i, j int) bool { return pq[i].maxDist > pq[j].maxDist }
func (pq cellQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}
func (pq *cellQueue) Push(x interface{}) {
	n := len(*pq)
	c := x.(*cell)
	c.index = n
	*pq = append(*pq, c)
}
func (pq *cellQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	c := old[n-1]
	c.index = -1
	*pq = old[0 : n-1]
	return c
}

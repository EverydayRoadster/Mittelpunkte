package methods

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"
)

// ReliefCenterOfGravity calculates the point of balance on a 3D surface model.
type ReliefCenterOfGravity struct {
	Resolution float64 // Grid resolution in meters
}

func (m ReliefCenterOfGravity) Name() string { return "ReliefCenterOfGravity" }

func (m ReliefCenterOfGravity) Calculate(areas []Area) Point {
	if len(areas) == 0 {
		return Point{}
	}

	res := m.Resolution
	if res == 0 {
		res = 100.0 // Default
	}

	gridPoints := GenerateGridPoints(areas, res)
	if len(gridPoints) == 0 {
		return Point{}
	}

	fmt.Printf("Fetching elevation for %d points inside the area...\n", len(gridPoints))
	
	// Fetch elevations (batched)
	elevations, err := fetchElevations(gridPoints)
	if err != nil {
		fmt.Printf("Warning: Could not fetch elevations: %v. Falling back to 2D CenterOfGravity.\n", err)
		return CenterOfGravity{Resolution: res}.Calculate(areas)
	}

	// Calculate weighted center of gravity
	var sumLat, sumLon, sumWeight, sumElev float64
	
	minLat := math.MaxFloat64
	for _, a := range areas {
		for _, part := range a.Parts {
			for _, p := range part {
				if p.Lat < minLat { minLat = p.Lat }
			}
		}
	}
	
	// Map points to their elevations for easy lookup
	type key struct{ i, j int }
	elevMap := make(map[key]float64)
	
	// Re-estimate steps to re-index
	center := Point{Lat: gridPoints[0].Lat, Lon: gridPoints[0].Lon}
	pOffsetLat := Point{Lat: center.Lat + 0.1, Lon: center.Lon}
	pOffsetLon := Point{Lat: center.Lat, Lon: center.Lon + 0.1}
	mPerDegLat := center.DistanceTo(pOffsetLat) * 10.0
	mPerDegLon := center.DistanceTo(pOffsetLon) * 10.0
	stepLat := res / mPerDegLat
	stepLon := res / mPerDegLon

	for idx, p := range gridPoints {
		i := int((p.Lat - minLat) / stepLat + 0.5)
		j := int(p.Lon / stepLon + 0.5) // Simplified key
		elevMap[key{i, j}] = elevations[idx]
	}

	for idx, p := range gridPoints {
		i := int((p.Lat - minLat) / stepLat + 0.5)
		j := int(p.Lon / stepLon + 0.5)
		
		z := elevations[idx]
		
		var dzdx, dzdy float64
		if zNext, ok := elevMap[key{i + 1, j}]; ok {
			dzdx = (zNext - z) / res
		}
		if zNext, ok := elevMap[key{i, j + 1}]; ok {
			dzdy = (zNext - z) / res
		}
		
		weight := math.Sqrt(1 + dzdx*dzdx + dzdy*dzdy)
		
		sumLat += p.Lat * weight
		sumLon += p.Lon * weight
		sumElev += z * weight
		sumWeight += weight
	}

	return Point{
		Lat:       sumLat / sumWeight,
		Lon:       sumLon / sumWeight,
		Elevation: sumElev / sumWeight,
		Method:    m.Name(),
	}
}

func (m ReliefCenterOfGravity) SVG(areas []Area, p Point, t SVGTransformer) string {
	cx, cy := t.Project(p)
	// Draw a star shape
	return fmt.Sprintf(`<path d="M %f,%f l 2,-5 l 2,5 l 5,0 l -4,3 l 2,5 l -5,-3 l -5,3 l 2,-5 l -4,-3 z" fill="yellow" stroke="orange" stroke-width="1" />`,
		cx, cy)
}

// ... fetchElevations and openTopoResponse remain same ...
func fetchElevations(points []Point) ([]float64, error) {
	elevations := make([]float64, len(points))
	batchSize := 100
	
	for i := 0; i < len(points); i += batchSize {
		end := i + batchSize
		if end > len(points) {
			end = len(points)
		}
		
		batch := points[i:end]
		var locs []string
		for _, p := range batch {
			locs = append(locs, fmt.Sprintf("%.6f,%.6f", p.Lat, p.Lon))
		}
		
		url := fmt.Sprintf("https://api.opentopodata.org/v1/srtm30m?locations=%s", strings.Join(locs, "|"))
		
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		
		var data struct {
			Results []struct {
				Elevation float64 `json:"elevation"`
			} `json:"results"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()
		
		for j, res := range data.Results {
			elevations[i+j] = res.Elevation
		}
		
		if i+batchSize < len(points) {
			time.Sleep(1000 * time.Millisecond)
		}
	}
	
	return elevations, nil
}

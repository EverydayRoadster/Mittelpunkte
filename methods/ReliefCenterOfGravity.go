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
		res = 10.0 // Default to 10m
	}

	// 1. Determine bounding box
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

	// 2. Determine degree steps for the resolution
	// Use a point in the center to estimate
	center := Point{Lat: (minLat + maxLat) / 2.0, Lon: (minLon + maxLon) / 2.0}
	pOffsetLat := Point{Lat: center.Lat + 0.001, Lon: center.Lon}
	pOffsetLon := Point{Lat: center.Lat, Lon: center.Lon + 0.001}
	
	mPerDegLat := center.DistanceTo(pOffsetLat) * 1000.0
	mPerDegLon := center.DistanceTo(pOffsetLon) * 1000.0
	
	stepLat := res / mPerDegLat * 0.001
	stepLon := res / mPerDegLon * 0.001

	// 3. Generate grid points inside areas
	var gridPoints []Point
	latSteps := int((maxLat-minLat)/stepLat) + 1
	lonSteps := int((maxLon-minLon)/stepLon) + 1

	fmt.Printf("Generating %dx%d grid (approx %d points)...\n", latSteps, lonSteps, latSteps*lonSteps)

	for i := 0; i < latSteps; i++ {
		lat := minLat + float64(i)*stepLat
		for j := 0; j < lonSteps; j++ {
			lon := minLon + float64(j)*stepLon
			
			inside := false
			for _, a := range areas {
				if isPointInPolygon(lat, lon, a.Points) {
					inside = true
					break
				}
			}
			if inside {
				gridPoints = append(gridPoints, Point{Lat: lat, Lon: lon})
			}
		}
	}

	if len(gridPoints) == 0 {
		return Point{}
	}

	fmt.Printf("Fetching elevation for %d points inside the area...\n", len(gridPoints))
	
	// 4. Fetch elevations (batched)
	elevations, err := fetchElevations(gridPoints)
	if err != nil {
		fmt.Printf("Warning: Could not fetch elevations: %v. Falling back to 2D CenterOfGravity.\n", err)
		return CenterOfGravity{}.Calculate(areas)
	}

	// 5. Calculate weighted center of gravity
	// Weight is the surface area. For a simple approximation on a grid,
	// we use w = sqrt(1 + dz/dx^2 + dz/dy^2).
	// To simplify, we'll use a local gradient approximation or just weight by elevation
	// if we assume constant density of the "shell".
	// Actually, a more physical "balance point" for a 3D shell uses the surface area.
	
	var sumLat, sumLon, sumWeight, sumElev float64
	
	// Map points to their elevations for easy lookup
	type key struct{ i, j int }
	elevMap := make(map[key]float64)
	// We need to re-index gridPoints to i, j to find neighbors
	for idx, p := range gridPoints {
		i := int((p.Lat - minLat) / stepLat + 0.5)
		j := int((p.Lon - minLon) / stepLon + 0.5)
		elevMap[key{i, j}] = elevations[idx]
	}

	for idx, p := range gridPoints {
		i := int((p.Lat - minLat) / stepLat + 0.5)
		j := int((p.Lon - minLon) / stepLon + 0.5)
		
		z := elevations[idx]
		
		// Estimate gradients
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

type openTopoResponse struct {
	Results []struct {
		Elevation float64 `json:"elevation"`
	} `json:"results"`
}

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
		
		var data openTopoResponse
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()
		
		for j, res := range data.Results {
			elevations[i+j] = res.Elevation
		}
		
		// Respect rate limits of public API
		if i+batchSize < len(points) {
			time.Sleep(1000 * time.Millisecond)
		}
	}
	
	return elevations, nil
}

package methods

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const cacheFile = "cache/elevation.json"

var elevationCache map[string]float64

func init() {
	elevationCache = make(map[string]float64)
	loadCache()
}

func loadCache() {
	info, err := os.Stat(cacheFile)
	if err != nil {
		return
	}

	file, err := os.Open(cacheFile)
	if err != nil {
		return
	}
	defer file.Close()

	if info.Size() > 1024*1024 { // If > 1MB, show progress
		fmt.Printf("Loading elevation cache (%d MB)...\n", info.Size()/(1024*1024))
		
		// For maps, json.Unmarshal is usually faster than streaming if memory allows,
		// but since we want progress, we'll just indicate we're working.
		// A full streaming progress bar for a single JSON object (the map) is complex
		// without a custom parser. Let's at least provide a start/end message.
		start := time.Now()
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&elevationCache); err != nil {
			fmt.Printf("Warning: Failed to load cache: %v\n", err)
		}
		fmt.Printf("Cache loaded in %v (%d points)\n", time.Since(start).Round(time.Millisecond), len(elevationCache))
	} else {
		decoder := json.NewDecoder(file)
		decoder.Decode(&elevationCache)
	}
}

func saveCache() {
	os.MkdirAll(filepath.Dir(cacheFile), 0755)
	data, _ := json.MarshalIndent(elevationCache, "", "  ")
	os.WriteFile(cacheFile, data, 0644)
}

func getCacheKey(p Point) string {
	return fmt.Sprintf("%.6f,%.6f", p.Lat, p.Lon)
}

// FetchElevations fetches elevations for a list of points, using a local cache.
func FetchElevations(points []Point, methodName string) ([]float64, error) {
	elevations := make([]float64, len(points))
	var missingIndices []int
	var missingLocs []string

	for i, p := range points {
		key := getCacheKey(p)
		if elev, ok := elevationCache[key]; ok {
			elevations[i] = elev
		} else {
			missingIndices = append(missingIndices, i)
			missingLocs = append(missingLocs, key)
		}
	}

	if len(missingLocs) == 0 {
		return elevations, nil
	}

	// Fetch missing in batches
	batchSize := 100
	cacheUpdated := false
	for i := 0; i < len(missingLocs); i += batchSize {
		UpdateProgress(methodName+" (API)", i, len(missingLocs))
		end := i + batchSize
		if end > len(missingLocs) {
			end = len(missingLocs)
		}

		batch := missingLocs[i:end]
		url := fmt.Sprintf("https://api.opentopodata.org/v1/srtm30m?locations=%s", strings.Join(batch, "|"))

		resp, err := http.Get(url)
		if err != nil {
			if cacheUpdated {
				saveCache()
			}
			return nil, err
		}

		var data struct {
			Results []struct {
				Elevation float64 `json:"elevation"`
			} `json:"results"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			resp.Body.Close()
			if cacheUpdated {
				saveCache()
			}
			return nil, err
		}
		resp.Body.Close()

		for j, res := range data.Results {
			idx := missingIndices[i+j]
			elevations[idx] = res.Elevation
			elevationCache[missingLocs[i+j]] = res.Elevation
			cacheUpdated = true
		}

		if i+batchSize < len(missingLocs) {
			time.Sleep(1000 * time.Millisecond)
		}
	}
	UpdateProgress(methodName+" (API)", len(missingLocs), len(missingLocs))

	if cacheUpdated {
		saveCache()
	}

	return elevations, nil
}

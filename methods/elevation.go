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
		
		start := time.Now()
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&elevationCache); err != nil {
			fmt.Printf("Warning: Failed to load cache: %v\n", err)
		}
		normalizeCache()
		fmt.Printf("Cache loaded in %v (%d points)\n", time.Since(start).Round(time.Millisecond), len(elevationCache))
	} else {
		decoder := json.NewDecoder(file)
		decoder.Decode(&elevationCache)
		normalizeCache()
	}
}

func normalizeCache() {
	if len(elevationCache) == 0 {
		return
	}
	newCache := make(map[string]float64)
	for key, elev := range elevationCache {
		var lat, lon float64
		if _, err := fmt.Sscanf(key, "%f,%f", &lat, &lon); err == nil {
			newKey := fmt.Sprintf("%.4f,%.4f", lat, lon)
			if _, ok := newCache[newKey]; !ok {
				newCache[newKey] = elev
			}
		}
	}
	elevationCache = newCache
}

func saveCache() {
	os.MkdirAll(filepath.Dir(cacheFile), 0755)
	data, _ := json.Marshal(elevationCache)
	os.WriteFile(cacheFile, data, 0644)
}

func getCacheKey(p Point) string {
	return fmt.Sprintf("%.4f,%.4f", p.Lat, p.Lon)
}

// FetchElevations fetches elevations for a list of points, using a local cache.
func FetchElevations(points []Point, methodName string) ([]float64, error) {
	elevations := make([]float64, len(points))
	
	// key -> list of indices in the original points slice
	missing := make(map[string][]int)
	var missingKeys []string

	for i, p := range points {
		key := getCacheKey(p)
		if elev, ok := elevationCache[key]; ok {
			elevations[i] = elev
		} else {
			if _, exists := missing[key]; !exists {
				missingKeys = append(missingKeys, key)
			}
			missing[key] = append(missing[key], i)
		}
	}

	if len(missingKeys) == 0 {
		return elevations, nil
	}

	// Fetch missing in batches
	batchSize := 100
	cacheUpdated := false
	for i := 0; i < len(missingKeys); i += batchSize {
		UpdateProgress(methodName+" (API)", i, len(missingKeys))
		end := i + batchSize
		if end > len(missingKeys) {
			end = len(missingKeys)
		}

		batch := missingKeys[i:end]
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

		if len(data.Results) != len(batch) {
			// This shouldn't happen with opentopodata unless there's an error
			fmt.Printf("\nWarning: API returned %d results for %d locations\n", len(data.Results), len(batch))
		}

		for j, res := range data.Results {
			if j >= len(batch) {
				break
			}
			key := batch[j]
			elevationCache[key] = res.Elevation
			for _, idx := range missing[key] {
				elevations[idx] = res.Elevation
			}
			cacheUpdated = true
		}

		if i+batchSize < len(missingKeys) {
			time.Sleep(1000 * time.Millisecond)
		}
	}
	UpdateProgress(methodName+" (API)", len(missingKeys), len(missingKeys))

	if cacheUpdated {
		saveCache()
	}

	return elevations, nil
}

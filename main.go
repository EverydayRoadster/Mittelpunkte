package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jonas-p/go-shp"
	"github.com/paulmach/go.geojson"
	"github.com/tkrajina/gpxgo/gpx"
)

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
	Calculate(points []Point) Point
}

// BoundingBoxCenter calculates the average between maximum and minimum lat and lon.
type BoundingBoxCenter struct{}

func (m BoundingBoxCenter) Name() string { return "BoundingBoxCenter" }

func (m BoundingBoxCenter) Calculate(points []Point) Point {
	if len(points) == 0 {
		return Point{}
	}
	minLat, maxLat := points[0].Lat, points[0].Lat
	minLon, maxLon := points[0].Lon, points[0].Lon
	sumElev := 0.0

	for _, p := range points {
		if p.Lat < minLat {
			minLat = p.Lat
		}
		if p.Lat > maxLat {
			maxLat = p.Lat
		}
		if p.Lon < minLon {
			minLon = p.Lon
		}
		if p.Lon > maxLon {
			maxLon = p.Lon
		}
		sumElev += p.Elevation
	}

	return Point{
		Lat:       (minLat + maxLat) / 2.0,
		Lon:       (minLon + maxLon) / 2.0,
		Elevation: sumElev / float64(len(points)),
		Method:    m.Name(),
	}
}

// IntersectionOfOutermost calculates the intersection of lines between the outermost points of lat and lon.
type IntersectionOfOutermost struct{}

func (m IntersectionOfOutermost) Name() string { return "IntersectionOfOutermost" }

func (m IntersectionOfOutermost) Calculate(points []Point) Point {
	if len(points) == 0 {
		return Point{}
	}

	var pMinLat, pMaxLat, pMinLon, pMaxLon Point
	pMinLat = points[0]
	pMaxLat = points[0]
	pMinLon = points[0]
	pMaxLon = points[0]
	sumElev := 0.0

	for _, p := range points {
		if p.Lat < pMinLat.Lat {
			pMinLat = p
		}
		if p.Lat > pMaxLat.Lat {
			pMaxLat = p
		}
		if p.Lon < pMinLon.Lon {
			pMinLon = p
		}
		if p.Lon > pMaxLon.Lon {
			pMaxLon = p
		}
		sumElev += p.Elevation
	}

	// Line 1: pMinLat to pMaxLat
	// Line 2: pMinLon to pMaxLon
	x1, y1 := pMinLat.Lon, pMinLat.Lat
	x2, y2 := pMaxLat.Lon, pMaxLat.Lat
	x3, y3 := pMinLon.Lon, pMinLon.Lat
	x4, y4 := pMaxLon.Lon, pMaxLon.Lat

	denom := (x1-x2)*(y3-y4) - (y1-y2)*(x3-x4)
	if denom == 0 {
		// Parallel lines, fallback to bounding box center for this case
		return BoundingBoxCenter{}.Calculate(points)
	}

	intersectX := ((x1*y2-y1*x2)*(x3-x4) - (x1-x2)*(x3*y4-y3*x4)) / denom
	intersectY := ((x1*y2-y1*x2)*(y3-y4) - (y1-y2)*(x3*y4-y3*x4)) / denom

	return Point{
		Lat:       intersectY,
		Lon:       intersectX,
		Elevation: sumElev / float64(len(points)),
		Method:    m.Name(),
	}
}

func main() {
	inputDir := flag.String("input", "", "Directory containing ESRI Shapefiles")
	outputDir := flag.String("output", ".", "Output directory")
	filterNames := flag.String("filter", "", "Comma-separated list of area names to include")
	flag.Parse()

	if *inputDir == "" {
		if flag.NArg() > 0 {
			*inputDir = flag.Arg(0)
		} else {
			log.Fatal("Input directory is required")
		}
	}

	absInputDir, err := filepath.Abs(*inputDir)
	if err != nil {
		log.Fatalf("Invalid input directory: %v", err)
	}

	info, err := os.Stat(absInputDir)
	if err != nil || !info.IsDir() {
		log.Fatalf("Input path %s is not a directory", absInputDir)
	}

	dirName := filepath.Base(absInputDir)
	finalOutputDir := filepath.Join(*outputDir, dirName)

	if err := os.MkdirAll(finalOutputDir, 0755); err != nil {
		log.Fatalf("Could not create output directory: %v", err)
	}

	files, err := os.ReadDir(absInputDir)
	if err != nil {
		log.Fatalf("Could not read input directory: %v", err)
	}

	var allAreas []Area
	for _, file := range files {
		if strings.ToLower(filepath.Ext(file.Name())) == ".shp" {
			shpPath := filepath.Join(absInputDir, file.Name())
			areas, err := readShapefile(shpPath)
			if err != nil {
				log.Printf("Error reading %s: %v", file.Name(), err)
				continue
			}
			allAreas = append(allAreas, areas...)
		}
	}

	if len(allAreas) == 0 {
		log.Fatal("No areas found in input directory")
	}

	// Filter areas if requested
	var selectedAreas []Area
	if *filterNames != "" {
		filters := strings.Split(*filterNames, ",")
		for i := range filters {
			filters[i] = strings.TrimSpace(filters[i])
		}
		for _, area := range allAreas {
			for _, f := range filters {
				if strings.EqualFold(area.Name, f) {
					selectedAreas = append(selectedAreas, area)
					break
				}
			}
		}
	} else {
		selectedAreas = allAreas
		// If no filter but many areas, print names to help user
		if len(allAreas) > 1 {
			fmt.Println("Available areas:")
			for _, a := range allAreas {
				fmt.Printf(" - %s\n", a.Name)
			}
			fmt.Println("Use -filter to select specific areas.")
		}
	}

	if len(selectedAreas) == 0 {
		log.Fatal("No areas selected")
	}

	// Save converted data as tracks
	saveGeoJSON(filepath.Join(finalOutputDir, "areas.geojson"), selectedAreas)
	saveGPX(filepath.Join(finalOutputDir, "areas.gpx"), selectedAreas)

	// Collect all points for middle point calculation
	var allPoints []Point
	for _, area := range selectedAreas {
		allPoints = append(allPoints, area.Points...)
	}

	// Calculate middle points
	methods := []CalculationMethod{
		BoundingBoxCenter{},
		IntersectionOfOutermost{},
	}

	var middlePoints []Point
	for _, method := range methods {
		mp := method.Calculate(allPoints)
		middlePoints = append(middlePoints, mp)
		fmt.Printf("Method %s: Lat %.6f, Lon %.6f\n", mp.Method, mp.Lat, mp.Lon)
	}

	// Save middle points
	saveMiddlePointsGeoJSON(filepath.Join(finalOutputDir, "middle_points.geojson"), middlePoints)
	saveMiddlePointsGPX(filepath.Join(finalOutputDir, "middle_points.gpx"), middlePoints)
}

func readShapefile(path string) ([]Area, error) {
	s, err := shp.Open(path)
	if err != nil {
		return nil, err
	}
	defer s.Close()

	// Check for projection
	prjPath := strings.TrimSuffix(path, ".shp") + ".prj"
	isSwiss := false
	if prjData, err := os.ReadFile(prjPath); err == nil {
		if strings.Contains(string(prjData), "CH1903+_LV95") || strings.Contains(string(prjData), "EPSG:2056") {
			isSwiss = true
		}
	}

	// Identify name attribute column
	nameIdx := -1
	fields := s.Fields()
	for i, f := range fields {
		fieldName := strings.ToUpper(f.String())
		if fieldName == "NAME" || fieldName == "GEMEINDE" || fieldName == "BEZIRKSNAM" {
			nameIdx = i
			break
		}
	}

	var areas []Area
	for s.Next() {
		idx, shape := s.Shape()
		
		name := fmt.Sprintf("Area %d", idx)
		if nameIdx != -1 {
			val := s.ReadAttribute(idx, nameIdx)
			if strings.TrimSpace(val) != "" {
				name = strings.TrimSpace(val)
			}
		}

		var points []Point
		switch shpObj := shape.(type) {
		case *shp.Point:
			points = append(points, convertPoint(shpObj.X, shpObj.Y, 0, isSwiss))
		case *shp.PointZ:
			points = append(points, convertPoint(shpObj.X, shpObj.Y, shpObj.Z, isSwiss))
		case *shp.PolyLine:
			for _, p := range shpObj.Points {
				points = append(points, convertPoint(p.X, p.Y, 0, isSwiss))
			}
		case *shp.PolyLineZ:
			for i, p := range shpObj.Points {
				points = append(points, convertPoint(p.X, p.Y, shpObj.ZArray[i], isSwiss))
			}
		case *shp.Polygon:
			for _, p := range shpObj.Points {
				points = append(points, convertPoint(p.X, p.Y, 0, isSwiss))
			}
		case *shp.PolygonZ:
			for i, p := range shpObj.Points {
				points = append(points, convertPoint(p.X, p.Y, shpObj.ZArray[i], isSwiss))
			}
		}

		if len(points) > 0 {
			areas = append(areas, Area{Name: name, Points: points})
		}
	}

	return areas, nil
}

func convertPoint(x, y, z float64, isSwiss bool) Point {
	if isSwiss {
		y_aux := (x - 2600000) / 1000000
		x_aux := (y - 1200000) / 1000000

		lonUnit := 2.6779094 +
			4.728982*y_aux +
			0.791484*y_aux*x_aux +
			0.1306*y_aux*x_aux*x_aux -
			0.0436*y_aux*y_aux*y_aux

		latUnit := 16.9023892 +
			3.238272*x_aux -
			0.270978*y_aux*y_aux -
			0.002528*x_aux*x_aux -
			0.0447*y_aux*y_aux*x_aux -
			0.0140*x_aux*x_aux*x_aux

		lon := lonUnit * 100 / 36
		lat := latUnit * 100 / 36

		return Point{Lat: lat, Lon: lon, Elevation: z}
	}
	return Point{Lat: y, Lon: x, Elevation: z}
}

func saveGeoJSON(path string, areas []Area) {
	fc := geojson.NewFeatureCollection()
	for _, a := range areas {
		coords := make([][]float64, len(a.Points))
		for i, p := range a.Points {
			coords[i] = []float64{p.Lon, p.Lat, p.Elevation}
		}
		f := geojson.NewLineStringFeature(coords)
		f.SetProperty("name", a.Name)
		fc.AddFeature(f)
	}
	data, _ := fc.MarshalJSON()
	os.WriteFile(path, data, 0644)
}

func saveGPX(path string, areas []Area) {
	g := &gpx.GPX{}
	for _, a := range areas {
		trk := gpx.GPXTrack{Name: a.Name}
		seg := gpx.GPXTrackSegment{}
		for _, p := range a.Points {
			seg.Points = append(seg.Points, gpx.GPXPoint{
				Point: gpx.Point{
					Latitude:  p.Lat,
					Longitude: p.Lon,
					Elevation: *gpx.NewNullableFloat64(p.Elevation),
				},
			})
		}
		trk.Segments = append(trk.Segments, seg)
		g.Tracks = append(g.Tracks, trk)
	}
	xml, _ := gpx.ToXml(g, gpx.ToXmlParams{Indent: true})
	os.WriteFile(path, xml, 0644)
}

func saveMiddlePointsGeoJSON(path string, points []Point) {
	fc := geojson.NewFeatureCollection()
	for _, p := range points {
		f := geojson.NewPointFeature([]float64{p.Lon, p.Lat, p.Elevation})
		f.SetProperty("method", p.Method)
		f.SetProperty("name", p.Method)
		fc.AddFeature(f)
	}
	data, _ := fc.MarshalJSON()
	os.WriteFile(path, data, 0644)
}

func saveMiddlePointsGPX(path string, points []Point) {
	g := &gpx.GPX{}
	for _, p := range points {
		wpt := gpx.GPXPoint{
			Point: gpx.Point{
				Latitude:  p.Lat,
				Longitude: p.Lon,
				Elevation: *gpx.NewNullableFloat64(p.Elevation),
			},
		}
		wpt.Name = p.Method
		g.Waypoints = append(g.Waypoints, wpt)
	}
	xml, _ := gpx.ToXml(g, gpx.ToXmlParams{Indent: true})
	os.WriteFile(path, xml, 0644)
}

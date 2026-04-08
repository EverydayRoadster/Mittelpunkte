package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/EverydayRoadster/Mittelpunkte/methods"
	"github.com/im7mortal/UTM"
	"github.com/jonas-p/go-shp"
	geojson "github.com/paulmach/go.geojson"
	"github.com/tkrajina/gpxgo/gpx"
)

type Projection int

const (
	ProjWGS84 Projection = iota
	ProjSwiss
	ProjUTM32N
)

func main() {
	inputDir := flag.String("input", "", "Directory containing ESRI Shapefiles")
	outputDir := flag.String("output", ".", "Output directory")
	filterNames := flag.String("filter", "", "Comma-separated list of area names to include")
	resolution := flag.Float64("resolution", 30.0, "Resolution in meters for grid-based methods (min 30m, auto-increases for elevation methods if > 16384 points)")
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
	filterDir := "all"
	if *filterNames != "" {
		// Clean up filter names for directory use
		filters := strings.Split(*filterNames, ",")
		for i := range filters {
			filters[i] = strings.TrimSpace(filters[i])
			filters[i] = strings.ReplaceAll(filters[i], " ", "_")
		}
		filterDir = strings.Join(filters, "_")
	}
	finalOutputDir := filepath.Join(*outputDir, dirName, filterDir)

	files, err := os.ReadDir(absInputDir)
	if err != nil {
		log.Fatalf("Could not read input directory: %v", err)
	}

	// Group areas by level
	var allAreas []methods.Area
	var levels []string
	areasByLevel := make(map[string][]methods.Area)

	for _, file := range files {
		if strings.ToLower(filepath.Ext(file.Name())) == ".shp" {
			shpPath := filepath.Join(absInputDir, file.Name())
			areas, err := readShapefile(shpPath)
			if err != nil {
				log.Printf("Error reading %s: %v", file.Name(), err)
				continue
			}

			if len(areas) > 0 {
				level := areas[0].Level
				if _, ok := areasByLevel[level]; !ok {
					levels = append(levels, level)
				}
				areasByLevel[level] = append(areasByLevel[level], areas...)
				allAreas = append(allAreas, areas...)
			}
		}
	}

	if len(allAreas) == 0 {
		log.Fatal("No areas found in input directory")
	}

	// Filter areas if requested
	var selectedAreas []methods.Area
	if *filterNames != "" {
		filters := strings.Split(*filterNames, ",")
		for i := range filters {
			filters[i] = strings.TrimSpace(filters[i])
		}
		for _, area := range allAreas {
			for _, f := range filters {
				if strings.EqualFold(area.Name, f) || strings.EqualFold(area.Level, f) {
					selectedAreas = append(selectedAreas, area)
					break
				}
			}
		}
	} else {
		// If no filter, print names and exit
		fmt.Println("Available areas by level (use -filter with name or level):")
		for _, level := range levels {
			areas := areasByLevel[level]
			fmt.Printf("\nLevel: %s (%d areas)\n", level, len(areas))
			for _, a := range areas {
				fmt.Printf(" - %s\n", a.Name)
			}
		}
		fmt.Println("\nUse -filter to select specific areas or an entire level for calculation.")
		os.Exit(0)
	}

	if len(selectedAreas) == 0 {
		log.Fatal("No areas selected")
	}

	for i := range selectedAreas {
		selectedAreas[i].PrecomputeBounds()
	}

	// Validate resolution: must be at least 30.0
	if *resolution < 30.0 {
		log.Fatalf("Error: Resolution %.1fm is too small. Minimum supported resolution is 30.0m.", *resolution)
	}

	// Check if resolution was explicitly provided
	isExplicitRes := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "resolution" {
			isExplicitRes = true
		}
	})

	// Calculate a separate resolution for elevation-based methods
	elevationResolution := *resolution
	if !isExplicitRes {
		minLat, maxLat, minLon, maxLon := methods.GetBoundingBox(selectedAreas)
		center := methods.Point{Lat: (minLat + maxLat) / 2.0, Lon: (minLon + maxLon) / 2.0}
		pOffsetLat := methods.Point{Lat: center.Lat + 0.1, Lon: center.Lon}
		pOffsetLon := methods.Point{Lat: center.Lat, Lon: center.Lon + 0.1}
		mPerDegLat := center.DistanceTo(pOffsetLat) * 10.0
		mPerDegLon := center.DistanceTo(pOffsetLon) * 10.0
		
		widthMeters := (maxLon - minLon) * mPerDegLon
		heightMeters := (maxLat - minLat) * mPerDegLat
		
		// Heuristic: assume 2/3 of the bounding box is occupied by the area.
		// We want (width/res * height/res) * (2/3) <= 16384
		// res^2 >= (width * height * 2/3) / 16384
		targetRes := math.Sqrt((widthMeters * heightMeters * 0.666) / 16384.0)
		
		if targetRes > elevationResolution {
			elevationResolution = targetRes
			fmt.Printf("Auto-adjusted elevation resolution to %.1fm (estimated) to limit data queries.\n", 
				elevationResolution)
		}
	} else {
		// Even if explicit, inform the user about the estimated point count for transparency
		minLat, maxLat, minLon, maxLon := methods.GetBoundingBox(selectedAreas)
		center := methods.Point{Lat: (minLat + maxLat) / 2.0, Lon: (minLon + maxLon) / 2.0}
		pOffsetLat := methods.Point{Lat: center.Lat + 0.1, Lon: center.Lon}
		pOffsetLon := methods.Point{Lat: center.Lat, Lon: center.Lon + 0.1}
		mPerDegLat := center.DistanceTo(pOffsetLat) * 10.0
		mPerDegLon := center.DistanceTo(pOffsetLon) * 10.0
		latSteps := int((maxLat-minLat) / (elevationResolution / mPerDegLat)) + 1
		lonSteps := int((maxLon-minLon) / (elevationResolution / mPerDegLon)) + 1
		estimatedCount := int(float64(latSteps*lonSteps) * 0.666)
		
		if estimatedCount > 16384 {
			fmt.Printf("Using explicit elevation resolution %.1fm (~%d estimated points). This may take some time.\n", 
				elevationResolution, estimatedCount)
		}
	}

	if err := os.MkdirAll(finalOutputDir, 0755); err != nil {
		log.Fatalf("Could not create output directory: %v", err)
	}

	// Save converted data as tracks
	saveGeoJSON(filepath.Join(finalOutputDir, "areas.geojson"), selectedAreas)
	saveGPX(filepath.Join(finalOutputDir, "areas.gpx"), selectedAreas)

	// Calculate middle points
	calcMethods := []methods.CalculationMethod{
		methods.BoundingBoxCenter{},
		methods.IntersectionOfOutermost{},
		methods.CenterOfGravity{Resolution: *resolution},
		methods.MinimalDistanceSum{},
		methods.RotatingBoundingBoxCenter{},
		methods.MinimalDistanceSumEqualSpaced{},
		methods.ReliefCenterOfGravity{Resolution: elevationResolution},
		methods.FermatPointF1{Resolution: *resolution},
		methods.SmallestEnclosingCircle{},
		methods.LargestInnerCircle{},
	}

	var middlePoints []methods.Point
	for _, method := range calcMethods {
		mp := method.Calculate(selectedAreas)
		middlePoints = append(middlePoints, mp)
		fmt.Printf("Method %s: Lat %.6f, Lon %.6f\n", mp.Method, mp.Lat, mp.Lon)
	}

	// Save middle points
	saveMiddlePointsGeoJSON(filepath.Join(finalOutputDir, "middle_points.geojson"), middlePoints)
	saveMiddlePointsGPX(filepath.Join(finalOutputDir, "middle_points.gpx"), middlePoints)

	// Save SVGs
	saveSVGs(finalOutputDir, selectedAreas, calcMethods, middlePoints)
}

func saveSVGs(dir string, areas []methods.Area, calcMethods []methods.CalculationMethod, results []methods.Point) {
	if len(areas) == 0 {
		return
	}

	// Calculate overall bounding box
	minLat, maxLat := math.MaxFloat64, -math.MaxFloat64
	minLon, maxLon := math.MaxFloat64, -math.MaxFloat64
	for _, a := range areas {
		for _, part := range a.Parts {
			for _, p := range part {
				if p.Lat < minLat { minLat = p.Lat }
				if p.Lat > maxLat { maxLat = p.Lat }
				if p.Lon < minLon { minLon = p.Lon }
				if p.Lon > maxLon { maxLon = p.Lon }
			}
		}
	}

	// Add 5% padding
	latPad := (maxLat - minLat) * 0.05
	lonPad := (maxLon - minLon) * 0.05
	if latPad == 0 { latPad = 0.01 }
	if lonPad == 0 { lonPad = 0.01 }

	// Account for meridian convergence (aspect ratio correction)
	avgLat := (minLat + maxLat) / 2.0
	cosLat := math.Cos(avgLat * math.Pi / 180.0)

	width := 800.0
	height := width * (maxLat - minLat + 2*latPad) / ((maxLon - minLon + 2*lonPad) * cosLat)
	if height > 1200 {
		height = 1200
		width = height * ((maxLon - minLon + 2*lonPad) * cosLat) / (maxLat - minLat + 2*latPad)
	}

	t := methods.SVGTransformer{
		MinLat: minLat - latPad, MaxLat: maxLat + latPad,
		MinLon: minLon - lonPad, MaxLon: maxLon + lonPad,
		Width: width, Height: height,
	}

	// Generate base polygon paths
	var basePaths []string
	for _, a := range areas {
		for _, part := range a.Parts {
			var points []string
			for _, p := range part {
				x, y := t.Project(p)
				points = append(points, fmt.Sprintf("%.2f,%.2f", x, y))
			}
			basePaths = append(basePaths, fmt.Sprintf(`<polygon points="%s" fill="#f0f0f0" stroke="#ccc" stroke-width="1" />`,
				strings.Join(points, " ")))
		}
	}
	baseSVG := strings.Join(basePaths, "\n")


	for i, method := range calcMethods {
		res := results[i]
		methodSVG := method.SVG(areas, res, t)

		// Final center point marker
		cx, cy := t.Project(res)
		markerSVG := fmt.Sprintf(`<circle cx="%.2f" cy="%.2f" r="4" fill="red" stroke="white" stroke-width="1" />`+
			`<text x="%.2f" y="%.2f" font-family="sans-serif" font-size="12" fill="black" dy="-10" text-anchor="middle">%s</text>`,
			cx, cy, cx, cy, method.Name())

		fullSVG := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<svg width="%.0f" height="%.0f" viewBox="0 0 %.0f %.0f" xmlns="http://www.w3.org/2000/svg">
	<rect width="100%%" height="100%%" fill="white" />
	%s
	%s
	%s
</svg>`, width, height, width, height, baseSVG, methodSVG, markerSVG)

		filename := filepath.Join(dir, fmt.Sprintf("%s.svg", method.Name()))
		os.WriteFile(filename, []byte(fullSVG), 0644)
	}
}

func readShapefile(path string) ([]methods.Area, error) {
	s, err := shp.Open(path)
	if err != nil {
		return nil, err
	}
	defer s.Close()

	// Check for projection
	prjPath := strings.TrimSuffix(path, ".shp") + ".prj"
	proj := ProjWGS84
	if prjData, err := os.ReadFile(prjPath); err == nil {
		prjStr := string(prjData)
		if strings.Contains(prjStr, "CH1903+_LV95") || strings.Contains(prjStr, "EPSG:2056") {
			proj = ProjSwiss
		} else if strings.Contains(prjStr, "ETRS_1989_UTM_Zone_32N") || strings.Contains(prjStr, "EPSG:25832") {
			proj = ProjUTM32N
		}
	}

	// Identify name attribute column with priority
	nameIdx := -1
	currentPriority := -1
	fields := s.Fields()
	for i, f := range fields {
		fieldName := strings.ToUpper(f.String())

		priority := -1
		switch fieldName {
		case "NAME", "GEMEINDE", "GEMEINDE_N", "GEN", "LAND_NAME", "LAND_N":
			priority = 100 // High priority
		case "BEZ", "BEZIRKSNAM", "KREIS_NAME", "REGION_NAM", "REGIERUN_1", "REGIERUN_N":
			priority = 50 // Medium priority
		case "KLASSE", "GML_ID":
			priority = 10 // Low priority
		}

		if priority > currentPriority {
			nameIdx = i
			currentPriority = priority
		}
	}

	// Base level name from file
	levelName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	levelName = strings.TrimPrefix(levelName, "v_at_")
	levelName = strings.Title(strings.ReplaceAll(levelName, "_", " "))
	if levelName == "Regierungebezirk" {
		levelName = "Regierungsbezirk"
	}

	var areas []methods.Area
	for s.Next() {
		idx, shape := s.Shape()

		// Default name is level if only 1 area, otherwise level + index
		name := levelName
		if s.AttributeCount() > 1 || idx > 0 {
			name = fmt.Sprintf("%s %d", levelName, idx)
		}

		if nameIdx != -1 {
			val := s.ReadAttribute(idx, nameIdx)
			trimmedVal := strings.TrimSpace(val)
			if trimmedVal != "" && trimmedVal != "**********" {
				name = trimmedVal
			}
		}

		var parts [][]methods.Point
		switch shpObj := shape.(type) {
		case *shp.Point:
			parts = append(parts, []methods.Point{convertPoint(shpObj.X, shpObj.Y, 0, proj)})
		case *shp.PointZ:
			parts = append(parts, []methods.Point{convertPoint(shpObj.X, shpObj.Y, shpObj.Z, proj)})
		case *shp.PolyLine:
			parts = extractParts(shpObj.Parts, shpObj.Points, nil, proj)
		case *shp.PolyLineZ:
			parts = extractParts(shpObj.Parts, shpObj.Points, shpObj.ZArray, proj)
		case *shp.Polygon:
			parts = extractParts(shpObj.Parts, shpObj.Points, nil, proj)
		case *shp.PolygonZ:
			parts = extractParts(shpObj.Parts, shpObj.Points, shpObj.ZArray, proj)
		}

		if len(parts) > 0 {
			areas = append(areas, methods.Area{Name: name, Level: levelName, Parts: parts})
		}
		}

		return areas, nil
		}

		func extractParts(partsIdx []int32, points []shp.Point, zArray []float64, proj Projection) [][]methods.Point {
		var parts [][]methods.Point
		for i := 0; i < len(partsIdx); i++ {
		start := partsIdx[i]
		end := int32(len(points))
		if i+1 < len(partsIdx) {
			end = partsIdx[i+1]
		}

		var partPoints []methods.Point
		for j := start; j < end; j++ {
			z := 0.0
			if zArray != nil {
				z = zArray[j]
			}
			partPoints = append(partPoints, convertPoint(points[j].X, points[j].Y, z, proj))
		}
		parts = append(parts, partPoints)
		}
		return parts
		}


func convertPoint(x, y, z float64, proj Projection) methods.Point {
	var lat, lon float64
	switch proj {
	case ProjSwiss:
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

		lon = lonUnit * 100 / 36
		lat = latUnit * 100 / 36
	case ProjUTM32N:
		var err error
		lat, lon, err = UTM.ToLatLon(x, y, 32, "N")
		if err != nil {
			log.Printf("UTM conversion error: %v", err)
		}
	default:
		lat, lon = y, x
	}
	return methods.Point{Lat: lat, Lon: lon, Elevation: z}
}

func saveGeoJSON(path string, areas []methods.Area) {
	fc := geojson.NewFeatureCollection()
	for _, a := range areas {
		if len(a.Parts) == 0 { continue }
		
		if len(a.Parts) == 1 {
			// Single polygon: [][][]float64 (outer ring + holes)
			ring := make([][]float64, len(a.Parts[0]))
			for i, p := range a.Parts[0] {
				ring[i] = []float64{p.Lon, p.Lat, p.Elevation}
			}
			f := geojson.NewPolygonFeature([][][]float64{ring})
			f.SetProperty("name", a.Name)
			fc.AddFeature(f)
		} else {
			// MultiPolygon: [][][]float64 (each element is a list of points representing a ring)
			// Wait, if it's MultiPolygon it should be list of polygons, and each polygon is list of rings.
			// Let's check the library. Usually MultiPolygon is [][][][]float64.
			// If the error says it wants [][][]float64, maybe it treats it as a list of outer rings only?
			
			multi := make([][][]float64, len(a.Parts))
			for i, part := range a.Parts {
				ring := make([][]float64, len(part))
				for j, p := range part {
					ring[j] = []float64{p.Lon, p.Lat, p.Elevation}
				}
				multi[i] = ring
			}
			f := geojson.NewMultiPolygonFeature(multi)
			f.SetProperty("name", a.Name)
			fc.AddFeature(f)
		}
	}
	data, _ := fc.MarshalJSON()
	os.WriteFile(path, data, 0644)
}

func saveGPX(path string, areas []methods.Area) {
	g := &gpx.GPX{}
	for _, a := range areas {
		trk := gpx.GPXTrack{Name: a.Name}
		for _, part := range a.Parts {
			seg := gpx.GPXTrackSegment{}
			for _, p := range part {
				seg.Points = append(seg.Points, gpx.GPXPoint{
					Point: gpx.Point{
						Latitude:  p.Lat,
						Longitude: p.Lon,
						Elevation: *gpx.NewNullableFloat64(p.Elevation),
					},
				})
			}
			trk.Segments = append(trk.Segments, seg)
		}
		g.Tracks = append(g.Tracks, trk)
	}
	xml, _ := gpx.ToXml(g, gpx.ToXmlParams{Indent: true})
	os.WriteFile(path, xml, 0644)
}

func saveMiddlePointsGeoJSON(path string, points []methods.Point) {
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

func saveMiddlePointsGPX(path string, points []methods.Point) {
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

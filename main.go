package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/EverydayRoadster/Mittelpunkte/methods"
	"github.com/jonas-p/go-shp"
	geojson "github.com/paulmach/go.geojson"
	"github.com/tkrajina/gpxgo/gpx"
)

func main() {
	inputDir := flag.String("input", "", "Directory containing ESRI Shapefiles")
	outputDir := flag.String("output", ".", "Output directory")
	filterNames := flag.String("filter", "", "Comma-separated list of area names to include")
	resolution := flag.Float64("resolution", 30.0, "Resolution in meters for grid-based methods (default 30m for speed)")
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

	var allAreas []methods.Area
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
	var selectedAreas []methods.Area
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

	// Calculate middle points
	calcMethods := []methods.CalculationMethod{
		methods.BoundingBoxCenter{},
		methods.IntersectionOfOutermost{},
		methods.CenterOfGravity{Resolution: *resolution},
		methods.MinimalDistanceSum{},
		methods.RotatingBoundingBoxCenter{},
		methods.MinimalDistanceSumEqualSpaced{},
		methods.ReliefCenterOfGravity{Resolution: *resolution},
		methods.FermatPointF1{Resolution: *resolution},
		methods.CenterOfMassSquared{Resolution: *resolution},
		methods.MaximumInscribedCircle{Resolution: *resolution},
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
}

func readShapefile(path string) ([]methods.Area, error) {
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

	var areas []methods.Area
	for s.Next() {
		idx, shape := s.Shape()

		name := fmt.Sprintf("Area %d", idx)
		if nameIdx != -1 {
			val := s.ReadAttribute(idx, nameIdx)
			if strings.TrimSpace(val) != "" {
				name = strings.TrimSpace(val)
			}
		}

		var points []methods.Point
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
			areas = append(areas, methods.Area{Name: name, Points: points})
		}
	}

	return areas, nil
}

func convertPoint(x, y, z float64, isSwiss bool) methods.Point {
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

		return methods.Point{Lat: lat, Lon: lon, Elevation: z}
	}
	return methods.Point{Lat: y, Lon: x, Elevation: z}
}

func saveGeoJSON(path string, areas []methods.Area) {
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

func saveGPX(path string, areas []methods.Area) {
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

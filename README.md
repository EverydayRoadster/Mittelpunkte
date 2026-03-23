# Mittelpunkte

**Mittelpunkte** (German for "Centers" or "Midpoints") is a Go-based command-line tool designed to calculate various types of geographical "center points" for areas defined in ESRI Shapefiles. Whether you are looking for the centroid, the point furthest from the boundary, or the point that minimizes travel distance, this tool provides a suite of algorithms to find the "middle" of any polygon.

## Features

- **Multiple Calculation Methods**: Supports 10 different algorithms for determining the center of an area.
- **ESRI Shapefile Support**: Reads standard `.shp` files (and associated `.prj` for coordinate systems).
- **Swiss Coordinate Support**: Automatically detects and converts Swiss CH1903+/LV95 coordinates to WGS84.
- **GeoJSON & GPX Output**: Saves both the input areas and the calculated middle points in formats ready for GIS tools or GPS devices.
- **Grid-based Approximations**: Uses efficient grid-based methods for complex calculations like the Maximum Inscribed Circle or Relief Center of Gravity.
- **Elevation Support**: Can fetch real-world elevation data via the OpenTopoData API for 3D center of gravity calculations.

## Installation

Ensure you have Go installed on your system. You can install the tool directly using:

```bash
go install github.com/EverydayRoadster/Mittelpunkte@latest
```

## Usage

Run the tool by pointing it to a directory containing ESRI Shapefiles:

```bash
Mittelpunkte -input ./data/my_shapes -output ./results
```

### Flags

- `-input`: (Required) Path to the directory containing `.shp` files.
- `-output`: (Default: `.`) Directory where the results will be saved.
- `-filter`: A comma-separated list of area names (from the Shapefile's attributes) to process. If omitted, all areas are processed.
- `-resolution`: (Default: `30.0`) Resolution in meters for grid-based methods. Smaller values increase accuracy but significantly increase computation time.

## Calculation Methods

The tool calculates the following 10 middle points for each selected area:

1.  **BoundingBoxCenter**: The arithmetic mean of the minimum and maximum latitudes and longitudes.
2.  **IntersectionOfOutermost**: The intersection point of the lines connecting the extreme North-South and East-West points.
3.  **CenterOfGravity**: A grid-based approximation of the area's centroid (arithmetic mean of all points inside the area).
4.  **MinimalDistanceSum**: The geometric median of the boundary points; the point that minimizes the sum of Euclidean distances to all points on the polygon's border.
5.  **RotatingBoundingBoxCenter**: The average center point of bounding boxes calculated at 1-degree rotation intervals.
6.  **MinimalDistanceSumEqualSpaced**: Similar to MinimalDistanceSum, but uses points sampled at equal 10-meter intervals along the boundary.
7.  **ReliefCenterOfGravity**: A 3D center of gravity that takes the terrain's surface area into account (requires internet access to fetch elevation data).
8.  **FermatPointF1**: The point inside the area that minimizes the sum of distances to all other points within the area.
9.  **CenterOfMassSquared**: The point that minimizes the sum of *squared* distances to all other points within the area.
10. **MaximumInscribedCircle**: Also known as the "Pole of Inaccessibility"; the point furthest from any boundary of the area.

## Output

For each run, the tool creates a sub-directory in the output path named after the input directory. It generates four files:

- `areas.geojson`: The boundary polygons of the processed areas.
- `areas.gpx`: The boundaries saved as GPX tracks.
- `middle_points.geojson`: All calculated middle points with their method names as properties.
- `middle_points.gpx`: The middle points saved as GPX waypoints.

## Coordinate Systems

The tool primarily works with WGS84 (latitude/longitude). However, it has built-in support for the Swiss coordinate system **CH1903+/LV95 (EPSG:2056)**. If a `.prj` file is found alongside the `.shp` file indicating Swiss coordinates, the tool will automatically perform the transformation.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

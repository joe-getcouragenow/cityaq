package cityaq

import (
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"sync"

	rpc "github.com/ctessum/cityaq/cityaqrpc"
	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/geojson"
	"github.com/ctessum/sparse"
	"github.com/spatialmodel/inmap/emissions/aep/aeputil"
)

// CityAQ estimates the air quality impacts of activities in cities.
type CityAQ struct {
	// CityGeomDir is the location of the directory that holds the
	// GeoJSON geometries of cities.
	CityGeomDir string

	aeputil.SpatialConfig

	// cityPaths holds the locations of the files containing the
	// boundaries of each city.
	cityPaths         map[string]string
	loadCityPathsOnce sync.Once
}

// Cities returns the files in the CityGeomDir directory field of the receiver.
func (c *CityAQ) Cities(ctx context.Context, _ *rpc.CitiesRequest) (*rpc.CitiesResponse, error) {
	c.cityPaths = make(map[string]string)
	r := new(rpc.CitiesResponse)
	err := filepath.Walk(os.ExpandEnv(c.CityGeomDir), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".geojson" {
			return nil
		}
		name, err := c.geojsonName(path, "en")
		if err != nil {
			return err
		}
		r.Names = append(r.Names, name)
		c.cityPaths[name] = path
		return nil
	})
	return r, err
}

func (c *CityAQ) loadCityPaths() {
	c.loadCityPathsOnce.Do(func() {
		if c.cityPaths == nil {
			_, err := c.Cities(context.Background(), nil)
			if err != nil {
				panic(err)
			}
		}
	})
}

// CityGeometry returns the geometry of the requested city.
func (c *CityAQ) CityGeometry(ctx context.Context, req *rpc.CityGeometryRequest) (*rpc.CityGeometryResponse, error) {
	polys, err := c.geojsonGeometry(req.CityName)
	if err != nil {
		return nil, err
	}
	o := &rpc.CityGeometryResponse{
		Polygons: polygonsToRPC([]geom.Polygonal{polys}),
	}
	return o, err
}

func polygonsToRPC(polys []geom.Polygonal) []*rpc.Polygon {
	o := make([]*rpc.Polygon, len(polys))
	for i, poly := range polys {
		o[i] = new(rpc.Polygon)
		o[i].Paths = make([]*rpc.Path, len(poly.(geom.Polygon)))
		for j, path := range poly.(geom.Polygon) {
			o[i].Paths[j] = new(rpc.Path)
			o[i].Paths[j].Points = make([]*rpc.Point, len(path))
			for k, pt := range path {
				o[i].Paths[j].Points[k] = &rpc.Point{X: float32(pt.X), Y: float32(pt.Y)}
			}
		}
	}
	return o
}

// geojsonGeometry returns the geometry of the requested geojson file.
func (c *CityAQ) geojsonGeometry(cityName string) (geom.Polygon, error) {
	type gj struct {
		Type     string `json:"type"`
		Features []struct {
			Type     string           `json:"type"`
			Geometry geojson.Geometry `json:"geometry"`
		} `json:"features"`
	}

	c.loadCityPaths()
	path, ok := c.cityPaths[cityName]
	if !ok {
		return nil, fmt.Errorf("invalid city name %s", cityName)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening city geojson file: %v", err)
	}
	dec := json.NewDecoder(f)
	var data gj
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	var polys geom.Polygon
	for _, ft := range data.Features {
		g, err := geojson.FromGeoJSON(&ft.Geometry)
		if err != nil {
			return nil, err
		}
		switch g.(type) {
		case geom.Polygon:
			polys = append(polys, g.(geom.Polygon)...)
		case geom.MultiPolygon:
			for _, poly := range g.(geom.MultiPolygon) {
				polys = append(polys, poly...)
			}
		}
	}
	return polys, nil
}

// geojsonName returns a city name (in the requested language)
// from a GeoJSON file.
func (c *CityAQ) geojsonName(path, language string) (string, error) {
	type gj interface{}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	dec := json.NewDecoder(f)
	var data gj
	if err := dec.Decode(&data); err != nil {
		return "", fmt.Errorf("file %s: %v", path, err)
	}
	features := data.(map[string]interface{})["features"].([]interface{})
	for _, feat := range features {
		featmap, ok := feat.(map[string]interface{})
		if !ok {
			continue
		}
		props, ok := featmap["properties"]
		if !ok {
			continue
		}
		propmap, ok := props.(map[string]interface{})
		if !ok {
			continue
		}
		name, ok := propmap["name:"+language]
		if !ok {
			name, ok = propmap["name"]
			if !ok {
				return "", fmt.Errorf("file %s, missing name in language `%s`", path, language)
			}
		}
		return name.(string), nil
	}
	return "", fmt.Errorf("couldn't find name in %v", data)
}

// EmissionsGrid returns the grid to be used for mapping gridded information about the requested city.
func (c *CityAQ) EmissionsGrid(ctx context.Context, req *rpc.EmissionsGridRequest) (*rpc.EmissionsGridResponse, error) {
	o, err := c.emissionsGrid(req.CityName)
	if err != nil {
		return nil, err
	}
	return &rpc.EmissionsGridResponse{Polygons: polygonsToRPC(o)}, nil
}

// emissionsGrid returns the grid to be used for mapping gridded information about the requested city.
func (c *CityAQ) emissionsGrid(cityName string) ([]geom.Polygonal, error) {
	cityGeom, err := c.geojsonGeometry(cityName)
	if err != nil {
		return nil, err
	}
	b := cityGeom.Bounds()

	var o []geom.Polygonal
	const bufferFrac = 0.1
	buffer := math.Sqrt((b.Max.X-b.Min.X)*(b.Max.Y-b.Min.Y)) * bufferFrac
	b.Min.X -= buffer
	b.Min.Y -= buffer
	b.Max.X += buffer
	b.Max.Y += buffer
	const dx = 0.002
	for y := b.Min.Y; y < b.Max.Y+dx; y += dx {
		for x := b.Min.X; x < b.Max.X+dx; x += dx {
			o = append(o, geom.Polygon{
				{
					{X: x, Y: y}, {X: x + dx, Y: y}, {X: x + dx, Y: y + dx}, {X: x, Y: y + dx},
				},
			})
		}
	}
	return o, nil
}

// EmissionsMap returns the requested mapped information about the requested city.
func (c *CityAQ) EmissionsMap(ctx context.Context, req *rpc.EmissionsMapRequest) (*rpc.EmissionsMapResponse, error) {
	emis, err := c.griddedEmissions(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(emis.Shape) != 2 {
		panic(fmt.Errorf("invalid shape %d", emis.Shape))
	}
	if emis.Shape[1] != 1 {
		panic(fmt.Errorf("emis.Shape[1] must be 1 but is %d", emis.Shape))
	}
	rows := emis.Shape[0]

	cm := newColormap(emis)

	out := new(rpc.EmissionsMapResponse)
	out.RGB = make([][]byte, rows)
	for i := 0; i < rows; i++ {
		v := emis.Get(i, 0)
		c, err := cm.At(v)
		if err != nil {
			return nil, fmt.Errorf("cityaq: creating emissions map: %v", err)
		}
		col := color.NRGBAModel.Convert(c).(color.NRGBA)
		out.RGB[i] = []byte{col.R, col.G, col.B}
	}
	out.Legend = legend(cm)
	return out, nil
}

// EmissionsGridBounds returns the bounds of the grid to be used for
// mapping gridded information about the requested city.
func (c *CityAQ) EmissionsGridBounds(ctx context.Context, req *rpc.EmissionsGridBoundsRequest) (*rpc.EmissionsGridBoundsResponse, error) {
	o, err := c.emissionsGrid(req.CityName)
	if err != nil {
		return nil, err
	}
	b := geom.NewBounds()
	for _, g := range o {
		b.Extend(g.Bounds())
	}
	return &rpc.EmissionsGridBoundsResponse{
		Min: &rpc.Point{X: float32(b.Min.X), Y: float32(b.Min.Y)},
		Max: &rpc.Point{X: float32(b.Max.X), Y: float32(b.Max.Y)},
	}, nil
}

// MapScale returns statistics about map data.
func (c *CityAQ) MapScale(ctx context.Context, req *rpc.MapScaleRequest) (*rpc.MapScaleResponse, error) {
	var data *sparse.SparseArray
	var err error
	switch req.ImpactType {
	case rpc.ImpactType_Emissions:
		data, err = c.griddedEmissions(ctx, &rpc.EmissionsMapRequest{
			CityName:   req.CityName,
			Emission:   req.Emission,
			SourceType: req.SourceType,
		})
	default:
		return nil, fmt.Errorf("invalid impact type %s", req.ImpactType.String())
	}
	if err != nil {
		return nil, err
	}

	min, max := math.Inf(1), math.Inf(-1)
	for _, e := range data.Elements {
		if e < min {
			min = e
		}
		if e > max {
			max = e
		}
	}
	max += max * 0.0001
	min -= min * 0.0001
	return &rpc.MapScaleResponse{Min: float32(min), Max: float32(max)}, nil
}

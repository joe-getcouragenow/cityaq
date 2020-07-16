package cityaq

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	rpc "github.com/ctessum/cityaq/cityaqrpc"
	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/geojson"
	"github.com/ctessum/geom/index/rtree"
	"github.com/ctessum/requestcache/v3"
	"github.com/spatialmodel/inmap/cloud"
	"github.com/spatialmodel/inmap/emissions/aep/aeputil"
)

// CityAQ estimates the air quality impacts of activities in cities.
type CityAQ struct {
	// CityGeomDir is the location of the directory that holds the
	// GeoJSON geometries of cities.
	CityGeomDir string

	aeputil.SpatialConfig

	// Location where temporary results should be stored.
	CacheLoc    string
	inmapClient *cloud.Client

	// InMAPConfigFile specifies the path to the file with InMAP
	// configuration information.
	InMAPConfigFile string

	// cityPaths holds the locations of the files containing the
	// boundaries of each city.
	cityPaths         map[string]string
	loadCityPathsOnce sync.Once

	countries         *rtree.Rtree
	loadCountriesOnce sync.Once
	cloudSetupOnce    sync.Once

	cacheSetupOnce sync.Once
	cache          *requestcache.Cache
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

func (c *CityAQ) setupCache() {
	c.cacheSetupOnce.Do(func() {
		workers := runtime.GOMAXPROCS(-1)
		d := requestcache.Deduplicate()
		m := requestcache.Memory(20)
		if c.CacheLoc == "" {
			c.cache = requestcache.NewCache(workers, d, m)
		} else if strings.HasPrefix(c.CacheLoc, "gs://") {
			loc, err := url.Parse(c.CacheLoc)
			if err != nil {
				panic(err)
			}
			cf, err := requestcache.GoogleCloudStorage(context.TODO(), loc.Host,
				strings.TrimLeft(loc.Path, "/"))
			if err != nil {
				panic(err)
			}
			c.cache = requestcache.NewCache(workers, d, m, cf)
		} else {
			c.cache = requestcache.NewCache(workers, d, m,
				requestcache.Disk(strings.TrimPrefix(c.CacheLoc, "file://")))
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
		Polygons: polygonsToRPC([]geom.Polygon{polys}),
	}
	return o, err
}

func polygonalsToRPC(polys []geom.Polygonal) []*rpc.Polygon {
	o := make([]*rpc.Polygon, len(polys))
	for i, poly := range polys {
		o[i] = new(rpc.Polygon)
		o[i].Paths = make([]*rpc.Path, len(poly.(geom.Polygon)))
		for j, path := range poly.(geom.Polygon) {
			o[i].Paths[j] = new(rpc.Path)
			o[i].Paths[j].Points = make([]*rpc.Point, len(path))
			for k, pt := range path {
				o[i].Paths[j].Points[k] = &rpc.Point{X: pt.X, Y: pt.Y}
			}
		}
	}
	return o
}

func polygonsToRPC(polys []geom.Polygon) []*rpc.Polygon {
	o := make([]*rpc.Polygon, len(polys))
	for i, poly := range polys {
		o[i] = new(rpc.Polygon)
		o[i].Paths = make([]*rpc.Path, len(poly))
		for j, path := range poly {
			o[i].Paths[j] = new(rpc.Path)
			o[i].Paths[j].Points = make([]*rpc.Point, len(path))
			for k, pt := range path {
				o[i].Paths[j].Points[k] = &rpc.Point{X: pt.X, Y: pt.Y}
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
// from a GeoJSON file. (Language doesn't currently do anything)
func (c *CityAQ) geojsonName(path, _ string) (string, error) {
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
		name, ok := propmap["c40_city_name"]
		if !ok {
			name, ok = propmap["name"]
			if !ok {
				return "", fmt.Errorf("file %s, missing name", path)
			}
		}
		return name.(string), nil
	}
	return "", fmt.Errorf("couldn't find name in %v", data)
}

// emissionsGrid returns the grid to be used for mapping gridded information about the requested city.
// dx is grid cell edge length in degrees.
func (c *CityAQ) emissionsGrid(cityName, sourceType string, dx float64) ([]geom.Polygonal, error) {
	if dx <= 0 {
		return nil, fmt.Errorf("cityaq: emissions grid dx must be >0 but is %g", dx)
	}
	polygon, err := c.geojsonGeometry(cityName)
	if err != nil {
		return nil, err
	}
	if egugridEmissions(sourceType) {
		// Use EGU grid geometry instead of city.
		country, err := c.countryOrGridBuffer(cityName)
		if err != nil {
			return nil, err
		}
		polygon = country.Polygon
	}
	b := polygon.Bounds()

	var o []geom.Polygonal
	const bufferFrac = 0.1
	buffer := math.Sqrt((b.Max.X-b.Min.X)*(b.Max.Y-b.Min.Y)) * bufferFrac
	b.Min.X -= buffer
	b.Min.Y -= buffer
	b.Max.X += buffer
	b.Max.Y += buffer
	// TODO: Revert so that all cities have the same resolution.
	if cityName == "Tokyo" || cityName == "Guadalajara" || cityName == "Melbourne" {
		b.Min.X = roundUnit(b.Min.X, dx)
		b.Min.Y = roundUnit(b.Min.Y, dx)
		b.Max.X = roundUnit(b.Max.X+dx/2, dx) // Round the max values up.
		b.Max.Y = roundUnit(b.Max.Y+dx/2, dx) // Round the max values up.
	}
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

// EmissionsGridBounds returns the bounds of the grid to be used for
// mapping gridded information about the requested city.
func (c *CityAQ) EmissionsGridBounds(ctx context.Context, req *rpc.EmissionsGridBoundsRequest) (*rpc.EmissionsGridBoundsResponse, error) {
	o, err := c.emissionsGrid(req.CityName, req.SourceType, mapResolution(req.SourceType, req.CityName))
	if err != nil {
		return nil, err
	}
	b := geom.NewBounds()
	for _, g := range o {
		b.Extend(g.Bounds())
	}
	return &rpc.EmissionsGridBoundsResponse{
		Min: &rpc.Point{X: b.Min.X, Y: b.Min.Y},
		Max: &rpc.Point{X: b.Max.X, Y: b.Max.Y},
	}, nil
}

// MapScale returns statistics about map data.
func (c *CityAQ) MapScale(ctx context.Context, req *rpc.MapScaleRequest) (*rpc.MapScaleResponse, error) {
	var data []float64
	switch req.ImpactType {
	case rpc.ImpactType_Emissions:
		response, err := c.GriddedEmissions(ctx, &rpc.GriddedEmissionsRequest{
			CityName:   req.CityName,
			Emission:   req.Emission,
			SourceType: req.SourceType,
		})
		if err != nil {
			return nil, err
		}
		data = response.Emissions
	case rpc.ImpactType_Concentrations:
		response, err := c.GriddedConcentrations(ctx, &rpc.GriddedConcentrationsRequest{
			CityName:   req.CityName,
			Emission:   req.Emission,
			SourceType: req.SourceType,
		})
		if err != nil {
			return nil, err
		}
		data = response.Concentrations
	default:
		return nil, fmt.Errorf("invalid impact type %s", req.ImpactType.String())
	}

	min, max := math.Inf(1), math.Inf(-1)
	for _, e := range data {
		if e < min {
			min = e
		}
		if e > max {
			max = e
		}
	}
	max += max * 0.0001
	min -= min * 0.0001
	return &rpc.MapScaleResponse{Min: min, Max: max}, nil
}

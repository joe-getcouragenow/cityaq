package cityaq

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	rpc "github.com/ctessum/cityaq/cityaqrpc"
	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/geojson"
)

// CityAQ estimates the air quality impacts of activities in cities.
type CityAQ struct {
	// CityGeomDir is the location of the directory that holds the
	// GeoJSON geometries of cities.
	CityGeomDir string
}

// Cities returns the files in the CityGeomDir directory field of the receiver.
func (c *CityAQ) Cities(ctx context.Context, _ *rpc.CitiesRequest) (*rpc.CitiesResponse, error) {
	r := new(rpc.CitiesResponse)
	err := filepath.Walk(os.ExpandEnv(c.CityGeomDir), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		r.Names = append(r.Names, path)
		r.Paths = append(r.Paths, path)
		return nil
	})
	return r, err
}

// CityGeometry returns the geometry of the requested city.
func (c *CityAQ) CityGeometry(ctx context.Context, req *rpc.CityGeometryRequest) (*rpc.CityGeometryResponse, error) {
	polys, err := c.geojsonGeometry(req.Path)
	if err != nil {
		return nil, err
	}
	o := &rpc.CityGeometryResponse{
		Polygons: make([]*rpc.Polygon, len(polys)),
	}
	for i, poly := range polys {
		o.Polygons[i] = new(rpc.Polygon)
		o.Polygons[i].Paths = make([]*rpc.Path, len(poly))
		for j, path := range poly {
			o.Polygons[i].Paths[j] = new(rpc.Path)
			o.Polygons[i].Paths[j].Points = make([]*rpc.Point, len(path))
			for k, pt := range path {
				o.Polygons[i].Paths[j].Points[k] = new(rpc.Point)
				o.Polygons[i].Paths[j].Points[k] = &rpc.Point{X: float32(pt.X), Y: float32(pt.Y)}
			}
		}
	}
	return o, err
}

// geojsonGeometry returns the geometry of the requested geojson file.
func (c *CityAQ) geojsonGeometry(path string) ([]geom.Polygon, error) {
	type gj struct {
		Type     string `json:"type"`
		Features []struct {
			Type     string           `json:"type"`
			Geometry geojson.Geometry `json:"geometry"`
		} `json:"features"`
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(f)
	var data gj
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	var polys []geom.Polygon
	for _, ft := range data.Features {
		g, err := geojson.FromGeoJSON(&ft.Geometry)
		if err != nil {
			return nil, err
		}
		if poly, ok := g.(geom.Polygon); ok {
			polys = append(polys, poly)
		}
	}
	return polys, nil
}

// MapGrid returns the grid to be used for mapping gridded information about the requested city.
func (c *CityAQ) MapGrid(ctx context.Context, req *rpc.MapGridRequest) (*rpc.MapGridResponse, error) {
	cityGeom, err := c.geojsonGeometry(req.Path)
	if err != nil {
		return nil, err
	}
	b := geom.NewBounds()
	for _, p := range cityGeom {
		b.Extend(p.Bounds())
	}

	o := new(rpc.MapGridResponse)
	const buffer = 0.5
	b.Min.X -= buffer
	b.Min.Y -= buffer
	b.Max.X += buffer
	b.Max.Y += buffer
	const dx = 0.001
	for y := float32(b.Min.Y); y < float32(b.Max.Y+dx); y += float32(dx) {
		for x := float32(b.Min.X); x < float32(b.Max.X+dx); x += float32(dx) {
			o.Polygons = append(o.Polygons, &rpc.Polygon{
				Paths: []*rpc.Path{{
					Points: []*rpc.Point{
						{X: x, Y: y}, {X: x + dx, Y: y}, {X: x + dx, Y: y + dx}, {X: x, Y: y + dx},
					},
				}},
			})
		}
	}
	return o, nil
}

// GriddedMapData returns the requested mapped information about the requested city.
func (c *CityAQ) GriddedMapData(ctx context.Context, req *rpc.GriddedMapDataRequest) (*rpc.GriddedMapDataResponse, error) {

	return nil, nil
}

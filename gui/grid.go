// +build js

package gui

import (
	"context"
	"encoding/json"
	"syscall/js"

	rpc "github.com/ctessum/cityaq/cityaqrpc"
	"github.com/ctessum/geom"
)

func (c *CityAQ) loadEmissionsGrid(ctx context.Context, sel *selections) error {
	if c.grid.gridCity == sel.cityName && c.grid.gridType == emission {
		return nil // We already have the correct grid loaded.
	}
	c.grid.gridCity = sel.cityName
	c.grid.gridType = emission
	resp, err := c.EmissionsGrid(ctx, &rpc.EmissionsGridRequest{
		CityName: sel.cityName,
		Path:     sel.cityPath,
	})
	if err != nil {
		return err
	}
	gj, bnds := polygonToGeoJSON(resp.Polygons)
	gjBytes, err := json.Marshal(gj)
	if err != nil {
		return err
	}
	c.grid.geometry = js.Global().Get("JSON").Call("parse", string(gjBytes))
	go func() {
		c.MoveMap(bnds) // Move map to new city.
	}()
	return nil
}

type geojson struct { // Omitting GeoJSON types to save space.
	//Type     string        `json:"type",js:"type"`
	Features []geojsonGeom `json:"features",js:"features"`
}
type geojsonGeom struct {
	//Type     string `json:"type",js:"type"`
	Geometry struct {
		//Type        string        `json:"type",js:"type"`
		Coordinates [][][]float32 `json:"coordinates",js:"coordinates"`
	} `json:"geometry,js:"geometry"`
}

func polygonToGeoJSON(polys []*rpc.Polygon) (*geojson, *geom.Bounds) {
	g := &geojson{
		//Type: "Polygon",
		Features: make([]geojsonGeom, len(polys)),
	}
	b := geom.NewBounds()
	for i, poly := range polys {
		g.Features[i].Geometry.Coordinates = make([][][]float32, len(poly.Paths))
		for j, path := range poly.Paths {
			g.Features[i].Geometry.Coordinates[j] = make([][]float32, len(path.Points))
			for k, point := range path.Points {
				g.Features[i].Geometry.Coordinates[j][k] = []float32{point.X, point.Y}
				b.Extend(geom.Point{X: float64(point.X), Y: float64(point.Y)}.Bounds())
			}
		}
	}
	return g, b
}

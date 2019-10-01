// +build js

package gui

import (
	"context"

	rpc "github.com/ctessum/cityaq/cityaqrpc"
	"github.com/norunners/vert"
	"google.golang.org/grpc/grpclog"
)

func (c *CityAQ) loadEmissionsGrid(ctx context.Context, sel *selections) {
	if c.grid.gridCity == sel.cityName && c.grid.gridType == emission {
		return // We already have the correct grid loaded.
	}
	c.grid.gridCity = sel.cityName
	c.grid.gridType = emission
	resp, err := c.EmissionsGrid(ctx, &rpc.EmissionsGridRequest{
		CityName: sel.cityName,
		Path:     sel.cityPath,
	})
	if err != nil {
		grpclog.Println(err)
		return
	}
	gj := polygonToGeoJSON(resp.Polygons)
	c.grid.geometry = vert.ValueOf(gj).JSValue()
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

func polygonToGeoJSON(polys []*rpc.Polygon) *geojson {
	g := &geojson{
		//Type: "Polygon",
		Features: make([]geojsonGeom, len(polys)),
	}
	for i, poly := range polys {
		g.Features[i].Geometry.Coordinates = make([][][]float32, len(poly.Paths))
		for j, path := range poly.Paths {
			g.Features[i].Geometry.Coordinates[j] = make([][]float32, len(path.Points))
			for k, point := range path.Points {
				g.Features[i].Geometry.Coordinates[j][k] = []float32{point.X, point.Y}
			}
		}
	}
	return g
}

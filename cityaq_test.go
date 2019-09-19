package cityaq

import (
	"context"
	"reflect"
	"testing"

	rpc "github.com/ctessum/cityaq/cityaqrpc"
	"github.com/ctessum/geom"
)

func TestCityAQ_Cities(t *testing.T) {
	c := &CityAQ{
		CityGeomDir: "testdata/cities",
	}

	cities, err := c.Cities(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	want := &rpc.CitiesResponse{
		Names: []string{
			"testdata/cities/accra_jurisdiction.geojson",
			"testdata/cities/karachi_jurisdiction.geojson",
		},
		Paths: []string{
			"testdata/cities/accra_jurisdiction.geojson",
			"testdata/cities/karachi_jurisdiction.geojson",
		},
	}
	if !reflect.DeepEqual(want, cities) {
		t.Errorf("%v != %v", cities, want)
	}
}

func TestCityAQ_CityGeometry(t *testing.T) {
	r := &rpc.CityGeometryRequest{
		Path: "testdata/cities/accra_jurisdiction.geojson",
	}

	c := &CityAQ{
		CityGeomDir: "testdata/cities",
	}
	polys, err := c.CityGeometry(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	b := polygonBounds(polys.Polygons)
	want := &geom.Bounds{
		Min: geom.Point{X: -0.28431379795074463, Y: 5.515096187591553},
		Max: geom.Point{X: -0.1248164027929306, Y: 5.653964996337891},
	}
	if !reflect.DeepEqual(want, b) {
		t.Errorf("%v != %v", b, want)
	}
}

func polygonBounds(polys []*rpc.Polygon) *geom.Bounds {
	b := geom.NewBounds()
	for _, poly := range polys {
		for _, path := range poly.Paths {
			for _, pt := range path.Points {
				b.Extend(geom.Point{X: float64(pt.X), Y: float64(pt.Y)}.Bounds())
			}
		}
	}
	return b
}

func TestCityAQ_MapGrid(t *testing.T) {
	r := &rpc.MapGridRequest{
		Path: "testdata/cities/accra_jurisdiction.geojson",
	}
	c := &CityAQ{
		CityGeomDir: "testdata/cities",
	}
	polys, err := c.MapGrid(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	b := polygonBounds(polys.Polygons)
	want := &geom.Bounds{
		Min: geom.Point{X: -0.7843137979507446, Y: 5.015096187591553},
		Max: geom.Point{X: 0.37667834758758545, Y: 6.155013561248779},
	}
	if !reflect.DeepEqual(want, b) {
		t.Errorf("%v != %v", b, want)
	}
}

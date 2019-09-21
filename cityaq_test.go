package cityaq

import (
	"context"
	"image/color"
	"math"
	"reflect"
	"testing"

	rpc "github.com/ctessum/cityaq/cityaqrpc"
	"github.com/ctessum/geom"
	"github.com/spatialmodel/inmap/emissions/aep/aeputil"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
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

func TestCityAQ_EmissionsGrid(t *testing.T) {
	r := &rpc.EmissionsGridRequest{
		Path: "testdata/cities/accra_jurisdiction.geojson",
	}
	c := &CityAQ{
		CityGeomDir: "testdata/cities",
	}
	polys, err := c.EmissionsGrid(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	b := polygonBounds(polys.Polygons)
	want := &geom.Bounds{
		Min: geom.Point{X: -0.7843137979507446, Y: 5.015096187591553}, // -44.9378704365,
		Max: geom.Point{X: 0.3856861889362335, Y: 6.165096282958984},
	}
	if !reflect.DeepEqual(want, b) {
		t.Errorf("%v != %v", b, want)
	}
}

// openstreetmap data from:
// https://api.openstreetmap.org/api/0.6/map?bbox=-0.28,5.52,-0.12,5.65
func TestCityAQ_griddedEmissions(t *testing.T) {
	c := &CityAQ{
		CityGeomDir: "testdata/cities",
		SpatialConfig: aeputil.SpatialConfig{
			SrgSpec:       "testdata/srgspec_osm.json",
			SrgSpecType:   "OSM",
			SCCExactMatch: true,
			GridRef:       []string{"testdata/gridref_osm.txt"},
			OutputSR:      "+proj=longlat",
			InputSR:       "+proj=longlat",
		},
	}
	req := &rpc.EmissionsMapRequest{
		CityPath:   "testdata/cities/accra_jurisdiction.geojson",
		CityName:   "accra",
		Emission:   rpc.Emission_PM2_5,
		SourceType: "roads",
	}

	emis, err := c.griddedEmissions(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if emis == nil {
		t.Fatal("nil emis")
	}
	sum := emis.Sum()
	want := 993.4302004987658
	if !similar(sum, want, 1e-10) {
		t.Errorf("have %g, want %g", sum, want)
	}
}

func similar(a, b, tol float64) bool {
	if math.Abs(a-b) > tol || 2*math.Abs(a-b)/(a+b) > tol {
		return false
	}
	return true
}

func TestCityAQ_EmissionsMap(t *testing.T) {
	c := &CityAQ{
		CityGeomDir: "testdata/cities",
		SpatialConfig: aeputil.SpatialConfig{
			SrgSpec:       "testdata/srgspec_osm.json",
			SrgSpecType:   "OSM",
			SCCExactMatch: true,
			GridRef:       []string{"testdata/gridref_osm.txt"},
			OutputSR:      "+proj=longlat",
			InputSR:       "+proj=longlat",
		},
	}
	req := &rpc.EmissionsMapRequest{
		CityPath:   "testdata/cities/accra_jurisdiction.geojson",
		CityName:   "accra",
		Emission:   rpc.Emission_PM2_5,
		SourceType: "roads",
	}
	emis, err := c.EmissionsMap(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	grid, err := c.EmissionsGrid(context.Background(), &rpc.EmissionsGridRequest{
		Path: "testdata/cities/accra_jurisdiction.geojson",
	})
	p, err := plot.New()
	if err != nil {
		t.Fatal(err)
	}
	for i, g := range grid.Polygons {
		xys := polygonToXYs(g)
		poly, err := plotter.NewPolygon(xys...)
		if err != nil {
			t.Fatal(err)
		}
		poly.Color = bytesToColor(emis.RGB[i])
		poly.LineStyle.Color = color.Transparent
		p.Add(poly)
	}
	p.Save(1000, 1000, "testdata/emis.png")
}

func bytesToColor(b []byte) color.Color {
	rgb := color.NRGBA{}
	rgb.R = uint8(b[0])
	rgb.G = uint8(b[1])
	rgb.B = uint8(b[2])
	rgb.A = 255
	return rgb
}

func polygonToXYs(poly *rpc.Polygon) []plotter.XYer {
	var o []plotter.XYer
	for _, path := range poly.Paths {
		xys := make(plotter.XYs, len(path.Points))
		for i, pt := range path.Points {
			xy := plotter.XY{float64(pt.X), float64(pt.Y)}
			xys[i] = xy
		}
		o = append(o, xys)
	}
	return o
}

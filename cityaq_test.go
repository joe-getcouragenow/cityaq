package cityaq

import (
	"context"
	"image/color"
	"io/ioutil"
	"math"
	"os"
	"reflect"
	"testing"

	"github.com/ctessum/cityaq/cityaqrpc"
	rpc "github.com/ctessum/cityaq/cityaqrpc"
	"github.com/ctessum/geom"
	"github.com/spatialmodel/inmap/emissions/aep/aeputil"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/cmpimg"
	"gonum.org/v1/plot/palette/moreland"
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
			"Accra Metropolitan",
			"ڪراچي Karachi",
		},
	}
	if !reflect.DeepEqual(want, cities) {
		t.Errorf("%v != %v", cities, want)
	}
}

func TestCityAQ_CityGeometry(t *testing.T) {
	r := &rpc.CityGeometryRequest{
		CityName: "Accra Metropolitan",
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
		Min: geom.Point{X: -0.2843138, Y: 5.5150962},
		Max: geom.Point{X: -0.1248164, Y: 5.6539649},
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
				b.Extend(geom.Point{X: pt.X, Y: pt.Y}.Bounds())
			}
		}
	}
	return b
}

func TestCityAQ_EmissionsGrid(t *testing.T) {
	c := &CityAQ{
		CityGeomDir: "testdata/cities",
		SpatialConfig: aeputil.SpatialConfig{
			SrgSpecOSM:    "testdata/srgspec_osm.json",
			SCCExactMatch: true,
			GridRef:       []string{"testdata/gridref.txt"},
			OutputSR:      "+proj=longlat",
			InputSR:       "+proj=longlat",
		},
	}

	r := &rpc.GriddedEmissionsRequest{
		CityName:   "Accra Metropolitan",
		Emission:   rpc.Emission_PM2_5,
		SourceType: "roadways",
	}

	emissions, err := c.GriddedEmissions(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	b := polygonBounds(emissions.Polygons)
	want := &geom.Bounds{
		Min: geom.Point{X: -0.3, Y: 5.5},
		Max: geom.Point{X: -0.09999999999999981, Y: 5.679999999999996},
	}
	if !reflect.DeepEqual(want, b) {
		t.Errorf("%v != %v", b, want)
	}
}

func TestCityAQ_EmissionsGridBounds(t *testing.T) {
	r := &rpc.EmissionsGridBoundsRequest{
		CityName: "Accra Metropolitan",
	}
	c := &CityAQ{
		CityGeomDir: "testdata/cities",
	}
	bounds, err := c.EmissionsGridBounds(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	want := &rpc.EmissionsGridBoundsResponse{
		Min: &rpc.Point{X: -0.3, Y: 5.5},
		Max: &rpc.Point{X: -0.09999999999999981, Y: 5.679999999999996},
	}
	if !reflect.DeepEqual(want, bounds) {
		t.Errorf("%v != %v", bounds, want)
	}
}

func TestCityAQ_griddedEmissions(t *testing.T) {
	c := &CityAQ{
		CityGeomDir: "testdata/cities",
		SpatialConfig: aeputil.SpatialConfig{
			SrgSpecOSM:    "testdata/srgspec_osm.json",
			SCCExactMatch: true,
			GridRef:       []string{"testdata/gridref.txt"},
			OutputSR:      "+proj=longlat",
			InputSR:       "+proj=longlat",
		},
	}

	for _, st := range []string{"roadways", "airports"} {
		t.Run(st, func(t *testing.T) {
			req := &rpc.GriddedEmissionsRequest{
				CityName:   "Accra Metropolitan",
				Emission:   rpc.Emission_PM2_5,
				SourceType: st,
			}

			emis, err := c.GriddedEmissions(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			if emis == nil {
				t.Fatal("nil emis")
			}
			sum := floats.Sum(emis.Emissions)
			want := 1.0e6
			if !similar(sum, want, 1e-8) {
				t.Errorf("have %g, want %g", sum, want)
			}
		})
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
			SrgSpecOSM:    "testdata/srgspec_osm.json",
			SCCExactMatch: true,
			GridRef:       []string{"testdata/gridref.txt"},
			OutputSR:      "+proj=longlat",
			InputSR:       "+proj=longlat",
		},
	}
	req := &rpc.GriddedEmissionsRequest{
		CityName:   "Accra Metropolitan",
		Emission:   rpc.Emission_PM2_5,
		SourceType: "roadways",
	}
	emis, err := c.GriddedEmissions(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	scale, err := c.MapScale(context.Background(), &rpc.MapScaleRequest{
		CityName:   "Accra Metropolitan",
		Emission:   rpc.Emission_PM2_5,
		ImpactType: rpc.ImpactType_Emissions,
		SourceType: "roadways",
	})
	if err != nil {
		t.Fatal(err)
	}
	p, err := plot.New()
	if err != nil {
		t.Fatal(err)
	}
	cm := moreland.ExtendedBlackBody()
	cm.SetMin(scale.Min)
	cm.SetMax(scale.Max)
	for i, g := range emis.Polygons {
		xys := polygonToXYs(g)
		poly, err := plotter.NewPolygon(xys...)
		if err != nil {
			t.Fatal(err)
		}
		poly.Color, err = cm.At(emis.Emissions[i])
		if err != nil {
			t.Fatal(err)
		}
		poly.LineStyle.Color = color.Transparent
		p.Add(poly)
	}

	city, err := c.CityGeometry(context.Background(), &rpc.CityGeometryRequest{
		CityName: "Accra Metropolitan",
	})
	if err != nil {
		t.Fatal(err)
	}

	cityXYs, err := plotter.NewPolygon(polygonToXYs(city.Polygons[0])...)
	if err != nil {
		t.Fatal(err)
	}
	cityXYs.Color = color.Transparent
	cityXYs.LineStyle.Color = color.White
	p.Add(cityXYs)

	if err := p.Save(200, 200, "testdata/emis.png"); err != nil {
		t.Fatal(err)
	}

	i1, err := ioutil.ReadFile("testdata/emis.png")
	if err != nil {
		t.Fatal(err)
	}
	i2, err := ioutil.ReadFile("testdata/emis_golden.png")
	if err != nil {
		t.Fatal(err)
	}
	eq, err := cmpimg.Equal("png", i1, i2)
	if err != nil {
		t.Fatal(err)
	}
	if !eq {
		t.Fatal("image doesn't match golden image")
	} else {
		if err := os.Remove("testdata/emis.png"); err != nil {
			t.Fatal(err)
		}
	}
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
			xy := plotter.XY{X: pt.X, Y: pt.Y}
			xys[i] = xy
		}
		o = append(o, xys)
	}
	return o
}

func TestCityAQ_MapScale(t *testing.T) {
	c := &CityAQ{
		CityGeomDir: "testdata/cities",
		SpatialConfig: aeputil.SpatialConfig{
			SrgSpecOSM:    "testdata/srgspec_osm.json",
			SCCExactMatch: true,
			GridRef:       []string{"testdata/gridref.txt"},
			OutputSR:      "+proj=longlat",
			InputSR:       "+proj=longlat",
		},
	}
	req := &rpc.MapScaleRequest{
		CityName:   "Accra Metropolitan",
		Emission:   rpc.Emission_PM2_5,
		SourceType: "roadways",
		ImpactType: rpc.ImpactType_Emissions,
	}
	scale, err := c.MapScale(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	wantScale := &cityaqrpc.MapScaleResponse{Min: 0, Max: 4594.353937014378}
	if !similar(scale.Min, wantScale.Min, 1.0e-10) {
		t.Errorf("scale min %+v != %+v", scale.Min, wantScale.Min)
	}
	if !similar(scale.Max, wantScale.Max, 1.0e-10) {
		t.Errorf("scale max %+v != %+v", scale.Max, wantScale.Max)
	}
}

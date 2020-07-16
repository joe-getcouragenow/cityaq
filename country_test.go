package cityaq

import (
	"context"
	"reflect"
	"testing"

	rpc "github.com/ctessum/cityaq/cityaqrpc"

	"github.com/ctessum/geom"
	"github.com/spatialmodel/inmap/emissions/aep/aeputil"
	"gonum.org/v1/gonum/floats"
)

func TestCityAQ_egugrid(t *testing.T) {
	c := &CityAQ{
		CityGeomDir: "testdata/cities",
		SpatialConfig: aeputil.SpatialConfig{
			SrgSpecOSM:            "testdata/srgspec_osm.json",
			SrgSpecSMOKE:          "testdata/srgspec_smoke.csv",
			SrgShapefileDirectory: "testdata",
			SCCExactMatch:         true,
			GridRef:               []string{"testdata/gridref.txt"},
			OutputSR:              "+proj=longlat",
			InputSR:               "+proj=longlat",
		},
	}
	t.Run("cityCountry", func(t *testing.T) {
		country, err := c.countryOrGridBuffer("Accra Metropolitan")
		if err != nil {
			t.Fatal(err)
		}
		wantName := "Ghana"
		wantBounds := &geom.Bounds{
			Min: geom.Point{X: -3.24888920783991, Y: 4.72708272933966},
			Max: geom.Point{X: 1.20277762413031, Y: 11.1556930541993},
		}
		if country.Name != wantName {
			t.Errorf("name: %s != %s", country.Name, wantName)
		}
		if !reflect.DeepEqual(country.Bounds(), wantBounds) {
			t.Errorf("name: %+v != %+v", country.Bounds(), wantBounds)
		}
	})

	t.Run("electric_gen_egugrid", func(t *testing.T) {
		req := &rpc.GriddedEmissionsRequest{
			CityName:   "Accra Metropolitan",
			Emission:   rpc.Emission_PM2_5,
			SourceType: "electric_gen_egugrid",
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

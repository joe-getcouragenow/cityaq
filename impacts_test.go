package cityaq

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	rpc "github.com/ctessum/cityaq/cityaqrpc"
	"github.com/spatialmodel/inmap/emissions/aep/aeputil"
)

func TestCityAQ_ImpactSummary(t *testing.T) {
	dir := fmt.Sprintf("temp_test_%d", time.Now().Unix())
	c := &CityAQ{
		CityGeomDir: "testdata/cities",
		SpatialConfig: aeputil.SpatialConfig{
			SrgSpec:       "testdata/srgspec_osm.json",
			SrgSpecType:   "OSM",
			SCCExactMatch: true,
			GridRef:       []string{"testdata/gridref.txt"},
			OutputSR:      "+proj=longlat",
			InputSR:       "+proj=longlat",
		},
		CacheLoc:        "file://" + dir,
		InMAPConfigFile: "testdata/inmap_config.toml",
	}
	os.Mkdir(dir, os.ModePerm)
	defer os.RemoveAll(dir)

	r := &rpc.ImpactSummaryRequest{
		CityName:   "Accra Metropolitan",
		Emission:   rpc.Emission_PM2_5,
		SourceType: "roadways",
	}

	s, err := c.ImpactSummary(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	// There is no population in the example domain, so we get a null
	// response.
	fmt.Println(s)
}

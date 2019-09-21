package cityaq

import (
	"time"
  "context"
  "fmt"

	rpc "github.com/ctessum/cityaq/cityaqrpc"
  "github.com/ctessum/geom"
	"github.com/ctessum/sparse"
	"github.com/ctessum/geom/proj"
	"github.com/ctessum/unit"
	"github.com/spatialmodel/inmap/emissions/aep"
)

type emissions struct {
	geom.Polygon
	SR *proj.SR
	aep.SourceData
	aep.Emissions
}

// Location returns the polygon representing the location of emissions.
func (e *emissions) Location() *aep.Location {
	return &aep.Location{Geom: e.Polygon, SR: e.SR}
}

func newEmissions(poly geom.Polygon, pollutant rpc.Emission, sourceType string) (*emissions, time.Time, time.Time, error) {
	begin := time.Date(2016, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2017, time.January, 1, 0, 0, 0, 0, time.UTC)

	duration := end.Sub(begin).Seconds()
	rate := unit.New(1000/duration, unit.Dimensions{
		unit.MassDim: 1,
		unit.TimeDim: -1,
	}) // 1 tonne/year in kg/s

	e := new(aep.Emissions)
	e.Add(begin, end, pollutant.String(), "", rate)

	sr, err := proj.Parse("+proj=longlat")
	if err != nil {
		return nil, time.Time{}, time.Time{}, err
	}

	emis := &emissions{
		Polygon:   poly,
		SR:        sr,
		Emissions: *e,
		SourceData: aep.SourceData{
			FIPS:    "00000",
			Country: aep.Global,
			SCC:     sourceType,
		},
	}
	return emis, begin, end, nil
}

func (c *CityAQ) griddedEmissions(ctx context.Context, req *rpc.EmissionsMapRequest) (*sparse.SparseArray, error) {
  g, err := c.geojsonGeometry(req.CityPath)
  if err != nil {
    return nil, err
  }
  e, begin, end, err := newEmissions(g, req.Emission, req.SourceType)
  if err != nil {
    return nil, err
  }

  // Make sure we're not making simultaneous changes to the grid.
  c.gridLock.Lock()
  defer c.gridLock.Unlock()

  grid, err := c.emissionsGrid(req.CityPath)
  if err != nil {
    return nil, err
  }
  c.SpatialConfig.GridCells = grid
  c.SpatialConfig.GridName = req.CityName

  sp, err := c.SpatialConfig.SpatialProcessor()
  if err != nil {
    return nil, err
  }
  r := sp.GridRecord(e)
  gridEmis, _, err := r.GriddedEmissions(begin, end, 0)
  if err != nil {
    return nil, err
  }
  polEmis, ok := gridEmis[aep.Pollutant{Name:req.Emission.String()}]
  if !ok {
    panic(fmt.Errorf("cityaq: missing gridded pollutant %v", req.Emission))
  }
return polEmis, nil
}
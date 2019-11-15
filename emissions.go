package cityaq

import (
	"context"
	"fmt"
	"os"
	"time"

	rpc "github.com/ctessum/cityaq/cityaqrpc"
	"github.com/ctessum/geom"
	"github.com/ctessum/geom/proj"
	"github.com/ctessum/unit"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/mvt"
	"github.com/paulmach/orb/geojson"
	"github.com/spatialmodel/inmap/emissions/aep"
	"github.com/spatialmodel/inmap/emissions/aep/aeputil"
)

type emissions struct {
	geom.Polygon
	SR *proj.SR
	aep.SourceData
	aep.Emissions
	cityName string
}

// Location returns the polygon representing the location of emissions.
func (e *emissions) Location() *aep.Location {
	return &aep.Location{Geom: e.Polygon, SR: e.SR, Name: e.cityName}
}

func newEmissions(poly geom.Polygon, pollutant rpc.Emission, sourceType, cityName string) (*emissions, time.Time, time.Time, error) {
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
			SCC:     "0000" + sourceType,
		},
		cityName: cityName,
	}
	return emis, begin, end, nil
}

func emissionsMapName(r *rpc.GriddedEmissionsRequest) string {
	return fmt.Sprintf("%s_%d_%d_%s", r.CityName, rpc.ImpactType_Emissions, r.Emission, r.SourceType)
}

// GriddedEmissions returns gridded emissions for the request.
// If req.SourceType has the suffix "_egugrid", emissions will be allocated
// to the smaller of country that the city is in or the intersection of
// the country with a 5.4 degree radius buffer around the city,
// otherwise they will be allocated within the city itself.
func (c *CityAQ) GriddedEmissions(ctx context.Context, req *rpc.GriddedEmissionsRequest) (*rpc.GriddedEmissionsResponse, error) {
	g, err := c.geojsonGeometry(req.CityName)
	if err != nil {
		return nil, err
	}
	if egugridEmissions(req.SourceType) {
		// Use EGU grid geometry instead of city.
		country, err := c.countryOrGridBuffer(req.CityName)
		if err != nil {
			return nil, err
		}
		g = country.Polygon
	}
	e, begin, end, err := newEmissions(g, req.Emission, req.SourceType, req.CityName)
	if err != nil {
		return nil, err
	}

	grid, err := c.emissionsGrid(req.CityName, req.SourceType, mapResolution(req.SourceType))
	if err != nil {
		return nil, err
	}

	// Make a copy of the spatial configuration to allow the
	// use of multiple grids.
	spatialConfig := aeputil.SpatialConfig{
		SrgSpec:               c.SpatialConfig.SrgSpec,
		SrgSpecType:           c.SpatialConfig.SrgSpecType,
		SrgShapefileDirectory: c.SpatialConfig.SrgShapefileDirectory,
		SCCExactMatch:         c.SpatialConfig.SCCExactMatch,
		GridRef:               c.SpatialConfig.GridRef,
		OutputSR:              c.SpatialConfig.OutputSR,
		InputSR:               c.SpatialConfig.InputSR,
		SimplifyTolerance:     c.SpatialConfig.SimplifyTolerance,
		SpatialCache:          c.SpatialConfig.SpatialCache,
		MaxCacheEntries:       c.SpatialConfig.MaxCacheEntries,
		GridCells:             grid,
		GridName:              req.CityName,
	}

	sp, err := spatialConfig.SpatialProcessor()
	if err != nil {
		return nil, err
	}

	if err := c.loadSMOKESrgSpecs(sp); err != nil {
		return nil, err
	}

	rSrg := sp.AddSurrogate(e)
	r := sp.GridRecord(rSrg)
	gridEmis, _, err := r.GriddedEmissions(begin, end, 0)
	if err != nil {
		return nil, err
	}
	if len(gridEmis) == 0 {
		return nil, fmt.Errorf("cityaq: no emissions for city %s, source %s", req.CityName, req.SourceType)
	}
	polEmis, ok := gridEmis[aep.Pollutant{Name: req.Emission.String()}]
	if !ok {
		panic(fmt.Errorf("cityaq: missing gridded pollutant %v", req.Emission))
	}

	o := &rpc.GriddedEmissionsResponse{
		Polygons:  polygonsToRPC(grid),
		Emissions: make([]float64, len(grid)),
	}
	for i, v := range polEmis.Elements {
		o.Emissions[i] = v
	}

	return o, nil
}

// loadSMOKESrgSpecs adds supplemental SMOKE-formatted surrogate
// specifications to the given SpatialProcessor.
func (c *CityAQ) loadSMOKESrgSpecs(sp *aep.SpatialProcessor) error {
	if c.SMOKESrgSpecs == "" {
		return nil
	}
	f, err := os.Open(c.SMOKESrgSpecs)
	if err != nil {
		return fmt.Errorf("cityaq: opening SMOKESrgSpecs: %w", err)
	}
	srgSpecs, err := aep.ReadSrgSpecSMOKE(f, c.SpatialConfig.SrgShapefileDirectory,
		true, c.SpatialConfig.SpatialCache, c.SpatialConfig.MaxCacheEntries)
	if err != nil {
		return fmt.Errorf("cityaq: reading SMOKESrgSpecs: %w", err)
	}
	f.Close()
	sp.SrgSpecs.AddAll(srgSpecs)
	return nil
}

func (c *CityAQ) emissionsMapData(ctx context.Context, req *rpc.GriddedEmissionsRequest) (*mvt.Layer, error) {
	emis, err := c.GriddedEmissions(ctx, req)
	if err != nil {
		return nil, err
	}

	layerData := geojson.NewFeatureCollection()
	for i, cell := range emis.Polygons {
		v := emis.Emissions[i]
		if v == 0 {
			continue
		}
		feature := geojson.NewFeature(rpcToOrb(cell))
		feature.ID = uint64(i)
		feature.Properties["v"] = v
		layerData = layerData.Append(feature)
	}
	layer := mvt.NewLayer(emissionsMapName(req), layerData)
	return layer, nil
}

func rpcToOrb(p *rpc.Polygon) orb.Polygon {
	o := make(orb.Polygon, len(p.Paths))
	for i, path := range p.Paths {
		o[i] = make(orb.Ring, len(path.Points))
		for j, point := range path.Points {
			o[i][j] = orb.Point{point.X, point.Y}
		}
	}
	return o
}

func geomToOrb(g geom.Polygonal) orb.Polygon {
	p := g.(geom.Polygon)
	o := make(orb.Polygon, len(p))
	for i, path := range p {
		o[i] = make(orb.Ring, len(path))
		for j, point := range path {
			o[i][j] = orb.Point{point.X, point.Y}
		}
	}
	return o
}

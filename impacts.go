package cityaq

import (
	"context"

	rpc "github.com/ctessum/cityaq/cityaqrpc"
	"github.com/ctessum/geom"
	"github.com/ctessum/geom/index/rtree"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/stat"
)

// ImpactSummary returns a summary of the impacts from the given request.
func (c *CityAQ) ImpactSummary(ctx context.Context, req *rpc.ImpactSummaryRequest) (*rpc.ImpactSummaryResponse, error) {
	conc, err := c.GriddedConcentrations(ctx, &rpc.GriddedConcentrationsRequest{
		CityName:   req.CityName,
		SourceType: req.SourceType,
		Emission:   req.Emission,
	})
	if err != nil {
		return nil, err
	}

	pop, err := c.GriddedPopulation(ctx, &rpc.GriddedPopulationRequest{
		CityName:   req.CityName,
		SourceType: req.SourceType,
		Emission:   req.Emission,
	})
	if err != nil {
		return nil, err
	}
	maskedPop, err := c.maskPopulation(ctx, pop, req.CityName)
	if err != nil {
		return nil, err
	}
	return &rpc.ImpactSummaryResponse{
		Population:     floats.Sum(pop.Population),
		CityPopulation: floats.Sum(maskedPop),
		TotalExposure:  exposure(conc.Concentrations, pop.Population),
		CityExposure:   exposure(conc.Concentrations, maskedPop),
		TotalIF:        iF(conc.Concentrations, pop.Population),
		CityIF:         iF(conc.Concentrations, maskedPop),
	}, nil
}

// exposure returns the population-weighted mean of the concentration.
func exposure(conc, pop []float64) float64 {
	return stat.Mean(conc, pop)
}

// iF returns the intake fraction (in ppm) of the given concentration (μg m-3) and
// population, assuming 1 kilotonne of emissions.
func iF(conc, pop []float64) float64 {
	const (
		br   = 15                  // m3 person-1 day-1
		emis = 1.0e6 * 1.0e9 / 365 // μg / day
	)
	avgConc := exposure(conc, pop) // μg m-3
	popSum := floats.Sum(pop)
	// m3 person-1 day-1 μg m-3 person μg-1 day * 1e6 = ppm
	return br * avgConc * popSum / emis * 1.0e6
}

// maskPopulation masks the given population grid with the city boundaries.
func (c *CityAQ) maskPopulation(ctx context.Context, pop *rpc.GriddedPopulationResponse, city string) ([]float64, error) {
	geomRPC, err := c.CityGeometry(ctx, &rpc.CityGeometryRequest{
		CityName: city,
	})
	if err != nil {
		return nil, err
	}

	type data struct {
		geom.Polygon
		i   int
		pop float64
	}
	index := rtree.NewTree(25, 50)
	for i, p := range pop.Polygons {
		index.Insert(data{Polygon: rpcToGeom(p), i: i, pop: pop.Population[i]})
	}

	maskedPop := make([]float64, len(pop.Population))
	for _, p := range geomRPC.Polygons {
		cityGeom := rpcToGeom(p)
		for _, dI := range index.SearchIntersect(cityGeom.Bounds()) {
			d := dI.(data)
			isect := d.Intersection(cityGeom)
			if isect == nil {
				continue
			}
			// masked population is the fraction of the grid cell area
			// that overlaps with the intersection multiplied by
			// the population.
			maskedPop[d.i] += d.pop * isect.Area() / d.Area()
		}
	}
	return maskedPop, nil
}

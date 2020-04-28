package cityaq

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/shp"
	"github.com/ctessum/geom/index/rtree"
)

// country returns the name and geometry of the country that the
// given city is nearest to.
func (c *CityAQ) country(cityName string) (*country, error) {
	c.loadCountries()
	cityGeom, err := c.geojsonGeometry(cityName)
	if err != nil {
		return nil, err
	}
	var ctry *country
	var isect float64
	for _, cI := range c.countries.SearchIntersect(cityGeom.Bounds()) {
		c := cI.(*country)
		iSect := cityGeom.Intersection(c)
		if iSect != nil {
			if ia := iSect.Area(); ia > isect {
				isect = ia
				ctry = c
			}
		}
	}
	if ctry == nil {
		return nil, fmt.Errorf("couldn't match country to city %s", cityName)
	}
	return ctry, nil
}

type country struct {
	geom.Polygon
	Name string `shp:"CNTRY_NAME"`
}

func (c *CityAQ) loadCountries() {
	c.loadCountriesOnce.Do(func() {
		c.countries = rtree.NewTree(25, 50)
		d, err := shp.NewDecoder(filepath.Join(c.SpatialConfig.SrgShapefileDirectory, "Countries_WGS84.shp"))
		if err != nil {
			panic(err)
		}
		for {
			var row country
			if more := d.DecodeRow(&row); !more {
				break
			}
			c.countries.Insert(&row)
		}
	})
}

// countryOrBuffer returns the smaller of the country that the city is located
// in or a circular buffer with area equivalent to the average area among
// the US NERC regions, intersected with the country.
// The buffer area is calculated from the shapefile downloaded from
// https://www.eia.gov/maps/layer_info-m.php on 10/22/2019.
// The shapefile data is from 12/5/2016.
//
// NERC grid statistics:
// Mean:91.6756666667
// StdDev:93.0167200239
// Sum:825.081
// Min:13.505
// Max:332.24
// N:9.0
// CV:1.01462823676
// Number of unique values:9
// Range:318.735
// Median:54.09
func (c *CityAQ) countryOrGridBuffer(cityName string) (*country, error) {
	const (
		area      = 91.6756666667 // degrees^2
		radius    = 5.40196918017 // sqrt(area/pi) [degrees]
		nSegments = 20            // Number of segments for the buffer.
	)
	// Make sure we don't go outside of the InMAP grid boundary.
	gridBounds := &geom.Bounds{
		Min: geom.Point{X: -178, Y: -88},
		Max: geom.Point{X: 178, Y: 88},
	}
	ctry, err := c.country(cityName)
	if err != nil {
		return nil, err
	}
	if ctry.Area() <= area {
		return ctry, nil
	}
	// Country is too big, use buffer.
	cityGeom, err := c.geojsonGeometry(cityName)
	if err != nil {
		return nil, err
	}
	return &country{
		Polygon: cityGeom.Centroid().Buffer(radius, nSegments).Intersection(ctry).Intersection(gridBounds).(geom.Polygon),
		Name:    "buffer",
	}, nil
}

// egugridEmissions returns whether the given sourceType should
// be allocated to the country or electric grid buffer rather than a city.
func egugridEmissions(sourceType string) bool {
	return strings.HasSuffix(sourceType, "_egugrid")
}

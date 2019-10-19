package cityaq

import (
	"path/filepath"
	"strings"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/shp"
	"github.com/ctessum/geom/index/rtree"
)

// cityCountry returns the name and geometry of the country that the
// given city is nearest to.
func (c *CityAQ) cityCountry(cityName string) (*country, error) {
	c.loadCountries()
	cityGeom, err := c.geojsonGeometry(cityName)
	if err != nil {
		return nil, err
	}
	countryI := c.countries.NearestNeighbor(cityGeom.Centroid())
	return countryI.(*country), nil
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

// nationalEmissions returns whether the given sourceType should
// be allocated to the a country rather than a city.
func nationalEmissions(sourceType string) bool {
	return strings.HasSuffix(sourceType, "_national")
}

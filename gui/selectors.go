package gui

import (
	"context"
	"errors"
	"strconv"
	"syscall/js"

	rpc "github.com/ctessum/cityaq/cityaqrpc"
)

const defaultSelectorText = "-- select an option --"

func updateSelector(doc, selector js.Value, values []interface{}, text []string) {
	selector.Set("innerHTML", "")
	option := doc.Call("createElement", "option")
	option.Set("disabled", true)
	option.Set("selected", true)
	option.Set("hidden", true)
	option.Set("text", defaultSelectorText)
	selector.Call("appendChild", option)
	for i, value := range values {
		option := doc.Call("createElement", "option")
		option.Set("value", value)
		option.Set("text", text[i])
		selector.Call("appendChild", option)
	}
}

// updateCitySelector updates the options of cities.
func (c *CityAQ) updateCitySelector(ctx context.Context) {
	if c.citySelector == js.Undefined() {
		c.citySelector = c.doc.Call("getElementById", "citySelector")
	}
	cities, err := c.Cities(ctx, &rpc.CitiesRequest{})
	if err != nil {
		c.logError(err)
		return
	}
	names := make([]interface{}, len(cities.Names))
	for i, n := range cities.Names {
		names[i] = n
	}
	updateSelector(c.doc, c.citySelector, names, cities.Names)
}

// updateImpactTypeSelector updates the options of impacts.
func (c *CityAQ) updateImpactTypeSelector() {
	if c.impactTypeSelector == js.Undefined() {
		c.impactTypeSelector = c.doc.Call("getElementById", "impactTypeSelector")
	}
	updateSelector(c.doc, c.impactTypeSelector, []interface{}{1}, []string{"Emissions"})
}

// updateEmissionSelector updates the options of emissions available.
func (c *CityAQ) updateEmissionSelector() {
	if c.emissionSelector == js.Undefined() {
		c.emissionSelector = c.doc.Call("getElementById", "emissionSelector")
	}
	values := make([]interface{}, len(rpc.Emission_value)-1)
	text := make([]string, len(rpc.Emission_value)-1)
	for i := 1; i < len(rpc.Emission_value); i++ {
		n := rpc.Emission_name[int32(i)]
		values[i-1] = i
		text[i-1] = n
	}
	updateSelector(c.doc, c.emissionSelector, values, text)
}

// updateSourceTypeSelector updates the options of source types available.
func (c *CityAQ) updateSourceTypeSelector() {
	if c.sourceTypeSelector == js.Undefined() {
		c.sourceTypeSelector = c.doc.Call("getElementById", "sourceTypeSelector")
	}
	updateSelector(c.doc, c.sourceTypeSelector,
		[]interface{}{"electric_gen_egugrid", "population", "residential", "commercial", "industrial", "builtup", "roadways", "railways", "waterways", "bus_routes", "airports", "agricultural"},
		[]string{"electric_gen_egugrid", "population", "residential", "commercial", "industrial", "builtup", "roadways", "railways", "waterways", "bus_routes", "airports", "agricultural"})
}

func (c *CityAQ) updateSelectors(ctx context.Context) {
	c.updateCitySelector(ctx)
	c.updateImpactTypeSelector()
	c.updateEmissionSelector()
	c.updateSourceTypeSelector()
}

func selectorValue(selector js.Value) (string, error) {
	v := selector.Get("value").String()
	if v == defaultSelectorText {
		return v, incompleteSelectionError
	}
	return v, nil
}

func (c *CityAQ) citySelectorValue() (string, error) {
	return selectorValue(c.citySelector)
}

func (c *CityAQ) impactTypeSelectorValue() (rpc.ImpactType, error) {
	v, err := selectorValue(c.impactTypeSelector)
	if err != nil {
		return -1, err
	}
	vInt, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return -1, err
	}
	return rpc.ImpactType(vInt), nil
}

func (c *CityAQ) emissionSelectorValue() (rpc.Emission, error) {
	v, err := selectorValue(c.emissionSelector)
	if err != nil {
		return -1, err
	}
	vInt, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return -1, err
	}
	return rpc.Emission(vInt), nil
}

func (c *CityAQ) sourceTypeSelectorValue() (string, error) {
	return selectorValue(c.sourceTypeSelector)
}

type selections struct {
	cityName   string
	impactType rpc.ImpactType
	sourceType string
	emission   rpc.Emission
}

var incompleteSelectionError = errors.New("incomplete selection")

func (c *CityAQ) selectorValues() (s *selections, err error) {
	s = new(selections)
	s.cityName, err = c.citySelectorValue()
	if err != nil {
		return
	}

	s.emission, err = c.emissionSelectorValue()
	if err != nil {
		return
	}

	s.impactType, err = c.impactTypeSelectorValue()
	if err != nil {
		return
	}

	s.sourceType, err = c.sourceTypeSelectorValue()
	if err != nil {
		return
	}

	return
}

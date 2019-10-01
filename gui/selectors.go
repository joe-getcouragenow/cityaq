package gui

import (
	"context"
	"errors"
	"syscall/js"

	rpc "github.com/ctessum/cityaq/cityaqrpc"
	"google.golang.org/grpc/grpclog"
)

func updateSelector(doc, selector js.Value, values []interface{}, text []string) {
	selector.Set("innerHTML", "")
	option := doc.Call("createElement", "option")
	option.Set("disabled", true)
	option.Set("selected", true)
	option.Set("hidden", true)
	option.Set("text", "-- select an option --")
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
	c.cityNames = make(map[string]string)
	cities, err := c.Cities(ctx, &rpc.CitiesRequest{})
	if err != nil {
		panic(err)
		grpclog.Println(err)
		return
	}
	paths := make([]interface{}, len(cities.Paths))
	for i, p := range cities.Paths {
		c.cityNames[p] = cities.Names[i]
		paths[i] = p
	}
	updateSelector(c.doc, c.citySelector, paths, cities.Names)
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
	updateSelector(c.doc, c.sourceTypeSelector, []interface{}{"roads"}, []string{"roads"})
}

func (c *CityAQ) updateSelectors(ctx context.Context) {
	c.updateCitySelector(ctx)
	c.updateImpactTypeSelector()
	c.updateEmissionSelector()
	c.updateSourceTypeSelector()
}

func selectorValue(selector js.Value) js.Value {
	return selector.Get("value")
}

func (c *CityAQ) citySelectorValue() (string, error) {
	v := selectorValue(c.citySelector)
	if v == js.Null() {
		return "", incompleteSelectionError
	}
	return v.String(), nil
}

func (c *CityAQ) impactTypeSelectorValue() (impactType, error) {
	v := selectorValue(c.impactTypeSelector)
	if v == js.Null() {
		return -1, incompleteSelectionError
	}
	return impactType(v.Int()), nil
}

func (c *CityAQ) emissionSelectorValue() (rpc.Emission, error) {
	v := selectorValue(c.emissionSelector)
	if v == js.Null() {
		return -1, incompleteSelectionError
	}
	return rpc.Emission(v.Int()), nil
}

func (c *CityAQ) sourceTypeSelectorValue() (string, error) {
	v := selectorValue(c.sourceTypeSelector)
	if v == js.Null() {
		return "", incompleteSelectionError
	}
	return v.String(), nil
}

type impactType int

const (
	emission impactType = iota + 1
)

type selections struct {
	cityPath   string
	cityName   string
	impactType impactType
	sourceType string
	emission   rpc.Emission
}

var incompleteSelectionError = errors.New("incomplete selection")

func (c *CityAQ) selectorValues() (s *selections, err error) {
	s = new(selections)
	s.cityPath, err = c.citySelectorValue()
	if err != nil {
		return
	}
	s.cityName = c.cityNames[s.cityPath]

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

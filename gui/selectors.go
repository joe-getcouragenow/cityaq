package gui

import (
	"context"
	"fmt"
	"strconv"
	"syscall/js"

	rpc "github.com/ctessum/cityaq/cityaqrpc"
	"google.golang.org/grpc/grpclog"
)

func updateSelector(doc, selector js.Value, values, text []string) {
	selector.Set("innerHTML", "")
	for i, value := range values {
		option := doc.Call("createElement", "option")
		option.Set("value", value)
		option.Set("text", text[i])
		selector.Call("appendChild", option)
	}
}

// updateCitySelector updates the options of cities.
func (c *CityAQ) updateCitySelector(ctx context.Context) {
	if c.citySelector == js.Null() {
		c.citySelector = c.doc.Call("getElementById", "citySelector")
	}
	cities, err := c.Cities(ctx, &rpc.CitiesRequest{})
	if err != nil {
		grpclog.Println(err)
		return
	}
	updateSelector(c.doc, c.citySelector, cities.Paths, cities.Names)
}

// updateImpactTypeSelector updates the options of impacts.
func (c *CityAQ) updateImpactTypeSelector() {
	if c.impactTypeSelector == js.Null() {
		c.impactTypeSelector = c.doc.Call("getElementById", "impactTypeSelector")
	}
	updateSelector(c.doc, c.impactTypeSelector, []string{"Emissions"}, []string{"Emissions"})
}

// updateEmissionSelector updates the options of emissions available.
func (c *CityAQ) updateEmissionSelector() {
	if c.emissionSelector == js.Null() {
		c.emissionSelector = c.doc.Call("getElementById", "emissionSelector")
	}
	values := make([]string, len(rpc.Emission_value)-1)
	text := make([]string, len(rpc.Emission_value)-1)
	for i := 1; i < len(rpc.Emission_value); i++ {
		n := rpc.Emission_name[int32(i)]
		values[i-1] = fmt.Sprint(i)
		text[i-1] = n
	}
	updateSelector(c.doc, c.emissionSelector, values, text)
}

func (c *CityAQ) updateSelectors(ctx context.Context) {
	c.updateCitySelector(ctx)
	c.updateImpactTypeSelector()
	c.updateEmissionSelector()
}

func selectorValue(selector js.Value) (value, text string) {
	options := selector.Get("options")
	selectedIndex := selector.Get("selectedIndex").Int()
	selection := options.Index(selectedIndex)
	value = selection.Get("value").String()
	text = selection.Get("text").String()
	return value, text
}

func (c *CityAQ) citySelectorValue() (value, text string) {
	return selectorValue(c.citySelector)
}

func (c *CityAQ) impactTypeSelectorValue() (value, text string) {
	return selectorValue(c.impactTypeSelector)
}

func (c *CityAQ) emissionSelectorValue() (value, text string) {
	return selectorValue(c.emissionSelector)
}

type selections struct {
	cityName, cityPath string
	impactType         string
	emission           rpc.Emission
}

func (c *CityAQ) selectorValues() (*selections, error) {
	s := new(selections)
	s.cityName, s.cityPath = c.citySelectorValue()
	s.impactType, _ = c.impactTypeSelectorValue()
	emissionIntStr, _ := c.emissionSelectorValue()
	emissionInt, err := strconv.ParseInt(emissionIntStr, 10, 64)
	if err != nil {
		return nil, err
	}
	s.emission = rpc.Emission(emissionInt)
	return s, nil
}

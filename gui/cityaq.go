// Package gui implements a web interface for the CityAQ service.
package gui

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"syscall/js"

	rpc "github.com/ctessum/cityaq/cityaqrpc"
	"github.com/ctessum/go-leaflet/plugin/glify"
	grpcwasm "github.com/johanbrandhorst/grpc-wasm"
)

type CityAQ struct {
	rpc.CityAQClient
	doc                  js.Value
	citySelector         js.Value
	impactTypeSelector   js.Value
	emissionSelector     js.Value
	sourceTypeSelector   js.Value
	legendDiv            js.Value
	mapDiv               js.Value
	mapboxMap            js.Value
	cityLayer, dataLayer js.Value
	mapColors            *glify.Shapes
	grid                 struct {
		geometry js.Value
		gridCity string
		gridType rpc.ImpactType
	}
}

// NewCityAQ returns a CityAQ client. Typically one would
// use DefaultConnection() as the input connection.
func NewCityAQ(conn *grpcwasm.ClientConn) *CityAQ {
	c := &CityAQ{
		CityAQClient: rpc.NewCityAQClient(conn),
		doc:          js.Global().Get("document"),
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		c.updateSelectors(context.Background())
		wg.Done()
	}()
	go func() {
		c.loadMap()
		wg.Done()
	}()
	wg.Wait()
	return c
}

// DefaultConnection is the connection the CityAQ client should
// use if it is running in a browser.
func DefaultConnection() *grpcwasm.ClientConn {
	doc := js.Global().Get("document")
	url, err := url.Parse(doc.Get("baseURI").String())
	if err != nil {
		panic(err)
	}
	cc, err := grpcwasm.Dial(url.Scheme + "://" + url.Host)
	if err != nil {
		panic(err)
	}
	return cc
}

// Monitor updates the map whenever a selector changes.
func (c *CityAQ) Monitor() {
	cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			c.doc.Call("getElementById", "error").Set("innerHTML", "")
			sel, err := c.selectorValues()
			if err != nil {
				if err == incompleteSelectionError {
					return
				}
				c.logError(err)
				return
			}
			c.updateMap(context.TODO(), sel)
		}()
		return nil
	})
	for _, s := range []js.Value{c.citySelector, c.impactTypeSelector, c.emissionSelector, c.sourceTypeSelector} {
		s.Call("addEventListener", "change", cb)
	}
	c.citySelector.Call("addEventListener", "change", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			cityName, err := c.citySelectorValue()
			if err != nil {
				c.logError(err)
				return
			}
			c.MoveMap(context.TODO(), cityName)
		}()
		return nil
	}))
}

func (c *CityAQ) startLoading() {
	go func() {
		for _, id := range []string{"loading", "loading_text", "loading_icon"} {
			c.doc.Call("getElementById", id).Set("hidden", false)
		}
		for _, s := range []js.Value{c.citySelector, c.impactTypeSelector, c.emissionSelector, c.sourceTypeSelector} {
			s.Set("disabled", true)
		}
	}()
}
func (c *CityAQ) stopLoading() {
	go func() {
		for _, id := range []string{"loading", "loading_text", "loading_icon"} {
			c.doc.Call("getElementById", id).Set("hidden", true)
		}
		for _, s := range []js.Value{c.citySelector, c.impactTypeSelector, c.emissionSelector, c.sourceTypeSelector} {
			s.Set("disabled", false)
		}
	}()
}

func (c *CityAQ) logError(err error) {
	s := fmt.Sprintf("<p class=\"text-danger\">%s</p>", err.Error())
	c.doc.Call("getElementById", "error").Set("innerHTML", s)
}

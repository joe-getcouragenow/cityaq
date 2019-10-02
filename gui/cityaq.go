// Package gui implements a web interface for the CityAQ service.
package gui

import (
	"context"
	"net/url"
	"sync"
	"syscall/js"

	rpc "github.com/ctessum/cityaq/cityaqrpc"
	"github.com/ctessum/go-leaflet"
	"github.com/ctessum/go-leaflet/plugin/glify"
	grpcwasm "github.com/johanbrandhorst/grpc-wasm"
	"google.golang.org/grpc/grpclog"
)

type CityAQ struct {
	rpc.CityAQClient
	doc                js.Value
	citySelector       js.Value
	impactTypeSelector js.Value
	emissionSelector   js.Value
	sourceTypeSelector js.Value
	legendDiv          js.Value
	mapDiv             js.Value
	leafletMap         *leaflet.Map
	mapColors          *glify.Shapes
	baseMapLayer       *leaflet.TileLayer
	grid               struct {
		geometry js.Value
		gridCity string
		gridType impactType
	}
	cityNames map[string]string
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
		grpclog.Println(err)
		panic(err)
	}
	cc, err := grpcwasm.Dial(url.Scheme + "://" + url.Host)
	if err != nil {
		grpclog.Println(err)
		panic(err)
		return nil
	}
	return cc
}

// Monitor updates the map whenever a selector changes.
func (c *CityAQ) Monitor() {
	cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			sel, err := c.selectorValues()
			if err != nil {
				if err == incompleteSelectionError {
					return
				}
				grpclog.Println(err)
				panic(err)
				return
			}
			c.updateMap(context.TODO(), sel)
		}()
		return nil
	})
	for _, s := range []js.Value{c.citySelector, c.impactTypeSelector, c.emissionSelector, c.sourceTypeSelector} {
		s.Call("addEventListener", "change", cb)
	}
}

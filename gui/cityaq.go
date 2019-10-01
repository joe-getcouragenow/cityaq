package gui

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"syscall/js"

	rpc "github.com/ctessum/cityaq/cityaqrpc"
	"github.com/ctessum/go-leaflet"
	"github.com/ctessum/go-leaflet/plugin/glify"
	"google.golang.org/grpc"
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
	grid               struct {
		geometry js.Value
		gridCity string
		gridType impactType
	}
}

// NewCityAQ returns a CityAQ client. Typically one would
// use DefaultConnection() as the input connection.
func NewCityAQ(conn *grpc.ClientConn) *CityAQ {
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
func DefaultConnection() *grpc.ClientConn {
	doc := js.Global().Get("document")
	url, err := url.Parse(doc.Get("baseURI").String())
	if err != nil {
		grpclog.Println(err)
		panic(err)
	}
	cc, err := grpc.Dial(fmt.Sprintf("%s://%s", url.Scheme, url.Host))
	if err != nil {
		grpclog.Println(err)
		return nil
	}
	return cc
}

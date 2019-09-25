package gui

import (
	"context"
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

func (c *CityAQ) updateCitySelector(ctx context.Context) {
	cities, err := c.Cities(ctx, &rpc.CitiesRequest{})
	if err != nil {
		grpclog.Println(err)
		return
	}
	updateSelector(c.doc, c.citySelector, cities.Paths, cities.Names)
}

package gui

import (
	"fmt"
	"net/url"
	"syscall/js"

	rpc "github.com/ctessum/cityaq/cityaqrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

type CityAQ struct {
	rpc.CityAQClient
	doc                js.Value
	citySelector       js.Value
	impactTypeSelector js.Value
	emissionSelector   js.Value
}

// NewCityAQ returns a CityAQ client. Typically one would
// use DefaultConnection() as the input connection.
func NewCityAQ(conn *grpc.ClientConn) *CityAQ {
	return &CityAQ{
		CityAQClient: rpc.NewCityAQClient(conn),
		doc:          js.Global().Get("document"),
	}
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

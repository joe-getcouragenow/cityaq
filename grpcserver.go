package cityaq

import (
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"

	"github.com/ctessum/cityaq/cityaqrpc"
)

// GRPCSServer is a GRPC server for city AQ simulation data.
type GRPCServer struct {
	CityAQ
	grpcServer *grpcweb.WrappedGrpcServer
}

// NewGRPCServer creates a new GRPC server for c.
func NewGRPCServer(c *CityAQ) *GRPCServer {
	gs := grpc.NewServer()
	cityaqrpc.RegisterCityAQServer(gs, c)
	s := new(GRPCServer)
	s.grpcServer = grpcweb.WrapServer(gs)
	return s
}

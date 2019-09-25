package cityaq

// Install the code generation dependencies.
// go get -u github.com/golang/protobuf/protoc-gen-go
// go get -u github.com/johanbrandhorst/grpc-wasm/protoc-gen-wasm

// Generate the gRPC client/server code. (Information at https://grpc.io/docs/quickstart/go.html)
//go:generate protoc cityaq.proto --go_out=plugins=grpc:cityaqrpc

// go get github.com/golang/mock/gomock
// go install github.com/golang/mock/mockgen

// Generate mock client
//go: generate mockgen github.com/ctessum/cityaq/cityaqrpc CityAQClient > cityaqrpc/mock_cityaqrpc/caqmock.go

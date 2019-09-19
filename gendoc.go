package cityaq

// Install the code generation dependencies.
// go get -u github.com/golang/protobuf/protoc-gen-go
// go get -u github.com/johanbrandhorst/grpc-wasm/protoc-gen-wasm

// Generate the gRPC client/server code. (Information at https://grpc.io/docs/quickstart/go.html)
//go:generate protoc cityaq.proto --go_out=plugins=grpc:cityaqrpc

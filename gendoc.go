package cityaq

// Manually download executables:
// go get github.com/golang/mock/mockgen@latest
// go get -u github.com/go-bindata/go-bindata/...
// Download protoc from: https://github.com/protocolbuffers/protobuf/releases/tag/v3.11.4
//    and put in $PATH

// Install the code generation dependencies.
//go:generate bash -c "rm ~/go/bin/protoc-gen-go"
//go:generate go get -u github.com/golang/protobuf/protoc-gen-go@v1.4.2

// Generate the gRPC client/server code. (Information at https://grpc.io/docs/quickstart/go.html)
//go:generate protoc cityaq.proto --go_out=plugins=grpc:cityaqrpc
//go:generate go build ./internal/addtags
//go:generate ./addtags -file=cityaqrpc/cityaq.pb.go -tags=!js

//go:generate go get -u github.com/johanbrandhorst/grpc-wasm/protoc-gen-wasm@v0.0.0-20180613181153-d79a93c3901e
//go:generate bash -c "mv ~/go/bin/protoc-gen-wasm ~/go/bin/protoc-gen-go"

// Generate the gRPC WASM client/server code. (Information at https://grpc.io/docs/quickstart/go.html)
//go:generate protoc cityaq.proto --go_out=plugins=grpc:cityaqrpc
//go:generate ./addtags -file=cityaqrpc/cityaq.wasm.pb.go -tags=js
//go:generate rm addtags

// go get github.com/golang/mock/gomock
// go install github.com/golang/mock/mockgen

// Generate mock client
//go:generate bash -c "GOOS=js GOARCH=wasm mockgen -source=cityaqrpc/cityaq.wasm.pb.go > cityaqrpc/mock_cityaqrpc/caqmock.go"

// Build the GUI with Go WASM
//go:generate bash -c "GOOS=js GOARCH=wasm go build -o ./gui/html/cityaq.wasm ./gui/cmd/main.go"

// Compress the GUI.
//go:generate go run internal/compress/main.go
//go:generate rm ./gui/html/cityaq.wasm

// Running next line will overwrite changes to index.html:
// //go:generate bash -c "cp $DOLLAR(go env GOROOT)/misc/wasm/wasm_exec.html ./gui/html/index.html"

//go:generate bash -c "cp $DOLLAR(go env GOROOT)/misc/wasm/wasm_exec.js ./gui/html/wasm_exec.js"

// Bin the GUI data.
//go:generate go-bindata --pkg cityaq -o assets.go gui/html/

# Path to the project.
ROOT_DIR:=			$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
LIB_FSPATH:=		$(GOPATH)/src/github.com/ctessum/cityaq
GO_OS:=				$(shell go env GOOS)
GO_ARCH:=			$(shell go env GOARCH)

#include git.mk

print:
	@echo
	@echo ROOT_DIR : 	$(ROOT_DIR)
	@echo LIB_FSPATH : 	$(LIB_FSPATH)
	@echo GO_OS : 		$(GO_OS)
	@echo GO_ARCH : 	$(GO_ARCH)
	@echo


dep:
	# Cross platform get correct Protoc
	go build -o download-protoc ./internal/download 
	./download-protoc
	# will create a folder called "lib-protoc"
	# Need to use this on your global path during compile. E.G. "$(LIB_FSPATH)/lib-protoc/bin/"


### BUILD
# Change PROTOC_FSPATH to match your OS. The location depends on the dep stage and where it puts the lib-protoc. I expect it will vary for Windows ?
PROTOC_FSPATH=$(LIB_FSPATH)/lib-protoc/bin/
export PATH:=$(PROTOC_FSPATH):$(PATH)

build: dep
	@echo -- Server deps --
	go get -u github.com/golang/protobuf/protoc-gen-go@v1.4.2

	# Generate the gRPC client/server code. (Information at https://grpc.io/docs/quickstart/go.html)
	@echo -- Server: Generate the gRPC client/server code --
	protoc cityaq.proto --go_out=plugins=grpc:cityaqrpc
	go build ./internal/addtags
	./addtags -file=cityaqrpc/cityaq.pb.go -tags=!js

	# Generate the gRPC WASM client/server code. (Information at https://grpc.io/docs/quickstart/go.html)
	@echo -- Server: Generate the gRPC WASM client/server code --
	# replace protoc-gen-go with the WASM version.
	go get -u github.com/johanbrandhorst/grpc-wasm/protoc-gen-wasm@v0.0.0-20180613181153-d79a93c3901e
	mv $(GOPATH)/bin/protoc-gen-wasm $(GOPATH)/bin/protoc-gen-go

	protoc cityaq.proto --go_out=plugins=grpc:cityaqrpc
	./addtags -file=cityaqrpc/cityaq.wasm.pb.go -tags=js
	rm addtags

	@echo

	@echo -- Client dep --
	go get github.com/golang/mock/gomock 
	go install github.com/golang/mock/mockgen
	@echo 

	@echo -- Client WASM build --
	GOOS=js GOARCH=wasm go build -o ./gui/html/cityaq.wasm ./gui/cmd/main.go
	ls -al gui/html
	@echo

	@echo -- Client Compression --
	go run internal/compress/main.go
	rm -f gui/html/cityaq.wasm
	ls -al gui/html
	@echo

	@echo -- Client Update WASM runners --
	@echo GOROOT:  $(GOROOT)
	# NOTE: This is where the index.html gets changed, and so is commented out, as we dont want to stomp on the index.html
	#cp $(GOROOT)/misc/wasm/wasm_exec.html $(LIB_FSPATH)/gui/html/index.html
	cp $(GOROOT)/misc/wasm/wasm_exec.js ./gui/html/wasm_exec.js
	ls -al gui/html
	@echo

	@echo -- Client Pack into bindata --
	go-bindata --pkg cityaq -o assets.go gui/html/
	@echo


gen:
	# OLD way

	go generate ./...

server-run:
	go run ./cmd .
	# https://localhost:10000/






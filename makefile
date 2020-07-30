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
	cd $(LIB_FSPATH) && go build -o download-protoc ./internal/download 
	cd $(LIB_FSPATH) && ./download-protoc
	# will create a folder called "lib-protoc"
	# Need to use this on your global path during compile. E.G. "$(LIB_FSPATH)/lib-protoc/bin/"


### BUILD
# Change PROTOC_FSPATH to match your OS. The location depends on the dep stage and where it puts the lib-protoc. I expect it will vary for Windows ?
PROTOC_FSPATH=$(LIB_FSPATH)/lib-protoc/bin/
export PATH:=$(PROTOC_FSPATH):$(PATH)

build: dep
	@echo -- Server deps --
	cd $(LIB_FSPATH) && go get -u github.com/golang/protobuf/protoc-gen-go@v1.4.2

	# Generate the gRPC client/server code. (Information at https://grpc.io/docs/quickstart/go.html)
	@echo -- Server: Generate the gRPC client/server code --
	cd $(LIB_FSPATH) && protoc cityaq.proto --go_out=plugins=grpc:cityaqrpc
	cd $(LIB_FSPATH) && go build ./internal/addtags
	cd $(LIB_FSPATH) && ./addtags -file=cityaqrpc/cityaq.pb.go -tags=!js

	# Generate the gRPC WASM client/server code. (Information at https://grpc.io/docs/quickstart/go.html)
	@echo -- Server: Generate the gRPC WASM client/server code --
	# replace protoc-gen-go with the WASM version.
	cd $(LIB_FSPATH) && go get -u github.com/johanbrandhorst/grpc-wasm/protoc-gen-wasm@v0.0.0-20180613181153-d79a93c3901e
	cd $(LIB_FSPATH) && mv $(GOPATH)/bin/protoc-gen-wasm $(GOPATH)/bin/protoc-gen-go

	cd $(LIB_FSPATH) && protoc cityaq.proto --go_out=plugins=grpc:cityaqrpc
	cd $(LIB_FSPATH) && ./addtags -file=cityaqrpc/cityaq.wasm.pb.go -tags=js
	cd $(LIB_FSPATH) && rm addtags

	@echo

	@echo -- Client dep --
	cd $(LIB_FSPATH) && go get github.com/golang/mock/gomock 
	cd $(LIB_FSPATH) && go install github.com/golang/mock/mockgen
	@echo 

	@echo -- Client WASM build --
	cd $(LIB_FSPATH) && GOOS=js GOARCH=wasm go build -o ./gui/html/cityaq.wasm ./gui/cmd/main.go
	cd $(LIB_FSPATH)/gui/html && ls -al
	@echo

	@echo -- Client Compression --
	cd $(LIB_FSPATH) && go run internal/compress/main.go
	cd $(LIB_FSPATH) && rm -f ./gui/html/cityaq.wasm
	cd $(LIB_FSPATH)/gui/html && ls -al
	@echo

	@echo -- Client Update WASM runners --
	@echo GOROOT:  $(GOROOT)
	# NOTE: This is where the index.html gets changed, and so is commented out, as we dont want to stomp on the index.html
	#cp $(GOROOT)/misc/wasm/wasm_exec.html $(LIB_FSPATH)/gui/html/index.html
	cp $(GOROOT)/misc/wasm/wasm_exec.js $(LIB_FSPATH)/gui/html/wasm_exec.js
	cd $(LIB_FSPATH)/gui/html && ls -al
	@echo

	@echo -- Client Pack into bindata --
	go-bindata --pkg cityaq -o $(LIB_FSPATH)/assets.go $(LIB_FSPATH)/gui/html/
	@echo


gen:
	# OLD way

	cd $(LIB_FSPATH) && go generate ./...

server-run:
	cd $(LIB_FSPATH) && go run ./cmd .
	# https://127.0.0.1:1000/






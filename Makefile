BIN=bin
BIN_NAME=dell-provisioner

all: build

fmt:
	go fmt ./...

$(BIN):
	mkdir -p $(BIN)

vendor:
	glide install -v

install:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go install

$(BIN)/$(BIN_NAME)-linux-amd64 build: $(BIN) $(shell find . -name "*.go")
	env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-extldflags "-static"' -o $(BIN)/$(BIN_NAME)-linux-amd64 .

$(BIN)/$(BIN_NAME)-darwin-amd64 darwin: $(BIN) $(shell find . -name "*.go")
	env CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -a -ldflags '-extldflags "-static"' -o $(BIN)/$(BIN_NAME)-darwin-amd64 .

clean:
	go clean -i
	rm -rf $(BIN)
	rm -rf vendor

.PHONY: all fmt clean build darwin vendor

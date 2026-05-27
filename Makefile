BINARY=cloudmux
BUILD_DIR=bin

.PHONY: build test test-verbose lint fmt vet clean

build:
	CGO_ENABLED=0 go build -o $(BUILD_DIR)/$(BINARY) ./cmd/cloudmux

test:
	go test ./...

test-verbose:
	go test -v ./...

lint:
	golangci-lint run

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -rf $(BUILD_DIR)

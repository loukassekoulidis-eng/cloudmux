BINARY=cloudmux
BUILD_DIR=bin

.PHONY: build test test-verbose lint fmt vet clean build-tray

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/cloudmux

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

build-tray:
	go build -o $(BUILD_DIR)/cloudmux-tray ./cmd/cloudmux-tray

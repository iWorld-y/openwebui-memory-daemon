.PHONY: build test tidy clean

BINARY_NAME ?= owui-memory-daemon
OUTPUT_DIR  ?= output

build:
	@mkdir -p "$(OUTPUT_DIR)"
	CGO_ENABLED=0 go build -o "$(OUTPUT_DIR)/$(BINARY_NAME)" ./cmd/daemon

test:
	go test ./...

tidy:
	GOPROXY=https://proxy.golang.org,direct go mod tidy

clean:
	rm -rf "$(OUTPUT_DIR)"


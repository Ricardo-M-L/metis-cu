# metis-cu — computer-use MCP server for metis (and any MCP-speaking client).

.PHONY: build test fmt clean install vet

BIN := metis-cu

build:
	go build -o $(BIN) .

install:
	go install .

test:
	go test ./... -count=1

fmt:
	gofmt -w .

vet:
	go vet ./...

clean:
	rm -f $(BIN)
	rm -rf dist/

# Cross-compile (cgo-required platforms must build on native host or via
# Docker buildx — pure-go builds aren't possible because robotgo +
# screenshot need platform SDKs).
.PHONY: dist-darwin dist-linux dist-windows
dist-darwin:
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build -o dist/$(BIN)-darwin-arm64 .
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -o dist/$(BIN)-darwin-amd64 .

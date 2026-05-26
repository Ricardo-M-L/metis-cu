# metis-cu — computer-use MCP server for metis (and any MCP-speaking client).

.PHONY: build test fmt clean install vet

BIN := metis-cu

build:
	go build -o $(BIN) .

install:
	go install .
	@# Mirror the freshly-built binary to ~/.local/bin so users whose
	@# mcp.toml pins the absolute path (e.g.
	@#   command = "/Users/X/.local/bin/metis-cu")
	@# pick up the new code without manually copying. metis's `/cu enable`
	@# writes that pinned path; before this mirror step every `make install`
	@# silently left mcp.toml pointing at a stale binary, masking
	@# cu-side fixes for hours (session 41040b / 87e366f post-mortem,
	@# 2026-05-26 — tier + OCR fixes "didn't work" because the actual
	@# cu spawned was a 4-day-old binary).
	@if [ -d "$$HOME/.local/bin" ]; then \
		install -m 0755 "$$HOME/go/bin/metis-cu" "$$HOME/.local/bin/metis-cu" && \
		echo "  installed $$HOME/.local/bin/metis-cu (mirrored from $$HOME/go/bin)"; \
	fi

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

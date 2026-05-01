# What

<!-- One paragraph: what the PR changes. Focus on the user-visible /
behavior change, not the file list. -->

# Why

<!-- One paragraph: the motivation. Bug? Feature gap? API parity with
Anthropic spec? Link the issue if there is one. -->

# Test

<!-- "go test ./..." is green. List anything you tested manually. -->

# API surface

<!-- If you added or changed a tool: -->

- [ ] Tool name appears in `pkg/tools/tools.go` `allToolNames`
- [ ] Schema lives in `docs/api-spec.md`
- [ ] Name matches Anthropic's `mcp__computer-use__*` namespace, OR is
      explicitly prefixed `metis_*` to avoid collision

# Checklist

- [ ] `go test ./...` is green
- [ ] `go vet ./...` is clean
- [ ] `gofmt -l .` returns nothing
- [ ] Touched platform code is in `pkg/platform/platform_<goos>.go`
      (no `runtime.GOOS` switches inside a single file)

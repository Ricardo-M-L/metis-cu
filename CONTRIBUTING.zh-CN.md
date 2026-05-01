# 为 metis-cu 贡献代码

感谢关注。本文记录维护者（以及任何贡献者）遵循的约定。

英文版：[CONTRIBUTING.md](CONTRIBUTING.md)。

## 项目结构

```
main.go                   stdio MCP server 入口
pkg/server/               协议适配器（JSON-RPC framing + 工具分发）
pkg/tools/                24 个工具注册 + 分级路由
pkg/platform/             OS 抽象层
  platform.go             Platform interface + tier 枚举（read/click/full）
  platform_darwin.go      macOS — CGEvent / screencapture / NSPasteboard
  platform_other.go       非 darwin stub（Sprint 4 补 Linux + Windows）
docs/api-spec.md          完整 schema 参考，对齐 Anthropic
```

`pkg/platform` 下是 OS 专属代码。新增平台只需放一个 `platform_<goos>.go` 文件；
`platform.go` 里的 interface 是契约。

## API 稳定性

24 个工具名称和参数形态**有意**镜像 Anthropic 的 `mcp__computer-use__*` 命名空间。
**不要重命名或改动。** 如果 Anthropic 发新工具，跟着镜像；
如果我们需要 Anthropic 没有的工具，前缀 `metis_*` 隔离，不要污染共享命名空间。

## 构建与测试

```sh
go build ./...                          # 本地二进制 ./metis-cu
go test -count=1 -timeout 60s ./...     # 单元测试
go vet ./...
```

需要 cgo（robotgo + screenshot）。跨平台 release 构建用 `make dist-darwin` 等命令——
纯 Go 交叉编译跑不通。

提交前检查清单：

1. `go test ./...` 全绿
2. `go vet ./...` 无报错
3. `gofmt -l .` 输出为空
4. 改动有测试，或者注释里写明"手动测试因为 <原因>"
5. 新增工具名同时出现在 `pkg/tools/tools.go` 的 allToolNames 和 `docs/api-spec.md` —
   保持一致

## 代码风格

- **注释解释为什么，不解释做什么。** 命名好的函数不需要 docstring 复述函数体。
- **不要多段 docstring。** 一行短注释为上限。
- **平台相关代码放 platform_<goos>.go** — 不要在单个文件里用 `runtime.GOOS` 分支，
  让 build tag 选实现。
- **Tier 检查集中在 `pkg/tools/gate.go`**。不要把 frontmost-app 检查散布到每个工具 handler。
- **除非用户明确要求，否则不用 emoji。**

## 报告安全问题

请不要在公开 issue 里讨论安全发现。用 [SECURITY.md](SECURITY.md) 链接的私下 advisory 表单。

# AGENTS.md — owui-memory-daemon

## 功能概述

一个 Go 语言编写的守护进程，定期将 OpenWebUI 中的对话压缩为“记忆”（每日/每周/每月），并将所有记忆快照保存到 Git 仓库中。可通过 cron 运行，也可使用 `-run` 标志执行单次运行。

## 命令

```bash
make build          # CGO_ENABLED=0 构建 → output/owui-memory-daemon
make test           # go test ./...
make tidy           # go mod tidy（使用 proxy.golang.org）
make clean          # rm -rf output/

# 单次运行（需要 config.yaml）
go run ./cmd/daemon -config ./config.yaml -run daily
go run ./cmd/daemon -config ./config.yaml -run daily -day 2026-04-09
go run ./cmd/daemon -config ./config.yaml -run weekly
go run ./cmd/daemon -config ./config.yaml -run monthly

# 守护进程模式（cron 调度器）
go run ./cmd/daemon -config ./config.yaml

# Docker
docker compose up -d --build
```

## 架构

```
cmd/daemon/main.go          — 入口，装配所有依赖，启动 cron
internal/
  memoryops/
    application/             — 用例：DailySummarizer, WeeklyCompressor, MonthlyCompressor
      ports.go               — OWUIPort, LLMPort, RetryPort, SnapshotPort, LoggerPort
      prompts.go             — LLM 提示词模板（中文，强制使用绝对时间）
      truncate.go            — 安全的 UTF-8 头尾截断
    domain/
      kind.go                — Kind 枚举：Daily(📋), Weekly(📦), Monthly(📅), None
      keys.go                — 前缀解析 + ISO 周数计算
  snapshotting/
    application/             — Snapshotter：转储 memories.json → git add/commit/push
  infrastructure/
    owui/client.go           — OpenWebUI REST 适配器（实现 OWUIPort）
    llm/openai.go            — OpenAI 兼容的 chat/completions 适配器（实现 LLMPort）
    gitrepo/repo.go          — 通过 exec 调用的本地 git 适配器（实现 RepoPort）
    retry/retry.go           — 支持可配置策略的通用重试逻辑
    config/config.go         — YAML 配置加载器
    logx/logx.go             — 带文件输出的 slog 封装
```

## 核心约定

- **记忆前缀约定**：守护进程仅处理带有前缀 `📋 `（每日）、`📦 `（每周）、`📅 `（每月）的记忆。用户创建的无前缀记忆永远不会被修改或删除。
- **定向日总结**：支持 `-run daily -day YYYY-MM-DD` 定向整理某一天的对话；`-day` 仅能与 `-run daily` 一起使用。
- **压缩层级**：每日 → 每周（删除每日记忆）→ 每月（删除每周记忆）。每次写入后都会调用快照。
- **LLM 提示词**：必须使用绝对时间表达式；提示词设计中禁止使用相对时间（如“昨天/最近/上周”）。
- **端口/适配器模式**：业务逻辑依赖 `application/ports.go` 中定义的接口；基础设施适配器实现这些接口。通过 `var _ Port = (*Adapter)(nil)` 进行编译时检查。
- **无外部框架**：纯 Go 标准库 + `robfig/cron/v3` + `gopkg.in/yaml.v3`。不使用 gorilla/chi/gin 等框架。

## 测试

- `go test ./...` — 仅包含单元测试；无集成测试框架。
- 测试文件与代码放在一起（例如 `retry/retry_test.go`、`llm/openai_test.go`）。
- 没有 mock 或测试辅助包；测试直接构造输入数据。
- 运行测试不需要任何外部服务（OWUI、LLM、Git）— 测试仅覆盖纯逻辑。

## 配置

- `config.yaml` 位于项目根目录（可通过 `-config` 标志指定路径）。
- 包含 OWUI 基础 URL 和 API 密钥、LLM 端点和模型、cron 调度计划、Git 仓库路径和远程地址、日志设置。
- `owui-memories-backup/` 已被 gitignore — 它是一个运行时克隆目录，快照会存放于此。

## Go 版本

- `go 1.26.1`（在 `go.mod` 中指定）。Dockerfile 使用 `golang:1.26-alpine`。

# owui-memory-daemon

一个常驻守护进程：按日/周/月把 OpenWebUI 的对话压缩为 Memories，并在每次变更后把全量 Memories 快照到 Git 仓库中。

## 快速开始

1) 复制并修改 `config.yaml`

2) 本地运行（一次性触发）

```bash
go run ./cmd/daemon -config ./config.yaml -run daily
go run ./cmd/daemon -config ./config.yaml -run daily -day 2026-04-09
go run ./cmd/daemon -config ./config.yaml -run weekly
go run ./cmd/daemon -config ./config.yaml -run monthly
```

其中 `-day` 仅可与 `-run daily` 搭配，格式必须为绝对日期 `YYYY-MM-DD`，用于定向整理某一天的对话记忆。

3) 常驻运行（按 cron）

```bash
go run ./cmd/daemon -config ./config.yaml
```

## Docker

```bash
docker compose up -d --build
```

## 重要约定

- 只处理带前缀的总结类 Memory：日 `📋` / 周 `📦` / 月 `📅`
- 无前缀的用户手动记忆不触碰
- Prompt 强制绝对时间表达，禁止“昨天/最近”等相对时间

## 目标

把当前 `internal/` 由“按技术关注点平铺（owui/llm/git/task/...）”改为“按 DDD 边界上下文 + Clean Architecture 分层”的结构，达到：

- `domain` 只表达业务规则与不变量，不依赖外部（HTTP/Git/文件系统）
- `application` 只编排用例，依赖 domain + ports（接口）
- `infrastructure` 只实现 ports（OWUI/LLM/Git/FS/日志/重试等）
- `cmd` 只做 wiring（组装依赖 + cron），不放业务判断

本次重构不改变现有功能与行为约定（见 `docs/2026-04-09-owui-memory-daemon.md`）。

---

## 边界上下文（Bounded Context）

### MemoryOps（记忆域）

关注“总结类 Memory”的语义与规则：

- Kind：日 `📋`、周 `📦`、月 `📅`、手动（无前缀）
- Key 解析与归属：日 `YYYY-MM-DD`、周 `YYYY-Www`（ISOWeek）、月 `YYYY-MM`（月合并目标为上月）
- 合并/删除规则：只触碰总结类 Memory；删除失败不回滚写入；空集合则跳过
- Prompt 规则：强制绝对时间表达

### ConversationIngest（对话摄取域）

关注“从 OWUI 抽取某时间窗的对话集合”的语义：

- 日任务：以某天的 [start,end) 窗口筛选 chats（按 updated_at），拉取 chat transcript

（为了控制改动面，本次实现允许其与 MemoryOps 的 daily 用例合并，后续可再拆出独立上下文。）

### Snapshotting（快照域/支撑域）

关注“把当前 Memories 状态落盘并推送 Git”这一用例语义：

- 每次任务成功后触发快照
- commit 可能因“nothing to commit”失败，应视为成功
- push 失败不阻断主流程（由调用者决定日志策略）

---

## 推荐目录结构

```
internal/
  memoryops/
    domain/
      kind.go
      keys.go
    application/
      ports.go
      prompts.go
      truncate.go
      daily_summarize.go
      weekly_compress.go
      monthly_compress.go
  snapshotting/
    application/
      ports.go
      snapshot_and_push.go
  infrastructure/
    config/
    logx/
    retry/
    owui/
    llm/
    gitrepo/
cmd/
  daemon/
    main.go
```

依赖方向：

- `cmd` → `application` → `domain`
- `infrastructure` 实现 `application` 中声明的 ports（接口），但不反向依赖 domain 的实现细节

---

## 迁移顺序（保证可编译、可回滚）

1. **新增新包（不删旧包）**
   - 建立 `internal/memoryops/{domain,application}`、`internal/snapshotting/application`、`internal/infrastructure/*`
   - 在 application 中声明 ports 接口（OWUI/LLM/Repo/FS/Clock/Logger/Retry）

2. **迁移 domain：Memory kind/key 解析**
   - 将原 `internal/memory/format.go` 的解析/格式化迁移到 `memoryops/domain`
   - 保持函数语义不变，先让旧代码可通过适配层调用新 domain（或直接更新引用）

3. **迁移 usecases：daily/weekly/monthly**
   - 将原 `internal/task/*.go` 迁移到 `memoryops/application`
   - 用 ports 替代对 `internal/owui`、`internal/llm` 的直接依赖
   - 复用既有 prompt/truncate 逻辑（迁入 application）

4. **迁移 snapshotting 用例**
   - 将 `internal/snapshot/snapshot.go` 迁移到 `snapshotting/application`
   - 把 Git/FS/OWUI 细节改为 ports

5. **迁移 infrastructure**
   - 把现有实现（OWUI HTTP、LLM HTTP、Git repo、config、retry、logx）迁到 `internal/infrastructure/*`
   - 让它们实现对应 ports

6. **更新 `cmd/daemon/main.go`**
   - main 只做 wiring：读取 config、创建 infra 客户端、创建 usecase、注册 cron、调用 usecase + snapshotting

7. **删除旧包**
   - 删除 `internal/task`、`internal/memory`、`internal/owui`、`internal/llm`、`internal/git`、`internal/snapshot` 等旧路径
   - 更新/搬迁对应单元测试

8. **验证**
   - `gofmt ./...`
   - `go test ./...`


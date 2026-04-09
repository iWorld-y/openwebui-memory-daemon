## 完整设计：owui-memory-daemon

### 1. 配置文件 (`config.yaml`)

```yaml
openwebui:
  base_url: "http://localhost:3000"
  api_key: "sk-xxx"           # Bearer token

llm:
  base_url: "http://localhost:11434/v1"  # Ollama / OpenAI-compatible
  api_key: ""
  model: "qwen3:32b"
  max_tokens: 4096

schedule:
  daily: "0 0 * * *"       # 每天 00:00
  weekly: "0 0 * * 0"      # 每周日 00:00
  monthly: "0 0 1 * *"     # 每月 1 日 00:00

git:
  repo_path: "./owui-memories-backup"
  remote: "origin"
  branch: "main"
  author_name: "owui-memory-daemon"
  author_email: "daemon@localhost"

log:
  level: "info"
  path: "./logs/daemon.log"
```

### 2. 三层合并数据流

```
日常对话（OpenWebUI）
       │
       ▼
┌─────────────────┐
│  DailySummarize  │  每天 00:00
│  拉取前一天对话    │
│  LLM 总结        │
│  写入 Memory      │
│  📋 2026-04-09   │
└────────┬────────┘
         │ 7天后
         ▼
┌─────────────────┐
│  WeeklyCompress  │  每周日 00:00
│  拉取本周日总结    │
│  LLM 压缩合并     │
│  删除日总结       │
│  写入周总结       │
│  📦 2026-W14     │
└────────┬────────┘
         │ 约4周后
         ▼
┌─────────────────┐
│  MonthlyCompress │  每月1日 00:00
│  拉取上月周总结    │
│  LLM 压缩合并     │
│  删除周总结       │
│  写入月总结       │
│  📅 2026-04      │
└────────┬────────┘
         │
         ▼
   月总结永久保留
```

### 3. Memory 内容格式规范

```
日: 📋 2026-04-09 对话总结：用户讨论了XXX，决定YYY，关注ZZZ...
周: 📦 2026-W14 周总结：本周核心主题包括AAA和BBB，用户在CCC方面有新进展...
月: 📅 2026-04 月总结：本月整体聚焦于DDD，完成了EEE，开始探索FFF...
```

**绝对时间约束**：LLM Prompt 中明确要求——
- ✅ `2026-04-09 讨论了...`
- ✅ `4月9日至4月11日期间...`
- ❌ `昨天讨论了`
- ❌ `最近在关注`

**非总结类 Memory 原样保留**（用户手动添加的、无前缀的），compress 不触碰。

### 4. 每个任务详细流程

#### DailySummarize

```
1. GET /api/v1/chats/list  → 获取对话列表
2. 遍历，筛选 updated_at 在前一天的对话
3. 对每个对话 GET /api/v1/chats/{id} → 提取当天消息
4. 按对话拼接内容，构造 Prompt:
   "以下是用户在 2026-04-09 的所有对话内容，请提取关键信息生成总结。
    要求：使用绝对时间，不使用相对时间。..."
5. 调用 LLM API → 获得总结文本
6. POST /api/v1/memories/add  {"content": "📋 2026-04-09 对话总结：..."}
7. 全量快照 → git commit + push
```

#### WeeklyCompress

```
1. GET /api/v1/memories/  → 拉取全量 Memories
2. 筛选 prefix="📋" 且日期在本周的日总结
3. 拼接所有日总结内容，构造 Prompt:
   "以下是本周的每日对话总结，请压缩合并为一条周总结。
    保留关键信息，消除冗余，使用绝对时间。..."
4. 调用 LLM API → 获得周总结
5. POST /api/v1/memories/add  {"content": "📦 2026-W14 周总结：..."}
6. 逐条 DELETE /api/v1/memories/{id}  删除已合并的日总结
7. 全量快照 → git commit + push
```

#### MonthlyCompress

```
1. GET /api/v1/memories/  → 拉取全量 Memories
2. 筛选 prefix="📦" 且属于上月的周总结
3. 拼接所有周总结，构造 Prompt:
   "以下是上月每周总结，请压缩合并为一条月总结。
    保留关键信息，消除冗余，使用绝对时间。..."
4. 调用 LLM API → 获得月总结
5. POST /api/v1/memories/add  {"content": "📅 2026-04 月总结：..."}
6. 逐条 DELETE /api/v1/memories/{id}  删除已合并的周总结
7. 全量快照 → git commit + push
```

### 5. Git 快照逻辑

```go
// 每次操作后执行
func snapshotAndPush(client *OpenWebUIClient, gitRepo *GitRepo) error {
    // 1. 拉取全量 Memories
    memories := client.GetAllMemories()
    
    // 2. 写入 JSON 文件
    data, _ := json.MarshalIndent(memories, "", "  ")
    os.WriteFile(filepath.Join(gitRepo.path, "memories.json"), data, 0644)
    
    // 3. git add + commit + push
    gitRepo.Add("memories.json")
    gitRepo.Commit(fmt.Sprintf("snapshot: %s", time.Now().Format("2006-01-02_15:04:05")))
    gitRepo.Push()
    
    return nil
}
```

仓库结构极简：
```
owui-memories-backup/
  memories.json          # 全量快照（覆盖写入）
  README.md              # 仓库说明
```

### 6. 错误处理

| 场景 | 策略 |
|------|------|
| LLM 调用失败 | 重试 3 次，间隔 10s/30s/60s，仍失败则跳过本次任务，记日志 |
| OpenWebUI API 不可达 | 同上重试策略 |
| Git push 失败 | 本地 commit 保留，下次操作时再次尝试 push |
| 单条对话内容过长 | 截断到 LLM context window 的 80%，保留开头和结尾 |
| 无前一天对话 | DailySummarize 跳过，写日志 "无对话" |
| 无本周/月总结可合并 | 同上跳过 |
| **删除日/周总结失败** | 不回滚周/月总结的写入，下次 compress 时会再尝试清理 |

### 7. 项目结构

```
owui-memory-daemon/
├── cmd/
│   └── daemon/
│       └── main.go           # 入口：加载配置 → 启动 cron → 常驻
├── internal/
│   ├── config/
│   │   └── config.go         # config.yaml 解析
│   ├── owui/
│   │   └── client.go         # OpenWebUI API 客户端
│   ├── llm/
│   │   └── client.go         # LLM API 客户端
│   ├── task/
│   │   ├── daily.go          # DailySummarize
│   │   ├── weekly.go         # WeeklyCompress
│   │   └── monthly.go        # MonthlyCompress
│   ├── memory/
│   │   └── format.go         # Memory 前缀解析、时间格式工具
│   └── git/
│       └── repo.go           # Git 操作封装
├── config.yaml               # 配置文件模板
├── Dockerfile
├── docker-compose.yaml
├── go.mod
└── README.md
```

### 8. Docker 部署

```yaml
# docker-compose.yaml
services:
  owui-memory-daemon:
    build: .
    volumes:
      - ./config.yaml:/app/config.yaml
      - ./owui-memories-backup:/app/owui-memories-backup
    restart: always
```

---

**整体设计到这里。** 你看有什么要调整的，或者确认没问题我写设计文档保存？
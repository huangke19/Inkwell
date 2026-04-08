# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 启动与运行

```bash
# 开发启动（需要 GROQ_API_KEY 环境变量）
GROQ_API_KEY=... go run main.go

# 构建二进制（运行时需在项目根目录执行，模板从工作目录读取）
go build -o inkwell .

# 代码检查
go vet ./...
```

环境变量：
- `GROQ_API_KEY`（必填）— Groq API 密钥
- `DB_PATH`（可选）— SQLite 文件路径，默认 `ewords.db`
- `PORT`（可选）— 监听端口，默认 `9090`

无测试文件，无需运行测试命令。

## 架构概述

Inkwell 是英语单词记忆 Web 应用：**添加单词 → AI 生成解释 → SRS 间隔复习**。

技术栈：Go 标准库 HTTP + SQLite（`go-sqlite3`）+ HTMX（前端无构建步骤）。

### 包职责

| 包 | 职责 |
|---|---|
| `config` | 从环境变量加载配置 |
| `db` | 打开 SQLite，embed 执行全部迁移 SQL |
| `models` | 所有 SQL 操作函数（无 ORM） |
| `srs` | 纯函数：计算下次复习时间 |
| `freq` | embed 词频 CSV，查词的 CEFR/频率/推荐度 |
| `handlers` | HTTP 处理器 |

### 关键设计决策

**handlers/db.go 包装层**：`handlers` 通过 `handlers/db.go` 调用 `models` 包，该文件用小写函数名做一层转发，目的是避免 handler 直接引用大写导出名，同时保持 `models` 对外可用。修改数据层时两处都需要看。

**HTMX 双模式**：每个 handler 都检查 `isHTMX(r)`（检测 `HX-Request` header），同一路由对普通请求返回完整页面，对 HTMX 请求用 `Renderer.Fragment()` 只返回 partial。

**模板系统**：`Page()` 和 `Fragment()` 都是每次请求时 `ParseFiles`（按需加载），不缓存，避免多页面 `{{define}}` 命名冲突。`Page()` 始终加载全部 partials 供内嵌 HTMX 调用使用。

**AI 两级模型**：主调用用 `llama-3.1-8b-instant`（快），JSON 解析失败时自动降级到 `llama-3.3-70b-versatile` 重试。上下文翻译始终用 70b（质量优先）。

**"已掌握"阈值**：`interval_days >= 30`，由 `MarkMastered()` 直接写入固定值，SRS 倍增上限也是 30 天。

**数据库迁移**：SQLite 不支持 `ADD COLUMN IF NOT EXISTS`，`db.applyMigration()` 通过捕获 `"duplicate column name"` 错误来实现幂等迁移。新增字段需创建新迁移文件并在 `db/db.go` 的 slice 中追加。

**外部 API 接口**：`POST /words` 同时支持 `application/json`（浏览器扩展调用）和 `application/x-www-form-urlencoded`（表单提交），根据 `Content-Type` header 区分响应格式。

**词频评级优先级**：`EnsureAI()` 中先查本地 `freq` 词表（原形和原词），命中则用本地数据，未命中才用 AI 返回的 CEFR 等级生成评级。

### 复习流程

```
GET /review          → 展示第一个到期单词
  用户点「不记得」  → POST /review/{id}/forgot（加载 AI 解释）
                     → POST /review/{id}/confirm-forgot（标记 incorrect，更新 SRS）
  用户点「记得」    → POST /review/{id}/remember（展示英文输入框）
                     → POST /review/{id}/explain（AI 判断英文解释 → 标记 correct/incorrect）
GET /review/next     → HTMX 加载下一道题
```

### 数据库

单文件 SQLite，WAL 模式，`MaxOpenConns=1`。AI 字段拆存为三列 JSON 字符串（`ai_meaning`、`ai_examples`、`ai_scenarios`）+ 一列纯文本（`ai_memory_tip`），解析在 `handlers/ai.go` 的 `parseAI()` 中进行。

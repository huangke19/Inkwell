# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 启动与运行

```bash
# 开发启动（需要 GROQ_API_KEY 环境变量）
go run main.go

# 服务固定运行在 9090 端口
PORT=9090 go run main.go

# 构建二进制
go build -o inkwell .
```

环境变量：
- `GROQ_API_KEY`（必填）— Groq API 密钥
- `DB_PATH`（可选）— SQLite 文件路径，默认 `ewords.db`
- `PORT`（可选）— 监听端口，固定为 `9090`

## 架构概述

Inkwell 是一个英语单词记忆 Web 应用，核心功能：**添加单词 → AI 生成解释 → SRS 间隔复习**。

技术栈：Go 标准库 HTTP + SQLite + HTMX（前端无构建步骤）。

### 数据流

```
用户添加单词
  → models.CreateWord() 写入 SQLite
  → GET /words/{id}/ai 触发 EnsureAI()
    → 检查缓存（ai_generated_at，30天有效）
    → 调用 Groq API（llama-3.3-70b-versatile）
    → AI 字段以 JSON 字符串存入 words 表

用户复习
  → GET /review 取最早到期的单词（next_review_at）
  → POST /review/{id} 提交答案
    → checkAnswer() 模糊匹配中文释义
    → srs.Next() 计算新间隔（倍增法，上限30天；答错重置为1天）
    → 更新 words.interval_days / next_review_at / repetitions
    → 写入 review_logs
```

### 包职责

| 包 | 职责 |
|---|---|
| `config` | 从环境变量加载配置 |
| `db` | 打开 SQLite，embed 执行 `001_init.sql` 初始化表结构 |
| `models` | 所有 SQL 操作函数（无 ORM） |
| `srs` | 纯函数：根据答题结果计算下次复习时间 |
| `handlers` | HTTP 处理器；`ai.go` 封装 Groq 调用和缓存逻辑 |

### 模板系统

- `Renderer.Page()` — 完整页面：`layout.html` + 具体页面 + partials
- `Renderer.Fragment()` — HTMX 局部更新：只渲染指定 partial，不套 layout
- 模板按需 ParseFiles，避免多页面 `{{define}}` 命名冲突
- 模板从工作目录读取，二进制运行时需在项目根目录执行

### 数据库

单文件 SQLite，WAL 模式，`MaxOpenConns=1`。表结构通过 embed 的 `db/migrations/001_init.sql` 在启动时幂等创建（`CREATE TABLE IF NOT EXISTS`）。AI 字段以 JSON 字符串存储在 `words` 表，解析在 `handlers/ai.go` 的 `parseAI()` 中进行。

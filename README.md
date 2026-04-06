# Inkwell

Inkwell 是一个面向中文母语者的英语单词记忆 Web 应用。

它的核心流程是：收集单词 -> AI 生成解释 -> 按间隔重复算法复习 -> 用英文解释反向验证是否真正理解。

项目当前采用 Go 标准库 HTTP + SQLite + HTMX，前端没有构建步骤，服务固定运行在 `9090` 端口。

## 功能概览

- AI 解释生成：调用 Groq 模型生成音标、英文释义、中文释义、例句、使用场景、记忆技巧和 CEFR 等级
- AI 结果缓存：已生成的解释会写回 SQLite，后续优先读取缓存
- 双模型回退：默认使用 `llama-3.1-8b-instant`，解析失败时自动回退到 `llama-3.3-70b-versatile`
- 词形归一化：识别常见时态、复数、比较级、最高级和大量不规则词形，优先解释原形词条
- 实用性评级：结合本地词表和 CEFR 等级，生成频率与推荐等级
- 三个词库视图：全部单词、生词库、已掌握
- 搜索、排序、分页：按评级或创建时间排序，每页 15 条，可直接跳页
- 来源回看：保存原始上下文、页面标题和来源 URL
- SRS 复习：正确时间隔倍增，错误时回到 1 天，上限 30 天
- 英文解释验收：用户用英文解释单词，AI 判断是否真正理解
- 浏览器扩展：支持选词后直接加入词库，也支持右键菜单收词
- 移动端适配：手机端单词列表保留三列，详情和复习页做了小屏优化

## 项目结构

```text
.
├── main.go                  # HTTP 入口与路由注册
├── config/                  # 环境变量读取
├── db/                      # SQLite 初始化与迁移
├── handlers/                # 页面、AI、复习相关 HTTP 处理
├── models/                  # SQL 操作
├── srs/                     # 间隔重复算法
├── freq/                    # 词频/CEFR 词表与词形归一化
├── templates/               # HTML 模板与 HTMX 片段
├── static/                  # 样式和 htmx
├── extension/               # Chrome 扩展
├── start.sh                 # 后台启动脚本
└── stop.sh                  # 停止脚本
```

## 技术栈

- Go 1.26.1
- SQLite (`github.com/mattn/go-sqlite3`)
- HTMX
- Groq Chat Completions API
- Chrome Extension Manifest V3

## 运行要求

- Go 1.26.1 或更高版本
- 一个可用的 Groq API Key
- macOS / Linux 下可直接使用 `start.sh` 与 `stop.sh`

## 快速开始

1. 克隆仓库

```bash
git clone https://github.com/huangke19/Inkwell.git
cd Inkwell
```

2. 准备环境变量

```bash
cp .env.example .env
```

编辑 `.env`，至少填入：

```bash
GROQ_API_KEY=your_groq_api_key_here
DB_PATH=ewords.db
```

3. 启动服务

```bash
./start.sh
```

启动成功后访问：

```text
http://localhost:9090
```

4. 停止服务

```bash
./stop.sh
```

## 直接开发运行

如果你不想使用脚本，也可以直接运行：

```bash
export GROQ_API_KEY=your_groq_api_key_here
go run main.go
```

构建二进制：

```bash
go build -o inkwell .
```

检查编译是否正常：

```bash
go build ./...
```

## 环境变量

| 变量 | 必填 | 默认值 | 说明 |
|---|---|---|---|
| `GROQ_API_KEY` | 是 | 无 | Groq API Key |
| `DB_PATH` | 否 | `ewords.db` | SQLite 文件路径 |

说明：

- 服务端口在代码中固定为 `9090`
- 即使设置 `PORT`，当前版本也不会生效

## 使用流程

### 1. 添加单词

可以通过两种方式添加：

- Web 表单：输入单词，可选填写原句上下文
- 浏览器扩展：选中文本后直接发送到 Inkwell

添加后会进入单词详情页，并自动请求 AI 解释。

### 2. 查看词条详情

详情页会展示：

- 词形原形信息
- 原始上下文
- 上下文中文翻译
- 音标
- 英文释义
- 中文释义
- 例句
- 使用场景
- 记忆技巧
- 词频/CEFR/推荐等级

如果已保存来源信息，还可以看到页面标题和原文链接。

### 3. 复习

复习页逻辑如下：

1. 取最早到期的单词
2. 用户选择“记得”或“不记得”
3. 不记得：先展示 AI 解释，再确认标记错误
4. 记得：用户用英文解释该词
5. AI 判断解释是否抓住核心含义
6. 根据结果更新复习间隔并写入 `review_logs`

### 4. 掌握状态

- 复习间隔达到 30 天的单词进入“已掌握”
- 也可以在词条详情页或列表页直接手动标记为已掌握

## SRS 规则

当前算法非常直接：

- 首次答对：间隔变为 1 天
- 之后每次答对：间隔翻倍
- 最大间隔：30 天
- 答错：重置为 1 天，重复次数归零

这套规则定义在 `srs/Next()` 中。

## AI 生成策略

### 解释生成

- 输入：单词、词形信息、可选上下文
- 输出：严格 JSON
- 默认模型：`llama-3.1-8b-instant`
- JSON 解析失败时回退：`llama-3.3-70b-versatile`

英文释义的目标是：

- 简洁
- 准确
- 只保留核心含义和最常见用法
- 避免写成百科式长文

### 上下文翻译

如果存在原始上下文，但 AI 主响应没有给出翻译，系统会再单独请求一次翻译并回写数据库。

### 解释判断

复习时用户提交英文解释后，AI 会返回：

- 是否正确
- 是否过于含糊
- 一段简短中文反馈

## 评级系统

项目内置一个词表：

- 文件：`freq/wordlist.csv`
- 用途：将单词映射到 CEFR 等级，再映射到“高频 / 中频 / 低频 / 罕见”以及推荐级别

如果本地词表中查不到该词，则退回使用 AI 返回的 CEFR 等级。

当前推荐策略：

- `A1 / A2` -> 高频 -> 强烈推荐
- `B1 / B2` -> 中频 -> 建议掌握
- `C1` -> 低频 -> 选择性记
- `C2 / 未知` -> 罕见 -> 可以跳过

## 数据存储

数据库使用单文件 SQLite。

主要表：

- `words`
- `review_logs`

`words` 表保存：

- 单词本身
- 原始上下文
- 来源 URL / 来源标题
- AI 解释 JSON
- 复习状态
- 评级信息
- 创建和更新时间

迁移文件：

- `db/migrations/001_init.sql`
- `db/migrations/002_add_ratings.sql`
- `db/migrations/003_add_source.sql`

数据库初始化特点：

- `journal_mode = WAL`
- `MaxOpenConns = 1`
- 启动时自动执行迁移
- 已存在列时忽略重复列错误

## Web 路由

### 页面路由

- `GET /`：全部单词
- `GET /unmastered`：生词库
- `GET /mastered`：已掌握
- `GET /words/add`：添加单词表单
- `GET /words/{id}`：单词详情
- `GET /review`：复习入口

### 词条操作

- `POST /words`：创建单词
- `DELETE /words/{id}`：删除单词
- `POST /words/{id}/master`：标记已掌握
- `GET /words/{id}/ai`：获取或刷新 AI 解释

### 复习操作

- `GET /review/next`：下一个复习单词
- `POST /review/{id}/forgot`：进入“不记得”流程
- `POST /review/{id}/confirm-forgot`：确认答错并更新 SRS
- `POST /review/{id}/remember`：进入英文解释输入
- `POST /review/{id}/explain`：提交英文解释并获取判断结果

## 浏览器扩展

目录：`extension/`

能力：

- 选中英文单词后弹出小气泡，一键加入词库
- 右键菜单直接收词
- 自动携带原文上下文、页面标题和来源链接
- 成功后自动打开词条详情页

当前扩展默认连接：

```text
http://localhost:9090
```

因此它适合在你自己的电脑浏览器里与本地 Inkwell 配合使用。

如果你把网站部署到其他地址，扩展中的以下位置也需要同步调整：

- `extension/content.js`
- `extension/popup.js`
- `extension/manifest.json` 中的 `host_permissions`

## 手机访问

如果网站运行在你的 Mac 上，推荐通过 `Tailscale` 从手机访问：

1. 在 Mac 和手机上安装 Tailscale
2. 使用同一个账号登录
3. 在 Mac 上启动 Inkwell
4. 用手机访问 Mac 的 Tailscale 地址，例如：`http://100.x.x.x:9090`

这种方式不需要端口转发，也不需要把服务直接暴露到公网。

## 移动端界面说明

当前移动端策略：

- 单词列表保留三列：单词、评级、核心释义
- 详情页保持完整展示
- 复习页和添加页针对小屏做了紧凑化处理
- 导航栏在小屏下折叠为菜单

## 前端说明

- 无 React / Vue / 前端构建链
- 页面主要通过服务端模板渲染
- 局部交互依赖 HTMX
- 样式集中在 `static/style.css`

这种结构的优点是简单直接，但复杂交互能力会弱于前后端分离方案。

## 已知限制

- 没有用户系统，默认单机使用
- 没有访问鉴权，不适合直接裸露到公网
- CORS 目前是全开放的（`Access-Control-Allow-Origin: *`）
- 扩展默认只连接本机 `localhost:9090`
- 复习算法目前是简单倍增模型，不是 SM-2 之类的成熟变体
- AI 返回依赖外部模型，偶尔可能出现响应慢或格式不稳定

## 适合的使用场景

- 自己日常积累阅读中遇到的生词
- 在浏览器里边读边收词
- 用英文解释来检查是否真的理解
- 从手机通过 Tailscale 远程访问自己的词库

## 不适合的使用场景

- 多用户共享使用
- 直接暴露到公网给陌生人访问
- 对稳定性、审计、权限控制要求很高的生产系统

## 开发建议

如果你后续继续扩展，比较自然的方向是：

- 增加登录与访问控制
- 为扩展支持自定义服务地址
- 增加导出/备份功能
- 给复习加入更多统计维度
- 引入更成熟的 SRS 模型

## 许可证

当前仓库未声明开源许可证；如需公开分发，建议补充 LICENSE。

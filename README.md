# Inkwell

英语单词记忆 Web 应用。添加单词后由 AI 自动生成详细解释，通过间隔重复算法（SRS）安排复习，复习时用英文解释单词，AI 判断是否正确。

## 功能

- **AI 解释**：调用 Groq API（llama-3.3-70b-versatile）生成音标、英文释义、中文释义、例句、使用场景和记忆技巧，缓存 30 天
- **实用性评级**：基于内嵌词表（~1500 词）和 CEFR 标准，自动评定词汇等级（A1–C2）和实用性（强烈推荐 / 建议掌握 / 选择性记 / 可以跳过）
- **间隔复习**：SRS 算法，答对后复习间隔翻倍（上限 30 天），答错重置为 1 天
- **英文验证**：复习时用英文解释单词，AI 判断是否理解正确并给出反馈

## 技术栈

Go 标准库 HTTP + SQLite + HTMX，无前端构建步骤。

## 快速开始

**依赖**：Go 1.21+，[Groq API Key](https://console.groq.com)

```bash
git clone https://github.com/huangke19/Inkwell.git
cd Inkwell

cp .env.example .env
# 编辑 .env，填入 GROQ_API_KEY

./start.sh
# 访问 http://localhost:8081
```

**停止服务**

```bash
./stop.sh
```

## 环境变量

| 变量 | 必填 | 默认值 | 说明 |
|---|---|---|---|
| `GROQ_API_KEY` | ✓ | — | Groq API 密钥 |
| `PORT` | | `8080` | 监听端口 |
| `DB_PATH` | | `ewords.db` | SQLite 文件路径 |

## 复习流程

1. 看到单词，选择「记得」或「不记得」
2. **不记得**：展示完整 AI 解释，确认后标记答错
3. **记得**：用英文解释该词，AI 判断理解是否正确并给出反馈

package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"inkwell/freq"
	"inkwell/models"
)

type AIExplanation struct {
	PrimaryMeaning     string       `json:"primary_meaning"`
	PartOfSpeech       string       `json:"part_of_speech"`
	Phonetic           string       `json:"phonetic"`
	EnglishDef         string       `json:"english_def"`
	ContextTranslation string       `json:"context_translation"`
	Definitions        []Definition `json:"definitions"`
	Examples           []Example    `json:"examples"`
	Scenarios          []string     `json:"scenarios"`
	MemoryTip          string       `json:"memory_tip"`
	CEFRLevel          string       `json:"cefr_level"`
}

type Definition struct {
	ZH   string `json:"zh"`
	Note string `json:"note"`
}

type Example struct {
	EN string `json:"en"`
	ZH string `json:"zh"`
}

const systemPrompt = `你是一个专业的英语词汇教学助手，帮助中文母语者学习英语单词。
用户会提供一个英语单词，以及可选的原始语境句子。
你需要返回严格的 JSON 格式，不要添加任何 markdown 代码块标记，直接输出 JSON。
解释使用中文，例句使用英文，中文翻译紧跟在例句后。
内容要准确、简洁、实用，贴近真实使用场景。`

const groqURL = "https://api.groq.com/openai/v1/chat/completions"

var groqAPIKey string

func InitProviders(groqKey, _ string) {
	groqAPIKey = groqKey
}

func callGroq(messages []map[string]any, maxTokens int) (string, error) {
	return callGroqWithModel("llama-3.1-8b-instant", messages, maxTokens)
}

func callGroqFallback(messages []map[string]any, maxTokens int) (string, error) {
	return callGroqWithModel("llama-3.3-70b-versatile", messages, maxTokens)
}

func callGroqWithModel(model string, messages []map[string]any, maxTokens int) (string, error) {
	reqBody := map[string]any{
		"model":       model,
		"messages":    messages,
		"temperature": 0,
		"max_tokens":  maxTokens,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, groqURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+groqAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Groq 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(respBytes, &errResp) == nil && errResp.Error.Message != "" {
			return "", fmt.Errorf("Groq 错误 %d: %s", resp.StatusCode, errResp.Error.Message)
		}
		return "", fmt.Errorf("Groq 错误 %d", resp.StatusCode)
	}

	var r struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBytes, &r); err != nil || len(r.Choices) == 0 {
		return "", fmt.Errorf("解析 Groq 响应失败: %s", string(respBytes))
	}
	return r.Choices[0].Message.Content, nil
}

func fetchAIExplanation(_ string, word *models.Word) (*AIExplanation, error) {
	form := freq.Normalize(word.Word)
	userPrompt := fmt.Sprintf("单词：%s\n", word.Word)
	if form.Changed {
		userPrompt += fmt.Sprintf("推测原形：%s\n词形说明：%s\n", form.Base, form.Kind)
		userPrompt += "请优先解释原形词条，并简要说明当前词形在原文中的语法作用。\n"
	}
	if word.Context != "" {
		userPrompt += fmt.Sprintf("用户遇到该词的语境：%s\n", word.Context)
	}
	userPrompt += `
请按以下 JSON 结构返回（不加代码块标记，直接输出 JSON）。
重要：所有字符串值内部不得使用英文双引号，需要引用时改用「」或()代替。
{
  "primary_meaning": "最核心的中文释义（10字以内）",
  "part_of_speech": "adj/n/v/adv/prep（英文缩写）",
  "phonetic": "/IPA音标，如 /ʌbˈɪkwɪtəs/，使用美式发音/",
	"english_def": "简洁的全英文释义，控制在60-90个英文单词左右，只保留核心含义、最常见用法，以及与近义词的关键区别，不要写成百科介绍。",
	"context_translation": "如果提供了原始语境句子，这里填自然中文翻译；没有语境则留空",
  "definitions": [
    {"zh": "中文释义1", "note": "适用场景或语体（可选）"},
    {"zh": "中文释义2", "note": ""}
  ],
  "examples": [
    {"en": "例句1（结合用户语境，如有）", "zh": "中文翻译1"},
    {"en": "例句2（日常口语场景）", "zh": "中文翻译2"},
    {"en": "例句3（正式书面场景）", "zh": "中文翻译3"},
    {"en": "例句4（学术或专业场景）", "zh": "中文翻译4"},
    {"en": "例句5（新闻报道风格）", "zh": "中文翻译5"},
    {"en": "例句6（对话场景）", "zh": "中文翻译6"},
    {"en": "例句7（描述性场景）", "zh": "中文翻译7"},
    {"en": "例句8（否定或反向用法，如适用）", "zh": "中文翻译8"},
    {"en": "例句9（搭配常用词组）", "zh": "中文翻译9"},
    {"en": "例句10（展示词义细微差异）", "zh": "中文翻译10"}
  ],
  "scenarios": [
    "常见使用场景1（中文，20字以内）",
    "常见使用场景2",
    "常见使用场景3"
  ],
  "memory_tip": "记忆技巧或词根词缀提示（中文，可选）",
  "cefr_level": "该词的CEFR等级，只填A1/A2/B1/B2/C1/C2之一"
}`

	messages := []map[string]any{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": userPrompt},
	}
	raw, err := callGroq(messages, 2048)
	if err != nil {
		return nil, err
	}

	var exp AIExplanation
	if err := json.Unmarshal([]byte(repairJSON(raw)), &exp); err != nil {
		// 小模型解析失败，用 70b 重试一次
		slog.Warn("小模型 JSON 解析失败，切换 70b 重试", "word", word.Word)
		raw, err = callGroqFallback(messages, 4096)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(repairJSON(raw)), &exp); err != nil {
			return nil, fmt.Errorf("解析 AI JSON 失败: %w\n原始内容: %s", err, raw)
		}
	}
	return &exp, nil
}

// repairJSON 处理 AI 常见的 JSON 格式问题
func repairJSON(s string) string {
	// 去掉 markdown 代码块
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		if idx := strings.Index(s, "\n"); idx != -1 {
			s = s[idx+1:]
		}
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}
	return s
}

// EnsureAI 确保单词有 AI 解释（缓存命中直接返回，否则调用 API）
func EnsureAI(db *sql.DB, apiKey string, w *models.Word) (*AIExplanation, error) {
	var exp *AIExplanation

	if w.AIReady() {
		var err error
		exp, err = parseAI(w)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		exp, err = fetchAIExplanation(apiKey, w)
		if err != nil {
			return nil, err
		}

		examplesJSON, _ := json.Marshal(exp.Examples)
		scenariosJSON, _ := json.Marshal(exp.Scenarios)
		meaningJSON, _ := json.Marshal(map[string]any{
			"primary_meaning":     exp.PrimaryMeaning,
			"part_of_speech":      exp.PartOfSpeech,
			"phonetic":            exp.Phonetic,
			"english_def":         exp.EnglishDef,
			"context_translation": exp.ContextTranslation,
			"definitions":         exp.Definitions,
		})

		if err := models.UpdateWordAI(db, w.ID,
			string(meaningJSON),
			string(examplesJSON),
			string(scenariosJSON),
			exp.MemoryTip,
		); err != nil {
			return nil, err
		}

		w.AIMeaning = string(meaningJSON)
		w.AIExamples = string(examplesJSON)
		w.AIScenarios = string(scenariosJSON)
		w.AIMemoryTip = exp.MemoryTip
		w.AIGeneratedAt = time.Now().Unix()
	}

	if strings.TrimSpace(w.Context) != "" && strings.TrimSpace(exp.ContextTranslation) == "" {
		if translation, err := translateContext(apiKey, w.Word, w.Context); err == nil && translation != "" {
			exp.ContextTranslation = translation
			if updatedMeaning, err := mergeContextTranslation(w.AIMeaning, translation); err == nil {
				if err := models.UpdateWordAI(db, w.ID, updatedMeaning, w.AIExamples, w.AIScenarios, w.AIMemoryTip); err == nil {
					w.AIMeaning = updatedMeaning
				}
			}
		}
	}

	// 评级：优先查本地词表，其次用 AI 返回的 CEFR
	if w.RatingCEFR == "" {
		var rating freq.Rating
		form := freq.Normalize(w.Word)
		if r, ok := freq.Lookup(form.Base); ok {
			rating = r
		} else if r, ok := freq.Lookup(w.Word); ok {
			rating = r
		} else {
			rating = freq.CEFRToRating(exp.CEFRLevel)
		}
		models.UpdateWordRating(db, w.ID, rating.CEFR, rating.Freq, rating.Rec)
		w.RatingCEFR = rating.CEFR
		w.RatingFreq = rating.Freq
		w.RatingRec = rating.Rec
	}

	return exp, nil
}

type JudgeResult struct {
	Correct  bool   `json:"correct"`
	TooVague bool   `json:"too_vague"`
	Feedback string `json:"feedback"`
}

type aiMeaningPayload struct {
	PrimaryMeaning     string       `json:"primary_meaning"`
	PartOfSpeech       string       `json:"part_of_speech"`
	Phonetic           string       `json:"phonetic"`
	EnglishDef         string       `json:"english_def"`
	ContextTranslation string       `json:"context_translation"`
	Definitions        []Definition `json:"definitions"`
}

func translateContext(_ string, word, context string) (string, error) {
	prompt := fmt.Sprintf(`请将下面的英文句子翻译成自然、准确的中文，只返回译文，不要解释。

单词：%s
句子：%s`, word, context)
	raw, err := callGroq([]map[string]any{{"role": "user", "content": prompt}}, 128)
	if err != nil {
		return "", err
	}
	return cleanTranslation(raw), nil
}

func cleanTranslation(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "`\"“”‘’")
	return strings.TrimSpace(s)
}

func mergeContextTranslation(rawMeaning, translation string) (string, error) {
	var meaning aiMeaningPayload
	if err := json.Unmarshal([]byte(rawMeaning), &meaning); err != nil {
		return "", err
	}
	meaning.ContextTranslation = translation
	updated, err := json.Marshal(meaning)
	if err != nil {
		return "", err
	}
	return string(updated), nil
}

// JudgeEnglishExplanation 调用 AI 判断用户的英文解释是否正确
func JudgeEnglishExplanation(_ string, word, explanation, englishDef string) (*JudgeResult, error) {
	prompt := fmt.Sprintf(`你是英语词汇评测助手。请判断学习者对单词的英文解释是否正确。

单词：%s
参考释义：%s
学习者的解释：%s

请返回严格 JSON（不加代码块标记）：
{
  "correct": true或false,
  "too_vague": true或false,
  "feedback": "简短的中文反馈（1-2句）"
}

判断规则：
- correct: 学习者抓住了核心含义即可，表达不必完美
- too_vague: 解释少于5个单词、无实质内容、或完全跑题时为true
- feedback: 鼓励性语气，指出正确点或欠缺之处`, word, englishDef, explanation)

	messages := []map[string]any{{"role": "user", "content": prompt}}
	raw, err := callGroq(messages, 256)
	if err != nil {
		return nil, err
	}

	var result JudgeResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("解析判断结果失败: %w", err)
	}
	return &result, nil
}

func parseAI(w *models.Word) (*AIExplanation, error) {
	var meaning aiMeaningPayload
	if err := json.Unmarshal([]byte(w.AIMeaning), &meaning); err != nil {
		return nil, err
	}

	var examples []Example
	json.Unmarshal([]byte(w.AIExamples), &examples)

	var scenarios []string
	json.Unmarshal([]byte(w.AIScenarios), &scenarios)

	return &AIExplanation{
		PrimaryMeaning:     meaning.PrimaryMeaning,
		PartOfSpeech:       meaning.PartOfSpeech,
		Phonetic:           meaning.Phonetic,
		EnglishDef:         meaning.EnglishDef,
		ContextTranslation: meaning.ContextTranslation,
		Definitions:        meaning.Definitions,
		Examples:           examples,
		Scenarios:          scenarios,
		MemoryTip:          w.AIMemoryTip,
	}, nil
}

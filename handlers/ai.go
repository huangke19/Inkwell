package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"inkwell/models"
)

type AIExplanation struct {
	PrimaryMeaning string       `json:"primary_meaning"`
	PartOfSpeech   string       `json:"part_of_speech"`
	Phonetic       string       `json:"phonetic"`
	EnglishDef     string       `json:"english_def"`
	Definitions    []Definition `json:"definitions"`
	Examples       []Example    `json:"examples"`
	Scenarios      []string     `json:"scenarios"`
	MemoryTip      string       `json:"memory_tip"`
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

func fetchAIExplanation(apiKey string, word *models.Word) (*AIExplanation, error) {
	userPrompt := fmt.Sprintf("单词：%s\n", word.Word)
	if word.Context != "" {
		userPrompt += fmt.Sprintf("用户遇到该词的语境：%s\n", word.Context)
	}
	userPrompt += `
请按以下 JSON 结构返回（不加代码块标记，直接输出 JSON）：
{
  "primary_meaning": "最核心的中文释义（10字以内）",
  "part_of_speech": "adj/n/v/adv/prep（英文缩写）",
  "phonetic": "/IPA音标，如 /ʌbˈɪkwɪtəs/，使用美式发音/",
  "english_def": "详尽的全英文释义，像英英词典一样解释这个词的含义、用法、语感、与近义词的区别，以及在不同语境下的细微差异。至少150个英文单词，越详细越好。",
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
  "memory_tip": "记忆技巧或词根词缀提示（中文，可选）"
}`

	reqBody := map[string]any{
		"model": "llama-3.3-70b-versatile",
		"messages": []map[string]any{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0,
		"max_tokens":  1024,
	}

	bodyBytes, _ := json.Marshal(reqBody)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, groqURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Groq API 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Groq API 返回错误 %d: %s", resp.StatusCode, string(respBytes))
	}

	var groqResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBytes, &groqResp); err != nil {
		return nil, fmt.Errorf("解析 Groq 响应失败: %w", err)
	}
	if len(groqResp.Choices) == 0 {
		return nil, fmt.Errorf("Groq 返回空内容")
	}

	raw := groqResp.Choices[0].Message.Content
	var exp AIExplanation
	if err := json.Unmarshal([]byte(raw), &exp); err != nil {
		return nil, fmt.Errorf("解析 AI JSON 失败: %w\n原始内容: %s", err, raw)
	}
	return &exp, nil
}

// EnsureAI 确保单词有 AI 解释（缓存命中直接返回，否则调用 API）
func EnsureAI(db *sql.DB, apiKey string, w *models.Word) (*AIExplanation, error) {
	if w.AIReady() {
		return parseAI(w)
	}

	exp, err := fetchAIExplanation(apiKey, w)
	if err != nil {
		return nil, err
	}

	examplesJSON, _ := json.Marshal(exp.Examples)
	scenariosJSON, _ := json.Marshal(exp.Scenarios)
	meaningJSON, _ := json.Marshal(map[string]any{
		"primary_meaning": exp.PrimaryMeaning,
		"part_of_speech":  exp.PartOfSpeech,
		"phonetic":        exp.Phonetic,
		"english_def":     exp.EnglishDef,
		"definitions":     exp.Definitions,
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

	return exp, nil
}

func parseAI(w *models.Word) (*AIExplanation, error) {
	var meaning struct {
		PrimaryMeaning string       `json:"primary_meaning"`
		PartOfSpeech   string       `json:"part_of_speech"`
		Phonetic       string       `json:"phonetic"`
		EnglishDef     string       `json:"english_def"`
		Definitions    []Definition `json:"definitions"`
	}
	if err := json.Unmarshal([]byte(w.AIMeaning), &meaning); err != nil {
		return nil, err
	}

	var examples []Example
	json.Unmarshal([]byte(w.AIExamples), &examples)

	var scenarios []string
	json.Unmarshal([]byte(w.AIScenarios), &scenarios)

	return &AIExplanation{
		PrimaryMeaning: meaning.PrimaryMeaning,
		PartOfSpeech:   meaning.PartOfSpeech,
		Phonetic:       meaning.Phonetic,
		EnglishDef:     meaning.EnglishDef,
		Definitions:    meaning.Definitions,
		Examples:       examples,
		Scenarios:      scenarios,
		MemoryTip:      w.AIMemoryTip,
	}, nil
}

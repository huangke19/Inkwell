package handlers

import (
	"database/sql"
	"log/slog"
	"net/http"
	"strings"

	"inkwell/srs"
)

type ReviewHandler struct {
	db     *sql.DB
	r      *Renderer
	apiKey string
}

func NewReviewHandler(db *sql.DB, r *Renderer, apiKey string) *ReviewHandler {
	return &ReviewHandler{db: db, r: r, apiKey: apiKey}
}

// Start 首次加载复习页面
func (h *ReviewHandler) Start(w http.ResponseWriter, r *http.Request) {
	word, err := nextDueWord(h.db)
	if err != nil {
		http.Error(w, "查询失败", http.StatusInternalServerError)
		return
	}
	due, _ := countDueWords(h.db)
	h.r.Page(w, "review", map[string]any{"Word": word, "Due": due})
}

// Next 加载下一个待复习单词（HTMX 片段）
func (h *ReviewHandler) Next(w http.ResponseWriter, r *http.Request) {
	word, err := nextDueWord(h.db)
	if err != nil {
		http.Error(w, "查询失败", http.StatusInternalServerError)
		return
	}
	due, _ := countDueWords(h.db)
	h.r.Fragment(w, "quiz_card", map[string]any{
		"Word": word,
		"Due":  due,
	}, "templates/review.html")
}

// Forgot 用户点「不记得」→ 加载 AI 解释供阅读
func (h *ReviewHandler) Forgot(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	word, err := getWordByID(h.db, id)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "查询失败", http.StatusInternalServerError)
		return
	}

	exp, err := EnsureAI(h.db, h.apiKey, word)
	if err != nil {
		slog.Error("AI 获取失败", "err", err)
		http.Error(w, "获取解释失败", http.StatusInternalServerError)
		return
	}

	due, _ := countDueWords(h.db)
	h.r.Fragment(w, "ai_reading", map[string]any{"Word": word, "AI": exp, "Due": due})
}

// ConfirmForgot 用户读完解释后确认 → 标记答错，更新 SRS
func (h *ReviewHandler) ConfirmForgot(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	word, err := getWordByID(h.db, id)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "查询失败", http.StatusInternalServerError)
		return
	}

	exp, err := EnsureAI(h.db, h.apiKey, word)
	if err != nil {
		slog.Error("AI 获取失败", "err", err)
		http.Error(w, "获取解释失败", http.StatusInternalServerError)
		return
	}

	intervalBefore := word.IntervalDays
	newInterval, newRep, nextAt := srs.Next(word.IntervalDays, word.Repetitions, srs.Incorrect)
	updateWordSRS(h.db, word.ID, newInterval, nextAt, newRep)
	createReviewLog(h.db, word.ID, "incorrect", "", intervalBefore, newInterval)

	due, _ := countDueWords(h.db)
	h.r.Fragment(w, "review_result", map[string]any{
		"Word":        word,
		"AI":          exp,
		"Correct":     false,
		"Feedback":    "",
		"NewInterval": newInterval,
		"Due":         due,
	})
}

// Remember 用户点「记得」→ 展示英文解释输入框
func (h *ReviewHandler) Remember(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	word, err := getWordByID(h.db, id)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "查询失败", http.StatusInternalServerError)
		return
	}

	due, _ := countDueWords(h.db)
	h.r.Fragment(w, "english_input", map[string]any{"Word": word, "Due": due})
}

// Explain 用户提交英文解释 → AI 判断 → 返回结果或提示补充
func (h *ReviewHandler) Explain(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	word, err := getWordByID(h.db, id)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "查询失败", http.StatusInternalServerError)
		return
	}

	explanation := strings.TrimSpace(r.FormValue("explanation"))
	due, _ := countDueWords(h.db)

	// 客户端长度预检（少于 10 个字符直接提示，不消耗 API）
	if len([]rune(explanation)) < 10 {
		h.r.Fragment(w, "english_input", map[string]any{
			"Word":       word,
			"Due":        due,
			"Error":      "请用至少一两句话解释这个词的意思和用法。",
			"PrevAnswer": explanation,
		})
		return
	}

	exp, err := EnsureAI(h.db, h.apiKey, word)
	if err != nil {
		slog.Error("AI 获取失败", "err", err)
		http.Error(w, "获取解释失败", http.StatusInternalServerError)
		return
	}

	judge, err := JudgeEnglishExplanation(h.apiKey, word.Word, explanation, exp.EnglishDef)
	if err != nil {
		slog.Error("AI 判断失败", "err", err)
		http.Error(w, "判断失败，请重试", http.StatusInternalServerError)
		return
	}

	// AI 认为解释太模糊，提示补充
	if judge.TooVague {
		h.r.Fragment(w, "english_input", map[string]any{
			"Word":       word,
			"Due":        due,
			"Error":      "解释太简短，请更详细地描述这个词的含义和用法。",
			"PrevAnswer": explanation,
		})
		return
	}

	srsResult := srs.Incorrect
	resultStr := "incorrect"
	if judge.Correct {
		srsResult = srs.Correct
		resultStr = "correct"
	}

	intervalBefore := word.IntervalDays
	newInterval, newRep, nextAt := srs.Next(word.IntervalDays, word.Repetitions, srsResult)
	updateWordSRS(h.db, word.ID, newInterval, nextAt, newRep)
	createReviewLog(h.db, word.ID, resultStr, explanation, intervalBefore, newInterval)

	h.r.Fragment(w, "review_result", map[string]any{
		"Word":        word,
		"AI":          exp,
		"Correct":     judge.Correct,
		"Feedback":    judge.Feedback,
		"NewInterval": newInterval,
		"Due":         due,
	})
}

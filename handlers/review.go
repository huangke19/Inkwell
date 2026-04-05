package handlers

import (
	"database/sql"
	"log/slog"
	"net/http"
	"strings"
	"unicode"

	"ewords/srs"
)

type ReviewHandler struct {
	db     *sql.DB
	r      *Renderer
	apiKey string
}

func NewReviewHandler(db *sql.DB, r *Renderer, apiKey string) *ReviewHandler {
	return &ReviewHandler{db: db, r: r, apiKey: apiKey}
}

func (h *ReviewHandler) Start(w http.ResponseWriter, r *http.Request) {
	word, err := nextDueWord(h.db)
	if err != nil {
		http.Error(w, "查询失败", http.StatusInternalServerError)
		return
	}
	due, _ := countDueWords(h.db)
	h.r.Page(w, "review", map[string]any{"Word": word, "Due": due})
}

func (h *ReviewHandler) Submit(w http.ResponseWriter, r *http.Request) {
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

	userAnswer := strings.TrimSpace(r.FormValue("answer"))

	exp, err := EnsureAI(h.db, h.apiKey, word)
	if err != nil {
		slog.Error("AI 获取失败", "err", err)
		http.Error(w, "获取解释失败", http.StatusInternalServerError)
		return
	}

	correct := checkAnswer(userAnswer, exp)
	resultStr := "incorrect"
	srsResult := srs.Incorrect
	if correct {
		resultStr = "correct"
		srsResult = srs.Correct
	}

	intervalBefore := word.IntervalDays
	newInterval, newRep, nextAt := srs.Next(word.IntervalDays, word.Repetitions, srsResult)

	updateWordSRS(h.db, word.ID, newInterval, nextAt, newRep)
	createReviewLog(h.db, word.ID, resultStr, userAnswer, intervalBefore, newInterval)

	due, _ := countDueWords(h.db)

	h.r.Fragment(w, "review_result", map[string]any{
		"Word":        word,
		"AI":          exp,
		"Correct":     correct,
		"UserAnswer":  userAnswer,
		"NewInterval": newInterval,
		"Due":         due,
	})
}

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

func checkAnswer(userAnswer string, exp *AIExplanation) bool {
	if userAnswer == "" {
		return false
	}
	ua := normalize(userAnswer)

	candidates := []string{exp.PrimaryMeaning}
	for _, d := range exp.Definitions {
		candidates = append(candidates, d.ZH)
	}

	for _, c := range candidates {
		cn := normalize(c)
		if cn == "" {
			continue
		}
		if strings.Contains(ua, cn) || strings.Contains(cn, ua) {
			return true
		}
	}
	return false
}

func normalize(s string) string {
	var b strings.Builder
	for _, r := range s {
		if !unicode.IsSpace(r) && !unicode.IsPunct(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	result := b.String()
	if len([]rune(result)) < 2 {
		return ""
	}
	return result
}

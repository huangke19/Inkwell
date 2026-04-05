package handlers

import (
	"database/sql"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

type WordHandler struct {
	db     *sql.DB
	r      *Renderer
	apiKey string
}

func NewWordHandler(db *sql.DB, r *Renderer, apiKey string) *WordHandler {
	return &WordHandler{db: db, r: r, apiKey: apiKey}
}

func (h *WordHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	words, err := listWords(h.db, q)
	if err != nil {
		http.Error(w, "查询失败", http.StatusInternalServerError)
		return
	}
	total, mastered, due, err := wordStats(h.db)
	if err != nil {
		http.Error(w, "统计失败", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Words":    words,
		"Query":    q,
		"Total":    total,
		"Mastered": mastered,
		"Due":      due,
	}

	if isHTMX(r) {
		h.r.Fragment(w, "word_list_rows", data, "templates/index.html")
		return
	}
	h.r.Page(w, "index", data)
}

func (h *WordHandler) AddForm(w http.ResponseWriter, r *http.Request) {
	h.r.Page(w, "add", nil)
}

func (h *WordHandler) Create(w http.ResponseWriter, r *http.Request) {
	word := strings.TrimSpace(r.FormValue("word"))
	context := strings.TrimSpace(r.FormValue("context"))

	if word == "" {
		h.r.Fragment(w, "form_error", map[string]any{"Error": "单词不能为空"})
		return
	}

	existing, _ := getWordByText(h.db, word)
	if existing != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		h.r.Fragment(w, "form_error", map[string]any{"Error": "单词已存在：" + word})
		return
	}

	created, err := createWord(h.db, word, context)
	if err != nil {
		slog.Error("创建单词失败", "err", err)
		w.WriteHeader(http.StatusUnprocessableEntity)
		h.r.Fragment(w, "form_error", map[string]any{"Error": "保存失败，请重试"})
		return
	}

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/words/"+strconv.FormatInt(created.ID, 10))
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/words/"+strconv.FormatInt(created.ID, 10), http.StatusSeeOther)
}

func (h *WordHandler) Detail(w http.ResponseWriter, r *http.Request) {
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

	h.r.Page(w, "word_detail", map[string]any{"Word": word})
}

func (h *WordHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := deleteWord(h.db, id); err != nil {
		http.Error(w, "删除失败", http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *WordHandler) GetAI(w http.ResponseWriter, r *http.Request) {
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

	// ?refresh=1 强制重新生成
	if r.URL.Query().Get("refresh") == "1" {
		word.AIGeneratedAt = 0
		word.AIMeaning = ""
	}

	exp, err := EnsureAI(h.db, h.apiKey, word)
	if err != nil {
		slog.Error("AI 生成失败", "word", word.Word, "err", err)
		http.Error(w, "AI 解释生成失败，请稍后重试", http.StatusInternalServerError)
		return
	}

	h.r.Fragment(w, "ai_explanation", map[string]any{"Word": word, "AI": exp})
}

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func idFromPath(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

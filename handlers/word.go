package handlers

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"inkwell/models"
)

const wordsPerPage = 15

type wordListPageData struct {
	Words      []*models.Word
	Query      string
	Sort       string
	BaseURL    string
	TargetID   string
	Total      int
	Mastered   int
	Due        int
	Count      int
	EmptyMsg   string
	Page       int
	TotalPages int
	HasPrev    bool
	HasNext    bool
	PrevURL    string
	NextURL    string
}

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
	sort := sortFromQuery(r)
	page := pageFromQuery(r)
	filteredTotal, err := countWordsFiltered(h.db, q)
	if err != nil {
		http.Error(w, "查询失败", http.StatusInternalServerError)
		return
	}
	totalPages := (filteredTotal + wordsPerPage - 1) / wordsPerPage
	if totalPages == 0 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	words, err := listWords(h.db, q, sort, wordsPerPage, (page-1)*wordsPerPage)
	if err != nil {
		http.Error(w, "查询失败", http.StatusInternalServerError)
		return
	}
	total, mastered, due, err := wordStats(h.db)
	if err != nil {
		http.Error(w, "统计失败", http.StatusInternalServerError)
		return
	}

	data := wordListPageData{
		Words:      words,
		Query:      q,
		Sort:       sort,
		BaseURL:    "/",
		TargetID:   "word-list",
		Total:      total,
		Mastered:   mastered,
		Due:        due,
		Count:      filteredTotal,
		Page:       page,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
		PrevURL:    listURL("/", q, sort, page-1),
		NextURL:    listURL("/", q, sort, page+1),
	}

	if isHTMX(r) {
		h.r.Fragment(w, "word_list", data, "templates/index.html")
		return
	}
	h.r.Page(w, "index", data)
}

func pageFromQuery(r *http.Request) int {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		return 1
	}
	return page
}

func sortFromQuery(r *http.Request) string {
	switch strings.ToLower(strings.TrimSpace(r.URL.Query().Get("sort"))) {
	case "created":
		return "created"
	default:
		return "rating"
	}
}

func listURL(path, q, sort string, page int) string {
	values := url.Values{}
	if q != "" {
		values.Set("q", q)
	}
	if sort != "" {
		values.Set("sort", sort)
	}
	if page > 1 {
		values.Set("page", strconv.Itoa(page))
	}
	if encoded := values.Encode(); encoded != "" {
		return path + "?" + encoded
	}
	return path
}

func (h *WordHandler) AddForm(w http.ResponseWriter, r *http.Request) {
	h.r.Page(w, "add", nil)
}

func (h *WordHandler) Create(w http.ResponseWriter, r *http.Request) {
	var word, context, sourceURL, sourceTitle string

	if r.Header.Get("Content-Type") == "application/json" {
		// 扩展 / API 调用
		var body struct {
			Word        string `json:"word"`
			Context     string `json:"context"`
			SourceURL   string `json:"sourceUrl"`
			SourceTitle string `json:"sourceTitle"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "请求格式错误"})
			return
		}
		word = strings.TrimSpace(body.Word)
		context = strings.TrimSpace(body.Context)
		sourceURL = strings.TrimSpace(body.SourceURL)
		sourceTitle = strings.TrimSpace(body.SourceTitle)
	} else {
		word = strings.TrimSpace(r.FormValue("word"))
		context = strings.TrimSpace(r.FormValue("context"))
		sourceURL = strings.TrimSpace(r.FormValue("source_url"))
		sourceTitle = strings.TrimSpace(r.FormValue("source_title"))
	}

	isJSON := r.Header.Get("Content-Type") == "application/json"

	if word == "" {
		if isJSON {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "单词不能为空"})
			return
		}
		h.r.Fragment(w, "form_error", map[string]any{"Error": "单词不能为空"})
		return
	}

	existing, _ := getWordByText(h.db, word)
	if existing != nil {
		if isJSON {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]any{"error": "already_exists", "word": word, "id": existing.ID})
			return
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		h.r.Fragment(w, "form_error", map[string]any{"Error": "单词已存在：" + word})
		return
	}

	created, err := createWord(h.db, word, context, sourceURL, sourceTitle)
	if err != nil {
		slog.Error("创建单词失败", "err", err)
		if isJSON {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "保存失败"})
			return
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		h.r.Fragment(w, "form_error", map[string]any{"Error": "保存失败，请重试"})
		return
	}

	if isJSON {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"id": created.ID, "word": created.Word})
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

	h.r.Page(w, "word_detail", map[string]any{"Word": word, "ShowContextTranslation": true})
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

func (h *WordHandler) Unmastered(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	sort := sortFromQuery(r)
	page := pageFromQuery(r)
	count, err := countUnmasteredWordsFiltered(h.db, q)
	if err != nil {
		http.Error(w, "查询失败", http.StatusInternalServerError)
		return
	}
	totalPages := (count + wordsPerPage - 1) / wordsPerPage
	if totalPages == 0 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	words, err := listUnmastered(h.db, q, sort, wordsPerPage, (page-1)*wordsPerPage)
	if err != nil {
		http.Error(w, "查询失败", http.StatusInternalServerError)
		return
	}
	due, _ := countDueWords(h.db)
	data := map[string]any{
		"Words":      words,
		"Query":      q,
		"Sort":       sort,
		"BaseURL":    "/unmastered",
		"TargetID":   "unmastered-list",
		"Count":      count,
		"Due":        due,
		"EmptyMsg":   "太棒了，所有单词都已掌握！",
		"Page":       page,
		"TotalPages": totalPages,
		"HasPrev":    page > 1,
		"HasNext":    page < totalPages,
		"PrevURL":    listURL("/unmastered", q, sort, page-1),
		"NextURL":    listURL("/unmastered", q, sort, page+1),
	}
	if isHTMX(r) {
		h.r.Fragment(w, "unmastered_list", data, "templates/unmastered.html")
		return
	}
	h.r.Page(w, "unmastered", data)
}

func (h *WordHandler) Mastered(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	sort := sortFromQuery(r)
	page := pageFromQuery(r)
	count, err := countMasteredWordsFiltered(h.db, q)
	if err != nil {
		http.Error(w, "查询失败", http.StatusInternalServerError)
		return
	}
	totalPages := (count + wordsPerPage - 1) / wordsPerPage
	if totalPages == 0 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	words, err := listMastered(h.db, q, sort, wordsPerPage, (page-1)*wordsPerPage)
	if err != nil {
		http.Error(w, "查询失败", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Words":      words,
		"Query":      q,
		"Sort":       sort,
		"BaseURL":    "/mastered",
		"TargetID":   "mastered-list",
		"Count":      count,
		"Page":       page,
		"TotalPages": totalPages,
		"HasPrev":    page > 1,
		"HasNext":    page < totalPages,
		"PrevURL":    listURL("/mastered", q, sort, page-1),
		"NextURL":    listURL("/mastered", q, sort, page+1),
	}

	if isHTMX(r) {
		h.r.Fragment(w, "mastered_list", data, "templates/mastered.html")
		return
	}
	h.r.Page(w, "mastered", data)
}

func (h *WordHandler) Master(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := markMastered(h.db, id); err != nil {
		http.Error(w, "操作失败", http.StatusInternalServerError)
		return
	}
	// inline=1 时（单词列表行）只删除行，不跳转
	if isHTMX(r) && r.URL.Query().Get("inline") == "1" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/mastered")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/mastered", http.StatusSeeOther)
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
		h.r.Fragment(w, "ai_error", map[string]any{
			"WordID": word.ID,
			"Error":  err.Error(),
		})
		return
	}

	h.r.Fragment(w, "ai_explanation", map[string]any{"Word": word, "AI": exp, "ShowContextTranslation": true})
}

// CORS 处理扩展发来的预检请求
func CORS(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	w.WriteHeader(http.StatusNoContent)
}

func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func idFromPath(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

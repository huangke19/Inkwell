package handlers

import (
	"html/template"
	"net/http"
	"strings"
	"time"

	"inkwell/models"
)

// Renderer 按需组合模板，避免多页面 {{define "content"}} 命名冲突
type Renderer struct {
	funcMap template.FuncMap
}

func NewRenderer() *Renderer {
	return &Renderer{
		funcMap: template.FuncMap{
			"inc": func(i int) int { return i + 1 },
			"firstSentence": func(s string) string {
				if s == "" {
					return ""
				}
				if idx := strings.Index(s, ". "); idx != -1 {
					return s[:idx+1]
				}
				if len(s) > 120 {
					return s[:120] + "…"
				}
				return s
			},
			"primaryMeaning": func(aiMeaning string) string {
				if aiMeaning == "" {
					return "—"
				}
				const key = `"primary_meaning":"`
				idx := strings.Index(aiMeaning, key)
				if idx == -1 {
					return "—"
				}
				start := idx + len(key)
				end := strings.Index(aiMeaning[start:], `"`)
				if end == -1 {
					return "—"
				}
				return aiMeaning[start : start+end]
			},
			"reviewTime": func(ts int64) string {
				if ts == 0 {
					return "立即"
				}
				t := time.Unix(ts, 0)
				diff := t.Sub(time.Now())
				if diff <= 0 {
					return "待复习"
				}
				days := int(diff.Hours() / 24)
				switch days {
				case 0:
					return "今天"
				case 1:
					return "明天"
				default:
					return models.FormatDays(days)
				}
			},
		},
	}
}

// Page 渲染完整页面（layout + 指定页面文件）
func (r *Renderer) Page(w http.ResponseWriter, page string, data any, extraFiles ...string) {
	files := []string{
		"templates/layout.html",
		"templates/" + page + ".html",
		"templates/partials/ai_explanation.html",
		"templates/partials/review_result.html",
		"templates/partials/ai_reading.html",
		"templates/partials/english_input.html",
		"templates/partials/form_error.html",
	}
	files = append(files, extraFiles...)

	t, err := template.New("layout.html").Funcs(r.funcMap).ParseFiles(files...)
	if err != nil {
		http.Error(w, "模板加载失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := t.Execute(w, data); err != nil {
		http.Error(w, "模板渲染失败: "+err.Error(), http.StatusInternalServerError)
	}
}

// Fragment 渲染局部片段（HTMX 使用，不套 layout）
func (r *Renderer) Fragment(w http.ResponseWriter, name string, data any, extraFiles ...string) {
	files := []string{
		"templates/partials/ai_explanation.html",
		"templates/partials/review_result.html",
		"templates/partials/ai_reading.html",
		"templates/partials/english_input.html",
		"templates/partials/form_error.html",
	}
	files = append(files, extraFiles...)

	t, err := template.New("").Funcs(r.funcMap).ParseFiles(files...)
	if err != nil {
		http.Error(w, "模板加载失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := t.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "模板渲染失败: "+err.Error(), http.StatusInternalServerError)
	}
}

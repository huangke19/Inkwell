package freq

import (
	_ "embed"
	"strings"
)

//go:embed wordlist.csv
var wordlistCSV string

type Rating struct {
	CEFR string // A1, A2, B1, B2, C1, C2
	Freq string // 高频, 中频, 低频, 罕见
	Rec  string // 强烈推荐, 建议掌握, 选择性记, 可以跳过
}

var wordMap map[string]Rating

func init() {
	wordMap = make(map[string]Rating, 2000)
	for _, line := range strings.Split(wordlistCSV, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "word,") {
			continue
		}
		parts := strings.SplitN(line, ",", 2)
		if len(parts) != 2 {
			continue
		}
		word := strings.ToLower(strings.TrimSpace(parts[0]))
		cefr := strings.TrimSpace(parts[1])
		wordMap[word] = CEFRToRating(cefr)
	}
}

// Lookup 查找单词的评级，found=false 表示不在词表中
func Lookup(word string) (Rating, bool) {
	r, ok := wordMap[strings.ToLower(word)]
	return r, ok
}

// CEFRToRating 根据 CEFR 等级生成评级信息
func CEFRToRating(cefr string) Rating {
	switch cefr {
	case "A1", "A2":
		return Rating{CEFR: cefr, Freq: "高频", Rec: "强烈推荐"}
	case "B1", "B2":
		return Rating{CEFR: cefr, Freq: "中频", Rec: "建议掌握"}
	case "C1":
		return Rating{CEFR: cefr, Freq: "低频", Rec: "选择性记"}
	default: // C2 或未知
		return Rating{CEFR: cefr, Freq: "罕见", Rec: "可以跳过"}
	}
}

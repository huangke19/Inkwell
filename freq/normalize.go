package freq

import "strings"

type FormInfo struct {
	Surface string
	Base    string
	Kind    string
	Changed bool
}

type baseCandidate struct {
	base string
	kind string
}

var irregularForms = map[string]baseCandidate{
	"am":         {base: "be", kind: "be 动词"},
	"is":         {base: "be", kind: "be 动词"},
	"are":        {base: "be", kind: "be 动词"},
	"was":        {base: "be", kind: "过去式"},
	"were":       {base: "be", kind: "过去式"},
	"been":       {base: "be", kind: "过去分词"},
	"being":      {base: "be", kind: "进行时/动名词"},
	"has":        {base: "have", kind: "第三人称单数"},
	"had":        {base: "have", kind: "过去式"},
	"having":     {base: "have", kind: "进行时/动名词"},
	"does":       {base: "do", kind: "第三人称单数"},
	"did":        {base: "do", kind: "过去式"},
	"done":       {base: "do", kind: "过去分词"},
	"doing":      {base: "do", kind: "进行时/动名词"},
	"went":       {base: "go", kind: "过去式"},
	"gone":       {base: "go", kind: "过去分词"},
	"made":       {base: "make", kind: "过去式"},
	"took":       {base: "take", kind: "过去式"},
	"taken":      {base: "take", kind: "过去分词"},
	"came":       {base: "come", kind: "过去式"},
	"become":     {base: "become", kind: "原形"},
	"began":      {base: "begin", kind: "过去式"},
	"begun":      {base: "begin", kind: "过去分词"},
	"bought":     {base: "buy", kind: "过去式"},
	"brought":    {base: "bring", kind: "过去式"},
	"broke":      {base: "break", kind: "过去式"},
	"broken":     {base: "break", kind: "过去分词"},
	"chose":      {base: "choose", kind: "过去式"},
	"chosen":     {base: "choose", kind: "过去分词"},
	"drove":      {base: "drive", kind: "过去式"},
	"driven":     {base: "drive", kind: "过去分词"},
	"ate":        {base: "eat", kind: "过去式"},
	"eaten":      {base: "eat", kind: "过去分词"},
	"fell":       {base: "fall", kind: "过去式"},
	"fallen":     {base: "fall", kind: "过去分词"},
	"felt":       {base: "feel", kind: "过去式"},
	"found":      {base: "find", kind: "过去式"},
	"gave":       {base: "give", kind: "过去式"},
	"given":      {base: "give", kind: "过去分词"},
	"knew":       {base: "know", kind: "过去式"},
	"known":      {base: "know", kind: "过去分词"},
	"left":       {base: "leave", kind: "过去式"},
	"met":        {base: "meet", kind: "过去式"},
	"paid":       {base: "pay", kind: "过去式"},
	"ran":        {base: "run", kind: "过去式"},
	"read":       {base: "read", kind: "原形/过去式"},
	"said":       {base: "say", kind: "过去式"},
	"saw":        {base: "see", kind: "过去式"},
	"seen":       {base: "see", kind: "过去分词"},
	"sang":       {base: "sing", kind: "过去式"},
	"sung":       {base: "sing", kind: "过去分词"},
	"sat":        {base: "sit", kind: "过去式"},
	"spoke":      {base: "speak", kind: "过去式"},
	"spoken":     {base: "speak", kind: "过去分词"},
	"stood":      {base: "stand", kind: "过去式"},
	"taught":     {base: "teach", kind: "过去式"},
	"told":       {base: "tell", kind: "过去式"},
	"thought":    {base: "think", kind: "过去式"},
	"threw":      {base: "throw", kind: "过去式"},
	"thrown":     {base: "throw", kind: "过去分词"},
	"wrote":      {base: "write", kind: "过去式"},
	"written":    {base: "write", kind: "过去分词"},
	"won":        {base: "win", kind: "过去式"},
	"held":       {base: "hold", kind: "过去式"},
	"kept":       {base: "keep", kind: "过去式"},
	"lost":       {base: "lose", kind: "过去式"},
	"heard":      {base: "hear", kind: "过去式"},
	"sold":       {base: "sell", kind: "过去式"},
	"sent":       {base: "send", kind: "过去式"},
	"spent":      {base: "spend", kind: "过去式"},
	"understood": {base: "understand", kind: "过去式"},
	"wasn't":     {base: "be", kind: "否定过去式"},
	"weren't":    {base: "be", kind: "否定过去式"},
}

func Normalize(word string) FormInfo {
	surface := strings.TrimSpace(word)
	if surface == "" {
		return FormInfo{}
	}
	cleaned := strings.Trim(surface, " 	\n\r.,;:!?()[]{}\"'“”‘’")
	if cleaned == "" {
		cleaned = surface
	}
	lower := strings.ToLower(cleaned)
	if lower == "" {
		return FormInfo{Surface: surface, Base: lower}
	}
	if _, ok := wordMap[lower]; ok {
		return FormInfo{Surface: surface, Base: lower, Changed: false}
	}
	if candidate, kind, ok := inflect(lower); ok {
		return FormInfo{Surface: surface, Base: candidate, Kind: kind, Changed: candidate != lower}
	}
	return FormInfo{Surface: surface, Base: lower, Changed: false}
}

func inflect(word string) (string, string, bool) {
	if candidate, ok := irregularForms[word]; ok {
		return candidate.base, candidate.kind, true
	}

	if strings.HasSuffix(word, "ying") && len(word) > 4 {
		candidate := strings.TrimSuffix(word, "ying") + "ie"
		return candidate, "进行时/动名词", true
	}

	if strings.HasSuffix(word, "ing") && len(word) > 4 {
		stem := strings.TrimSuffix(word, "ing")
		if candidate := pickExisting(stem, stem[:len(stem)-1]); candidate != "" {
			return candidate, "进行时/动名词", true
		}
		if candidate := pickExisting(stem + "e"); candidate != "" {
			return candidate, "进行时/动名词", true
		}
		return stem, "进行时/动名词", true
	}

	if strings.HasSuffix(word, "ied") && len(word) > 4 {
		candidate := strings.TrimSuffix(word, "ied") + "y"
		return candidate, "过去式", true
	}

	if strings.HasSuffix(word, "ed") && len(word) > 3 {
		stem := strings.TrimSuffix(word, "ed")
		if candidate := pickExisting(stem, stem[:len(stem)-1]); candidate != "" {
			return candidate, "过去式/过去分词", true
		}
		if candidate := pickExisting(stem + "e"); candidate != "" {
			return candidate, "过去式/过去分词", true
		}
		return stem, "过去式/过去分词", true
	}

	if strings.HasSuffix(word, "ies") && len(word) > 3 {
		candidate := strings.TrimSuffix(word, "ies") + "y"
		return candidate, "复数/三单", true
	}

	if strings.HasSuffix(word, "ves") && len(word) > 3 {
		stem := strings.TrimSuffix(word, "ves")
		if candidate := pickExisting(stem+"f", stem+"fe"); candidate != "" {
			return candidate, "复数", true
		}
		return stem + "f", "复数", true
	}

	if strings.HasSuffix(word, "es") && len(word) > 2 {
		stem := strings.TrimSuffix(word, "es")
		if candidate := pickExisting(stem, stem+"e"); candidate != "" {
			return candidate, "复数/三单", true
		}
		return stem, "复数/三单", true
	}

	if strings.HasSuffix(word, "s") && len(word) > 1 {
		stem := strings.TrimSuffix(word, "s")
		if candidate := pickExisting(stem); candidate != "" {
			return candidate, "复数/三单", true
		}
		return stem, "复数/三单", true
	}

	if strings.HasSuffix(word, "est") && len(word) > 3 {
		stem := strings.TrimSuffix(word, "est")
		if candidate := pickExisting(stem, stem+"e"); candidate != "" {
			return candidate, "最高级", true
		}
		return stem, "最高级", true
	}

	if strings.HasSuffix(word, "er") && len(word) > 2 {
		stem := strings.TrimSuffix(word, "er")
		if candidate := pickExisting(stem, stem+"e"); candidate != "" {
			return candidate, "比较级", true
		}
		return stem, "比较级", true
	}

	return "", "", false
}

func pickExisting(candidates ...string) string {
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, ok := wordMap[candidate]; ok {
			return candidate
		}
	}
	return ""
}

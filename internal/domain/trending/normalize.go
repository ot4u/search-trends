package trending

import (
	"strings"
	"unicode"
)

var intentWords = map[string]struct{}{
	"купить":   {},
	"заказать": {},
	"новый":    {},
	"дешево":   {},
}

func NormalizeText(raw string) string {
	if raw == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(raw))

	spacePending := false

	for _, r := range strings.ToLower(strings.TrimSpace(raw)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			b.WriteRune(r)
			spacePending = false
		case unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r):
			if !spacePending && b.Len() > 0 {
				b.WriteByte(' ')
				spacePending = true
			}
		default:
			if !spacePending && b.Len() > 0 {
				b.WriteByte(' ')
				spacePending = true
			}
		}
	}

	return strings.TrimSpace(b.String())
}
func NormalizeQuery(raw string) string {
	cleaned := NormalizeText(raw)
	if cleaned == "" {
		return ""
	}

	tokens := strings.Fields(cleaned)
	filtered := tokens[:0]

	for _, token := range tokens {
		if _, skip := intentWords[token]; skip {
			continue
		}
		filtered = append(filtered, token)
	}

	return strings.Join(filtered, " ")
}
func SplitTokens(normalized string) []string {
	if normalized == "" {
		return nil
	}

	return strings.Fields(normalized)
}

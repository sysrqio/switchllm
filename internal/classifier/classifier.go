package classifier

import (
	"strings"
	"unicode"
)

// Score returns a complexity score in [0,1] for routing decisions.
type Scorer struct {
	Keywords  []string
	Threshold float64
}

func New(keywords []string, threshold float64) *Scorer {
	kw := make([]string, len(keywords))
	for i, k := range keywords {
		kw[i] = strings.ToLower(strings.TrimSpace(k))
	}
	return &Scorer{Keywords: kw, Threshold: threshold}
}

// ScoreMessage combines keyword hits and message length heuristics.
func (s *Scorer) ScoreMessage(text string) float64 {
	if text == "" {
		return 0
	}
	lower := strings.ToLower(text)
	var score float64

	// Length: longer prompts tend to need stronger models
	runes := len([]rune(text))
	switch {
	case runes > 4000:
		score += 0.45
	case runes > 1500:
		score += 0.30
	case runes > 600:
		score += 0.18
	case runes > 200:
		score += 0.08
	}

	// Code-like content
	if strings.Contains(text, "```") || strings.Count(text, "\n") > 8 {
		score += 0.12
	}

	// Keyword hits (cap contribution)
	hits := 0
	for _, kw := range s.Keywords {
		if kw != "" && strings.Contains(lower, kw) {
			hits++
		}
	}
	if hits > 0 {
		score += float64(min(hits, 4)) * 0.08
	}

	// Question complexity: multiple interrogatives
	if strings.Count(lower, "?") >= 2 {
		score += 0.06
	}

	// Mixed scripts / heavy punctuation (rough proxy for dense tasks)
	if densePunctuation(text) {
		score += 0.05
	}

	if score > 1 {
		return 1
	}
	return score
}

func (s *Scorer) IsPremium(text string) bool {
	return s.ScoreMessage(text) >= s.Threshold
}

func densePunctuation(text string) bool {
	var punct int
	for _, r := range text {
		if unicode.IsPunct(r) {
			punct++
		}
	}
	return len(text) > 0 && float64(punct)/float64(len(text)) > 0.08
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

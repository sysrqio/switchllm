package classifier

import "testing"

func TestScoreMessage_empty(t *testing.T) {
	s := New(nil, 0.35)
	if got := s.ScoreMessage(""); got != 0 {
		t.Fatalf("expected 0, got %f", got)
	}
}

func TestScoreMessage_simpleShort(t *testing.T) {
	s := New([]string{"architecture"}, 0.35)
	score := s.ScoreMessage("hi there")
	if score >= 0.35 {
		t.Fatalf("short greeting should stay simple, score=%f", score)
	}
}

func TestScoreMessage_keywordAndLength(t *testing.T) {
	s := New([]string{"analyze", "security"}, 0.35)
	text := "Please analyze the security architecture of this multi-step deployment.\n" +
		"```go\nfunc main() {}\n```\n?" + stringsRepeat(" detail.", 80)
	score := s.ScoreMessage(text)
	if score < 0.35 {
		t.Fatalf("expected premium-level score, got %f", score)
	}
	if !s.IsPremium(text) {
		t.Fatal("expected IsPremium true")
	}
}

func stringsRepeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}

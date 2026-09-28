package router

import (
	"testing"

	"github.com/sysrqio/switchllm/internal/config"
)

func testConfig() *config.Config {
	cfg, _ := config.Load("../../switchllm.yaml")
	return cfg
}

func TestRoute_simple(t *testing.T) {
	cfg := testConfig()
	r := New(cfg, nil)
	d := r.Route("hello")
	if d.Premium {
		t.Fatalf("expected simple route, complexity=%f", d.Complexity)
	}
	if d.Model != cfg.Routing.SimpleModel {
		t.Fatalf("model=%s want %s", d.Model, cfg.Routing.SimpleModel)
	}
}

func TestRoute_premiumKeyword(t *testing.T) {
	cfg := testConfig()
	th := 0.35
	r := New(cfg, &th)
	d := r.Route("Please analyze, refactor and optimize the security architecture with formal proofs and multi-step reasoning?\nWhat is the threat model?\n```go\npackage main\n```")
	if !d.Premium {
		t.Fatalf("expected premium, complexity=%f", d.Complexity)
	}
	if d.Model != cfg.Routing.PremiumModel {
		t.Fatalf("model=%s want %s", d.Model, cfg.Routing.PremiumModel)
	}
}

func TestRoute_thresholdOverride(t *testing.T) {
	cfg := testConfig()
	th := 0.99
	r := New(cfg, &th)
	d := r.Route("analyze everything in depth with architecture review")
	if d.Premium {
		t.Fatalf("high threshold should keep simple, score=%f", d.Complexity)
	}
}

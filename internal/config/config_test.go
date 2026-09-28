package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_missingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestLoad_badYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("routing:\n  threshold: [not,a,scalar"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected parse error for invalid YAML")
	}
}

func TestLoad_appliesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "minimal.yaml")
	if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != 8080 {
		t.Fatalf("port=%d want 8080", cfg.Server.Port)
	}
	if cfg.Routing.Threshold != 0.35 {
		t.Fatalf("threshold=%v want 0.35", cfg.Routing.Threshold)
	}
	if cfg.Routing.SimpleModel != "gpt-4o-mini" {
		t.Fatalf("simple_model=%q", cfg.Routing.SimpleModel)
	}
	if cfg.Routing.PremiumModel != "gpt-4o" {
		t.Fatalf("premium_model=%q", cfg.Routing.PremiumModel)
	}
	if cfg.Providers.OpenAI.BaseURL != "https://api.openai.com/v1" {
		t.Fatalf("openai base_url=%q", cfg.Providers.OpenAI.BaseURL)
	}
	if len(cfg.ComplexityKeywords) == 0 {
		t.Fatal("expected default complexity keywords")
	}
}

func TestLoad_preservesExplicitValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "custom.yaml")
	yaml := `
server:
  port: 9090
routing:
  threshold: 0.5
  simple_model: custom-simple
  premium_model: custom-premium
providers:
  openai:
    base_url: http://localhost:1234/v1
complexity_keywords:
  - bespoke
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != 9090 {
		t.Fatalf("port=%d", cfg.Server.Port)
	}
	if cfg.Routing.Threshold != 0.5 {
		t.Fatalf("threshold=%v", cfg.Routing.Threshold)
	}
	if cfg.Routing.SimpleModel != "custom-simple" {
		t.Fatalf("simple_model=%q", cfg.Routing.SimpleModel)
	}
	if len(cfg.ComplexityKeywords) != 1 || cfg.ComplexityKeywords[0] != "bespoke" {
		t.Fatalf("keywords=%v", cfg.ComplexityKeywords)
	}
}

package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port int `yaml:"port"`
	} `yaml:"server"`
	Routing struct {
		Threshold   float64 `yaml:"threshold"`
		SimpleModel string  `yaml:"simple_model"`
		PremiumModel string `yaml:"premium_model"`
	} `yaml:"routing"`
	Providers struct {
		OpenAI struct {
			BaseURL string `yaml:"base_url"`
		} `yaml:"openai"`
		Anthropic struct {
			BaseURL string `yaml:"base_url"`
		} `yaml:"anthropic"`
	} `yaml:"providers"`
	ComplexityKeywords []string `yaml:"complexity_keywords"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	applyDefaults(&cfg)
	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Routing.Threshold == 0 {
		cfg.Routing.Threshold = 0.35
	}
	if cfg.Routing.SimpleModel == "" {
		cfg.Routing.SimpleModel = "gpt-4o-mini"
	}
	if cfg.Routing.PremiumModel == "" {
		cfg.Routing.PremiumModel = "gpt-4o"
	}
	if cfg.Providers.OpenAI.BaseURL == "" {
		cfg.Providers.OpenAI.BaseURL = "https://api.openai.com/v1"
	}
	if len(cfg.ComplexityKeywords) == 0 {
		cfg.ComplexityKeywords = []string{
			"analyze", "architecture", "refactor", "security",
			"prove", "theorem", "multi-step", "debug", "optimize",
		}
	}
}

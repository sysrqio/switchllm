package router

import (
	"github.com/sysrqio/switchllm/internal/classifier"
	"github.com/sysrqio/switchllm/internal/config"
)

type Decision struct {
	Model      string
	Complexity float64
	Premium    bool
}

type Router struct {
	cfg      *config.Config
	scorer   *classifier.Scorer
}

func New(cfg *config.Config, thresholdOverride *float64) *Router {
	th := cfg.Routing.Threshold
	if thresholdOverride != nil {
		th = *thresholdOverride
	}
	return &Router{
		cfg:    cfg,
		scorer: classifier.New(cfg.ComplexityKeywords, th),
	}
}

func (r *Router) Route(userText string) Decision {
	score := r.scorer.ScoreMessage(userText)
	premium := score >= r.scorer.Threshold
	model := r.cfg.Routing.SimpleModel
	if premium {
		model = r.cfg.Routing.PremiumModel
	}
	return Decision{
		Model:      model,
		Complexity: score,
		Premium:    premium,
	}
}

func (r *Router) PremiumModel() string {
	return r.cfg.Routing.PremiumModel
}

func (r *Router) SimpleModel() string {
	return r.cfg.Routing.SimpleModel
}

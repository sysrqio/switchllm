package metrics

import (
	"encoding/json"
	"sync/atomic"
)

type Snapshot struct {
	RequestsTotal       uint64  `json:"requests_total"`
	SimpleRouted        uint64  `json:"simple_routed"`
	PremiumRouted       uint64  `json:"premium_routed"`
	FallbacksTotal      uint64  `json:"fallbacks_total"`
	EstimatedSavingsUSD float64 `json:"estimated_savings_usd"`
	DryRun              bool    `json:"dry_run"`
}

type Collector struct {
	requests       atomic.Uint64
	simple         atomic.Uint64
	premium        atomic.Uint64
	fallbacks      atomic.Uint64
	savingsCents   atomic.Uint64 // stored as cents * 100 for fractional
	dryRun         atomic.Bool
	showSavings    bool
}

func New(showSavings bool) *Collector {
	c := &Collector{showSavings: showSavings}
	return c
}

func (c *Collector) SetDryRun(v bool) {
	c.dryRun.Store(v)
}

func (c *Collector) RecordRoute(premium bool, estimatedSaveUSD float64) {
	c.requests.Add(1)
	if premium {
		c.premium.Add(1)
	} else {
		c.simple.Add(1)
		if c.showSavings && estimatedSaveUSD > 0 {
			// store micro-dollars
			c.savingsCents.Add(uint64(estimatedSaveUSD * 1_000_000))
		}
	}
}

func (c *Collector) RecordFallback() {
	c.fallbacks.Add(1)
}

func (c *Collector) Snapshot() Snapshot {
	savings := float64(c.savingsCents.Load()) / 1_000_000
	return Snapshot{
		RequestsTotal:       c.requests.Load(),
		SimpleRouted:        c.simple.Load(),
		PremiumRouted:       c.premium.Load(),
		FallbacksTotal:      c.fallbacks.Load(),
		EstimatedSavingsUSD: savings,
		DryRun:              c.dryRun.Load(),
	}
}

func (c *Collector) JSON() ([]byte, error) {
	return json.Marshal(c.Snapshot())
}

package pricing

import "context"

// StaticProvider returns fixed rates configured in config.yaml.
type StaticProvider struct {
	rates UsageRates
}

func NewStaticProvider(cpuPerHour, memPerGBHour float64) *StaticProvider {
	return &StaticProvider{
		rates: UsageRates{
			CPUPerHour:   cpuPerHour,
			MemPerGBHour: memPerGBHour,
		},
	}
}

func (p *StaticProvider) UsageRates(_ context.Context) (UsageRates, error) {
	return p.rates, nil
}

func (p *StaticProvider) Source() string {
	return "config"
}

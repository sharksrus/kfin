package pricing

import "context"

// UsageRates are cloud usage-based rates used by kfin history calculations.
type UsageRates struct {
	CPUPerHour   float64
	MemPerGBHour float64
}

// Provider supplies pricing rates from a backing source (config, MCP, etc.).
type Provider interface {
	UsageRates(ctx context.Context) (UsageRates, error)
	Source() string
}

package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Pricing PricingConfig `yaml:"pricing"`
	Stats   StatsConfig   `yaml:"stats"`
}

type PricingConfig struct {
	HardwareMonthlyPerGB  float64            `yaml:"hardware_monthly_per_gb"`
	ElectricityRate       float64            `yaml:"electricity_rate"` // $/kWh
	WattsPerNode          float64            `yaml:"watts_per_node"`   // watts
	InstanceMonthlyByType map[string]float64 `yaml:"instance_monthly_by_type"`
	EKS                   EKSPricingConfig   `yaml:"eks"`
	MCP                   MCPPricingConfig   `yaml:"mcp"`
	Cloud                 CloudPricing       `yaml:"cloud"`
}

type CloudPricing struct {
	CPUPerHour   float64 `yaml:"cpu_per_hour"`    // $/vCPU/hour
	MemPerGBHour float64 `yaml:"mem_per_gb_hour"` // $/GB/hour
}

type MCPPricingConfig struct {
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
}

type EKSPricingConfig struct {
	ControlPlanePerHour float64 `yaml:"control_plane_per_hour"`
}

type StatsConfig struct {
	BaseURL              string `yaml:"base_url"`
	QueryTimeoutSeconds  int    `yaml:"query_timeout_seconds"`
	DefaultLookbackHours int    `yaml:"default_lookback_hours"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := *DefaultConfig()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.Stats.QueryTimeoutSeconds <= 0 {
		cfg.Stats.QueryTimeoutSeconds = 15
	}
	if cfg.Stats.DefaultLookbackHours <= 0 {
		cfg.Stats.DefaultLookbackHours = 24
	}

	return &cfg, nil
}

func DefaultConfig() *Config {
	return &Config{
		Pricing: PricingConfig{
			HardwareMonthlyPerGB:  0.26,
			ElectricityRate:       0.12,
			WattsPerNode:          15,
			InstanceMonthlyByType: map[string]float64{},
			EKS: EKSPricingConfig{
				ControlPlanePerHour: 0.10,
			},
			MCP: MCPPricingConfig{
				Command: "",
				Args:    []string{},
			},
			Cloud: CloudPricing{
				CPUPerHour:   0.025,
				MemPerGBHour: 0.006,
			},
		},
		Stats: StatsConfig{
			BaseURL:              "",
			QueryTimeoutSeconds:  15,
			DefaultLookbackHours: 24,
		},
	}
}

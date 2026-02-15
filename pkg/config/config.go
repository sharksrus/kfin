package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Pricing PricingConfig `yaml:"pricing"`
}

type PricingConfig struct {
	HardwareMonthlyPerGB float64      `yaml:"hardware_monthly_per_gb"`
	ElectricityRate      float64      `yaml:"electricity_rate"` // $/kWh
	WattsPerNode         float64      `yaml:"watts_per_node"`   // watts
	Cloud                CloudPricing `yaml:"cloud"`
}

type CloudPricing struct {
	CPUPerHour   float64 `yaml:"cpu_per_hour"`    // $/vCPU/hour
	MemPerGBHour float64 `yaml:"mem_per_gb_hour"` // $/GB/hour
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func DefaultConfig() *Config {
	return &Config{
		Pricing: PricingConfig{
			HardwareMonthlyPerGB: 0.26,
			ElectricityRate:      0.12,
			WattsPerNode:         15,
			Cloud: CloudPricing{
				CPUPerHour:   0.025,
				MemPerGBHour: 0.006,
			},
		},
	}
}

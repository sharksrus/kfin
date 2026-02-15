package cmd

import (
	"testing"

	"github.com/newman-bot/kfin/pkg/config"
)

func TestBuildPricingProvider_ConfigSource(t *testing.T) {
	prev := cfg
	cfg = config.DefaultConfig()
	t.Cleanup(func() { cfg = prev })

	p, err := buildPricingProvider("config", "", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := p.Source(); got != "config" {
		t.Fatalf("expected source=config, got %s", got)
	}
}

func TestBuildPricingProvider_MCPRequiresCommand(t *testing.T) {
	prev := cfg
	cfg = config.DefaultConfig()
	t.Cleanup(func() { cfg = prev })

	_, err := buildPricingProvider("mcp", "", nil)
	if err == nil {
		t.Fatalf("expected error for missing mcp command")
	}
}

func TestBuildPricingProvider_MCPWithCommand(t *testing.T) {
	prev := cfg
	cfg = config.DefaultConfig()
	t.Cleanup(func() { cfg = prev })

	p, err := buildPricingProvider("mcp", "/bin/echo", []string{"{\"cpu_per_hour\":0.01,\"mem_per_gb_hour\":0.001}"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := p.Source(); got != "mcp" {
		t.Fatalf("expected source=mcp, got %s", got)
	}
}

func TestBuildPricingProvider_InvalidSource(t *testing.T) {
	prev := cfg
	cfg = config.DefaultConfig()
	t.Cleanup(func() { cfg = prev })

	_, err := buildPricingProvider("nope", "", nil)
	if err == nil {
		t.Fatalf("expected error for invalid source")
	}
}


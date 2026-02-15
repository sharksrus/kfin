package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// MCPProvider resolves pricing by executing an external command (typically an MCP client wrapper)
// that prints JSON like: {"cpu_per_hour":0.031,"mem_per_gb_hour":0.004}
type MCPProvider struct {
	command string
	args    []string
}

type mcpRatesOutput struct {
	CPUPerHour   float64 `json:"cpu_per_hour"`
	MemPerGBHour float64 `json:"mem_per_gb_hour"`
}

func NewMCPProvider(command string, args []string) *MCPProvider {
	return &MCPProvider{command: strings.TrimSpace(command), args: args}
}

func (p *MCPProvider) UsageRates(ctx context.Context) (UsageRates, error) {
	if p.command == "" {
		return UsageRates{}, fmt.Errorf("mcp pricing command is empty")
	}

	cmd := exec.CommandContext(ctx, p.command, p.args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return UsageRates{}, fmt.Errorf("run mcp pricing command: %w (output: %s)", err, strings.TrimSpace(string(out)))
	}

	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return UsageRates{}, fmt.Errorf("mcp pricing command produced empty output")
	}

	parsed, err := parseMCPRatesOutput(raw)
	if err != nil {
		return UsageRates{}, err
	}
	if parsed.CPUPerHour <= 0 || parsed.MemPerGBHour <= 0 {
		return UsageRates{}, fmt.Errorf("mcp pricing returned non-positive rates: cpu_per_hour=%.6f mem_per_gb_hour=%.6f", parsed.CPUPerHour, parsed.MemPerGBHour)
	}

	return UsageRates{
		CPUPerHour:   parsed.CPUPerHour,
		MemPerGBHour: parsed.MemPerGBHour,
	}, nil
}

func (p *MCPProvider) Source() string {
	return "mcp"
}

func parseMCPRatesOutput(raw string) (mcpRatesOutput, error) {
	var parsed mcpRatesOutput
	if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
		return parsed, nil
	}

	// Allow wrappers that log before final JSON by parsing the last non-empty line.
	lines := strings.Split(raw, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if err := json.Unmarshal([]byte(line), &parsed); err == nil {
			return parsed, nil
		}
	}

	return mcpRatesOutput{}, fmt.Errorf("failed to parse mcp pricing output as JSON: %s", raw)
}

package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/newman-bot/kfin/pkg/stats"
	"github.com/spf13/cobra"
)

const (
	defaultCPUQuery = `sum(rate(container_cpu_usage_seconds_total{container!="",pod!=""}[5m]))`
	defaultMemQuery = `sum(container_memory_working_set_bytes{container!="",pod!=""})`
)

func HistoryCmd() *cobra.Command {
	lookbackHours := cfg.Stats.DefaultLookbackHours
	step := "5m"
	debug := false

	cmd := &cobra.Command{
		Use:   "history",
		Short: "Analyze historical cluster usage from Prometheus-compatible stats API",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHistory(lookbackHours, step, debug)
		},
	}

	cmd.Flags().IntVar(&lookbackHours, "hours", lookbackHours, "Lookback window in hours")
	cmd.Flags().StringVar(&step, "step", step, "Query step duration (for example: 1m, 5m, 15m)")
	cmd.Flags().BoolVar(&debug, "debug", debug, "Print query URLs and returned series/point details")

	return cmd
}

func runHistory(lookbackHours int, step string, debug bool) error {
	baseURL := strings.TrimSpace(cfg.Stats.BaseURL)
	if baseURL == "" {
		return fmt.Errorf("stats.base_url is empty; set it in config.yaml (example: http://stats.kramerica.ai)")
	}
	if lookbackHours <= 0 {
		return fmt.Errorf("--hours must be greater than 0")
	}

	stepDur, err := time.ParseDuration(step)
	if err != nil {
		return fmt.Errorf("invalid --step %q: %w", step, err)
	}
	if stepDur <= 0 {
		return fmt.Errorf("--step must be greater than 0")
	}

	timeout := time.Duration(cfg.Stats.QueryTimeoutSeconds) * time.Second
	client, err := stats.NewClient(baseURL, timeout)
	if err != nil {
		return err
	}

	end := time.Now()
	start := end.Add(-time.Duration(lookbackHours) * time.Hour)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cpuURL, err := client.QueryRangeURL(defaultCPUQuery, start, end, stepDur)
	if err != nil {
		return fmt.Errorf("build cpu query URL: %w", err)
	}
	memURL, err := client.QueryRangeURL(defaultMemQuery, start, end, stepDur)
	if err != nil {
		return fmt.Errorf("build memory query URL: %w", err)
	}

	if debug {
		fmt.Printf("Debug\n")
		fmt.Printf("=====\n")
		fmt.Printf("CPU query URL: %s\n", cpuURL)
		fmt.Printf("Memory query URL: %s\n\n", memURL)
	}

	cpuResp, err := client.QueryRange(ctx, defaultCPUQuery, start, end, stepDur)
	if err != nil {
		return fmt.Errorf("query cpu usage: %w", err)
	}
	memResp, err := client.QueryRange(ctx, defaultMemQuery, start, end, stepDur)
	if err != nil {
		return fmt.Errorf("query memory usage: %w", err)
	}

	avgCPU, cpuSamples, err := stats.AverageSeriesValue(cpuResp)
	if err != nil {
		return fmt.Errorf("parse cpu usage response: %w", err)
	}
	avgMemBytes, memSamples, err := stats.AverageSeriesValue(memResp)
	if err != nil {
		return fmt.Errorf("parse memory usage response: %w", err)
	}

	avgMemGB := avgMemBytes / (1024 * 1024 * 1024)
	monthlyCPUCost := avgCPU * 730 * cfg.Pricing.Cloud.CPUPerHour
	monthlyMemCost := avgMemGB * 730 * cfg.Pricing.Cloud.MemPerGBHour
	cpuPointStats := stats.GetSeriesPointStats(cpuResp)
	memPointStats := stats.GetSeriesPointStats(memResp)

	fmt.Printf("Historical Usage Summary\n")
	fmt.Printf("========================\n")
	fmt.Printf("Endpoint: %s\n", baseURL)
	fmt.Printf("Window:   %s to %s (%dh)\n", start.Format(time.RFC3339), end.Format(time.RFC3339), lookbackHours)
	fmt.Printf("Step:     %s\n\n", stepDur)

	fmt.Printf("Avg CPU usage:    %.3f cores  (%d samples)\n", avgCPU, cpuSamples)
	fmt.Printf("Avg Memory usage: %.3f GB     (%d samples)\n\n", avgMemGB, memSamples)
	if debug {
		fmt.Printf("CPU series:       %d (points total=%d, min=%d, max=%d)\n",
			cpuPointStats.Series, cpuPointStats.TotalPoints, cpuPointStats.MinPoints, cpuPointStats.MaxPoints)
		fmt.Printf("Memory series:    %d (points total=%d, min=%d, max=%d)\n\n",
			memPointStats.Series, memPointStats.TotalPoints, memPointStats.MinPoints, memPointStats.MaxPoints)
	}

	fmt.Printf("Estimated monthly usage-based cost (cloud pricing)\n")
	fmt.Printf("CPU:              $%.2f\n", monthlyCPUCost)
	fmt.Printf("Memory:           $%.2f\n", monthlyMemCost)
	fmt.Printf("Total:            $%.2f\n", monthlyCPUCost+monthlyMemCost)

	return nil
}

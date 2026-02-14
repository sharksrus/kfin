package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/newman-bot/pod-cost-analyzer/cmd"
)

var rootCmd = &cobra.Command{
	Use:   "pod-cost-analyzer",
	Short: "Analyze pod costs in K3s/Kubernetes clusters",
	Long: `Pod Cost Analyzer tracks resource usage and estimates costs for pods in your K3s cluster.

Identifies waste, suggests rightsizing, and helps optimize your infrastructure spending.`,
}

func init() {
	rootCmd.AddCommand(cmd.AnalyzeCmd())
	rootCmd.AddCommand(cmd.StatusCmd())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

package main

import (
	"fmt"
	"os"

	"github.com/newman-bot/kfin/cmd"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kfin",
	Short: "Analyze pod costs in K3s/Kubernetes clusters",
	Long: `Pod Cost Analyzer tracks resource usage and estimates costs for pods in your K3s cluster.

Identifies waste, suggests rightsizing, and helps optimize your infrastructure spending.`,
}

func init() {
	rootCmd.AddCommand(cmd.AnalyzeCmd())
	rootCmd.AddCommand(cmd.StatusCmd())
	rootCmd.AddCommand(cmd.TuiCmd())
	rootCmd.AddCommand(cmd.PdfCmd())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

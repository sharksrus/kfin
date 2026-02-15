package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/newman-bot/kfin/cmd"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kfin",
	Short: "Analyze pod costs in K3s/Kubernetes clusters",
	Long: `Pod Cost Analyzer tracks resource usage and estimates costs for pods in your K3s cluster.

Identifies waste, suggests rightsizing, and helps optimize your infrastructure spending.

Running 'kfin' with no subcommand opens the interactive TUI dashboard by default.`,
}

var version = "dev"
var buildNumber = ""

func init() {
	tuiCmd := cmd.TuiCmd()
	rootCmd.Run = func(c *cobra.Command, args []string) {
		showVersion, _ := c.Flags().GetBool("version")
		if showVersion {
			v := version
			if buildNumber != "" {
				v = fmt.Sprintf("%s+%s", version, buildNumber)
			}
			fmt.Printf("kfin %s (%s/%s)\n", v, runtime.GOOS, runtime.GOARCH)
			return
		}

		if tuiCmd.Run != nil {
			tuiCmd.Run(tuiCmd, args)
		}
	}
	rootCmd.Flags().BoolP("version", "v", false, "Print kfin version")

	rootCmd.AddCommand(cmd.AnalyzeCmd())
	rootCmd.AddCommand(cmd.HistoryCmd())
	rootCmd.AddCommand(cmd.StatusCmd())
	rootCmd.AddCommand(tuiCmd)
	rootCmd.AddCommand(cmd.PdfCmd())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

package cmd

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/newman-bot/kfin/pkg/pdf"
	"github.com/newman-bot/kfin/pkg/stats"
	"github.com/newman-bot/kfin/pkg/tui"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func TuiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Open interactive TUI dashboard",
		Run: func(cmd *cobra.Command, args []string) {
			runTui()
		},
	}
}

func PdfCmd() *cobra.Command {
	var output string
	pdfCmd := &cobra.Command{
		Use:   "pdf",
		Short: "Export cost report to PDF",
		Run: func(cmd *cobra.Command, args []string) {
			runPdf(output)
		},
	}
	pdfCmd.Flags().StringVarP(&output, "output", "o", "kfin-report.pdf", "Output PDF filename")
	return pdfCmd
}

func runTui() {
	clientset, err := getClientset()
	if err != nil {
		log.Fatalf("Failed to connect to cluster: %v", err)
	}
	resolveUsageRates()
	contextName, clusterName := getKubeContextDetails()

	ctx := context.Background()

	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Failed to list pods: %v", err)
	}

	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Failed to list nodes: %v", err)
	}

	// Collect data
	podCosts := collectPodCosts(pods.Items)
	nodeInfo := collectNodeInfo(nodes.Items)
	hardwareCost, elecCost, controlPlaneCost := calculateClusterCosts(nodes.Items)
	statsFreshness := collectStatsFreshness()

	data := tui.ReportData{
		PodCosts:         podCosts,
		HardwareCost:     hardwareCost,
		ElecCost:         elecCost,
		ControlPlaneCost: controlPlaneCost,
		TotalCost:        hardwareCost + elecCost + controlPlaneCost,
		Nodes:            nodeInfo,
		ContextName:      contextName,
		ClusterName:      clusterName,
		PricingSource:    activeUsageRatesSource,
		StatsFreshness:   statsFreshness,
	}

	tui.ShowDashboard(data)
}

func collectStatsFreshness() tui.StatsFreshness {
	baseURL := strings.TrimSpace(cfg.Stats.BaseURL)
	if baseURL == "" {
		return tui.StatsFreshness{
			Ready: false,
			Note:  "No Prometheus endpoint configured",
		}
	}

	lookbackHours := cfg.Stats.DefaultLookbackHours
	if lookbackHours <= 0 {
		lookbackHours = 24
	}

	timeout := time.Duration(cfg.Stats.QueryTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	client, err := stats.NewClient(baseURL, timeout)
	if err != nil {
		return tui.StatsFreshness{
			Ready: false,
			Note:  fmt.Sprintf("Stats client error: %v", err),
		}
	}

	end := time.Now()
	start := end.Add(-time.Duration(lookbackHours) * time.Hour)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resp, err := client.QueryRange(ctx, defaultCPUQuery, start, end, 5*time.Minute)
	if err != nil {
		return tui.StatsFreshness{
			Ready: false,
			Note:  fmt.Sprintf("Stats query failed: %v", err),
		}
	}

	pointStats := stats.GetSeriesPointStats(resp)
	coverage := stats.GetCoverageStats(resp)
	if !coverage.HasPoints {
		return tui.StatsFreshness{
			Ready: false,
			Note:  "No Prometheus datapoints returned",
		}
	}

	return tui.StatsFreshness{
		Ready:            true,
		BaseURL:          baseURL,
		LookbackDuration: time.Duration(lookbackHours) * time.Hour,
		ObservedDuration: coverage.ObservedDuration,
		SampleCount:      pointStats.TotalPoints,
		LastSampleAt:     coverage.Latest,
	}
}

func runPdf(output string) {
	clientset, err := getClientset()
	if err != nil {
		log.Fatalf("Failed to connect to cluster: %v", err)
	}
	resolveUsageRates()
	contextName, clusterName := getKubeContextDetails()

	ctx := context.Background()

	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Failed to list pods: %v", err)
	}

	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Failed to list nodes: %v", err)
	}

	// Collect data
	podCosts := collectPodCostsForPdf(pods.Items)
	nodeInfo := collectNodeInfoForPdf(nodes.Items)
	hardwareCost, elecCost, controlPlaneCost := calculateClusterCosts(nodes.Items)

	data := pdf.ReportData{
		PodCosts:         podCosts,
		HardwareCost:     hardwareCost,
		ElecCost:         elecCost,
		ControlPlaneCost: controlPlaneCost,
		TotalCost:        hardwareCost + elecCost + controlPlaneCost,
		Nodes:            nodeInfo,
		GeneratedAt:      time.Now(),
		ContextName:      contextName,
		ClusterName:      clusterName,
	}

	if err := pdf.Generate(data, output); err != nil {
		log.Fatalf("Failed to generate PDF: %v", err)
	}

	fmt.Printf("PDF report saved to: %s\n", output)
}

func getClientset() (*kubernetes.Clientset, error) {
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	k8sConfig, err := kubeconfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(k8sConfig)
}

func getKubeContextDetails() (string, string) {
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
	rawCfg, err := kubeconfig.RawConfig()
	if err != nil {
		return "unknown", "unknown"
	}

	contextName := rawCfg.CurrentContext
	if contextName == "" {
		contextName = "unknown"
	}
	clusterName := "unknown"
	if ctx, ok := rawCfg.Contexts[rawCfg.CurrentContext]; ok && ctx != nil && ctx.Cluster != "" {
		clusterName = ctx.Cluster
	}
	return contextName, clusterName
}

package cmd

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/newman-bot/kfin/pkg/pdf"
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
	hardwareCost, elecCost := calculateClusterCosts(nodes.Items)

	data := tui.ReportData{
		PodCosts:     podCosts,
		HardwareCost: hardwareCost,
		ElecCost:     elecCost,
		TotalCost:    hardwareCost + elecCost,
		Nodes:        nodeInfo,
		ContextName:  contextName,
		ClusterName:  clusterName,
	}

	tui.ShowDashboard(data)
}

func runPdf(output string) {
	clientset, err := getClientset()
	if err != nil {
		log.Fatalf("Failed to connect to cluster: %v", err)
	}
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
	hardwareCost, elecCost := calculateClusterCosts(nodes.Items)

	data := pdf.ReportData{
		PodCosts:     podCosts,
		HardwareCost: hardwareCost,
		ElecCost:     elecCost,
		TotalCost:    hardwareCost + elecCost,
		Nodes:        nodeInfo,
		GeneratedAt:  time.Now(),
		ContextName:  contextName,
		ClusterName:  clusterName,
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

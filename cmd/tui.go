package cmd

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"github.com/newman-bot/kfin/pkg/pdf"
	"github.com/newman-bot/kfin/pkg/tui"
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
	}

	tui.ShowDashboard(data)
}

func runPdf(output string) {
	clientset, err := getClientset()
	if err != nil {
		log.Fatalf("Failed to connect to cluster: %v", err)
	}

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
	}

	if err := pdf.Generate(data, output); err != nil {
		log.Fatalf("Failed to generate PDF: %v", err)
	}

	fmt.Printf("PDF report saved to: %s\n", output)
}

func getClientset() (*kubernetes.Clientset, error) {
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: clientcmd.RecommendedHomeFile},
		&clientcmd.ConfigOverrides{},
	)

	k8sConfig, err := kubeconfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(k8sConfig)
}

package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"github.com/newman-bot/kfin/pkg/config"
	"github.com/newman-bot/kfin/pkg/pdf"
	"github.com/newman-bot/kfin/pkg/tui"
)

var cfg *config.Config

func init() {
	// Try to load config, fall back to defaults
	var err error
	cfg, err = config.Load("config.yaml")
	if err != nil {
		cfg = config.DefaultConfig()
	}
}

func AnalyzeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "analyze",
		Short: "Analyze pod costs in the cluster",
		Run: func(cmd *cobra.Command, args []string) {
			analyzeCluster()
		},
	}
}

func analyzeCluster() {
	// Load kubeconfig
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: clientcmd.RecommendedHomeFile},
		&clientcmd.ConfigOverrides{},
	)

	k8sConfig, err := kubeconfig.ClientConfig()
	if err != nil {
		log.Fatalf("Failed to load kubeconfig: %v", err)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		log.Fatalf("Failed to create kubernetes client: %v", err)
	}

	ctx := context.Background()

	// List all pods across all namespaces
	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Failed to list pods: %v", err)
	}

	// Get nodes for hardware cost calculation
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Failed to list nodes: %v", err)
	}

	// Calculate total hardware costs
	totalMemGB := calculateTotalMemoryGB(nodes.Items)
	hardwareCostMonthly := totalMemGB * cfg.Pricing.HardwareMonthlyPerGB
	electricityCostMonthly := float64(len(nodes.Items)) * cfg.Pricing.WattsPerNode / 1000.0 * 730 * cfg.Pricing.ElectricityRate // 730 hours/month

	fmt.Printf("Found %d pods across %d nodes\n\n", len(pods.Items), len(nodes.Items))
	
	// Print cost summary
	fmt.Printf("=== Monthly Cost Summary ===\n")
	fmt.Printf("Hardware (amortized): $%.2f\n", hardwareCostMonthly)
	fmt.Printf("Electricity:         $%.2f\n", electricityCostMonthly)
	fmt.Printf("Total:               $%.2f\n\n", hardwareCostMonthly+electricityCostMonthly)

	fmt.Printf("%-40s %-15s %-12s %-12s %-12s\n", "POD", "NAMESPACE", "CPU REQ", "MEM REQ", "MONTHLY $")
	fmt.Println("================================================================================")

	var totalCPU, totalMem resource.Quantity

	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			cpu := container.Resources.Requests.Cpu()
			mem := container.Resources.Requests.Memory()
			
			// Add to totals
			totalCPU.Add(*cpu)
			totalMem.Add(*mem)

			// Calculate cost for this container
			cost := calculateContainerCost(cpu, mem)
			
			// Only show containers with requests
			if cost > 0 {
				fmt.Printf("%-40s %-15s %-12s %-12s $%-11.2f\n", 
					truncate(container.Name, 40),
					pod.Namespace, 
					cpu.String(), 
					mem.String(),
					cost)
			}
		}
	}

	fmt.Println("================================================================================")
	totalCost := calculateContainerCost(&totalCPU, &totalMem)
	fmt.Printf("%-40s %-15s %-12s %-12s $%-11.2f\n", 
		"TOTAL", "", totalCPU.String(), totalMem.String(), totalCost)
	
	// Per-node breakdown
	fmt.Printf("\n=== Node Hardware Costs (monthly) ===\n")
	for _, node := range nodes.Items {
		memGB := float64(node.Status.Capacity.Memory().Value()) / (1024 * 1024 * 1024)
		nodeCost := memGB * cfg.Pricing.HardwareMonthlyPerGB
		elecCost := cfg.Pricing.WattsPerNode / 1000.0 * 730 * cfg.Pricing.ElectricityRate
		fmt.Printf("%s: $%.2f (hardware) + $%.2f (electricity) = $%.2f/month\n", 
			node.Name, nodeCost, elecCost, nodeCost+elecCost)
	}
}

func calculateTotalMemoryGB(nodes []corev1.Node) float64 {
	var total int64
	for _, node := range nodes {
		mem := node.Status.Capacity.Memory().Value()
		total += mem
	}
	return float64(total) / (1024 * 1024 * 1024)
}

func calculateContainerCost(cpu *resource.Quantity, mem *resource.Quantity) float64 {
	// Cost based on hardware allocation (memory-based)
	memGB := float64(mem.Value()) / (1024 * 1024 * 1024)
	
	// Hardware cost (allocated portion)
	hardwareCost := memGB * cfg.Pricing.HardwareMonthlyPerGB
	
	return hardwareCost
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func collectPodCosts(pods []corev1.Pod) []tui.PodInfo {
	var result []tui.PodInfo
	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {
			cpu := container.Resources.Requests.Cpu()
			mem := container.Resources.Requests.Memory()
			cost := calculateContainerCost(cpu, mem)

			result = append(result, tui.PodInfo{
				Name:      container.Name,
				Namespace: pod.Namespace,
				CPU:       cpu.String(),
				Memory:    mem.String(),
				Cost:      cost,
			})
		}
	}
	return result
}

func collectPodCostsForPdf(pods []corev1.Pod) []pdf.PodCost {
	var result []pdf.PodCost
	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {
			cpu := container.Resources.Requests.Cpu()
			mem := container.Resources.Requests.Memory()
			cost := calculateContainerCost(cpu, mem)

			result = append(result, pdf.PodCost{
				Name:      container.Name,
				Namespace: pod.Namespace,
				CPU:       cpu.String(),
				Memory:    mem.String(),
				Cost:      cost,
			})
		}
	}
	return result
}

func collectNodeInfo(nodes []corev1.Node) []tui.NodeInfo {
	var result []tui.NodeInfo
	for _, node := range nodes {
		memGB := float64(node.Status.Capacity.Memory().Value()) / (1024 * 1024 * 1024)
		hardwareCost := memGB * cfg.Pricing.HardwareMonthlyPerGB
		elecCost := cfg.Pricing.WattsPerNode / 1000.0 * 730 * cfg.Pricing.ElectricityRate

		result = append(result, tui.NodeInfo{
			Name:         node.Name,
			MemoryGB:     memGB,
			HardwareCost: hardwareCost,
			ElecCost:     elecCost,
			TotalCost:    hardwareCost + elecCost,
		})
	}
	return result
}

func collectNodeInfoForPdf(nodes []corev1.Node) []pdf.NodeInfo {
	var result []pdf.NodeInfo
	for _, node := range nodes {
		memGB := float64(node.Status.Capacity.Memory().Value()) / (1024 * 1024 * 1024)
		hardwareCost := memGB * cfg.Pricing.HardwareMonthlyPerGB
		elecCost := cfg.Pricing.WattsPerNode / 1000.0 * 730 * cfg.Pricing.ElectricityRate

		result = append(result, pdf.NodeInfo{
			Name:         node.Name,
			MemoryGB:     memGB,
			HardwareCost: hardwareCost,
			ElecCost:     elecCost,
			TotalCost:    hardwareCost + elecCost,
		})
	}
	return result
}

func calculateClusterCosts(nodes []corev1.Node) (float64, float64) {
	var totalMemGB float64
	for _, node := range nodes {
		memGB := float64(node.Status.Capacity.Memory().Value()) / (1024 * 1024 * 1024)
		totalMemGB += memGB
	}

	hardwareCost := totalMemGB * cfg.Pricing.HardwareMonthlyPerGB
	elecCost := float64(len(nodes)) * cfg.Pricing.WattsPerNode / 1000.0 * 730 * cfg.Pricing.ElectricityRate

	return hardwareCost, elecCost
}

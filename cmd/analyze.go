package cmd

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/newman-bot/kfin/pkg/config"
	"github.com/newman-bot/kfin/pkg/pdf"
	"github.com/newman-bot/kfin/pkg/pricing"
	"github.com/newman-bot/kfin/pkg/tui"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const nodeInstanceTypeLabel = "node.kubernetes.io/instance-type"

var cfg *config.Config
var activeUsageRates pricing.UsageRates
var activeUsageRatesSource = "config"

func init() {
	// Try to load config, fall back to defaults
	var err error
	cfg, err = config.Load("config.yaml")
	if err != nil {
		cfg = config.DefaultConfig()
	}
	activeUsageRates = pricing.UsageRates{
		CPUPerHour:   cfg.Pricing.Cloud.CPUPerHour,
		MemPerGBHour: cfg.Pricing.Cloud.MemPerGBHour,
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

	resolveUsageRates()

	// Calculate total hardware costs
	hardwareCostMonthly, electricityCostMonthly, controlPlaneMonthly := calculateClusterCosts(nodes.Items)
	totalMonthly := hardwareCostMonthly + electricityCostMonthly + controlPlaneMonthly

	fmt.Printf("Found %d pods across %d nodes\n\n", len(pods.Items), len(nodes.Items))

	// Print cost summary
	fmt.Printf("=== Monthly Cost Summary ===\n")
	fmt.Printf("Hardware (amortized): $%.2f\n", hardwareCostMonthly)
	fmt.Printf("Electricity:         $%.2f\n", electricityCostMonthly)
	fmt.Printf("EKS control plane:   $%.2f\n", controlPlaneMonthly)
	fmt.Printf("Total:               $%.2f\n", totalMonthly)
	fmt.Printf("Pod pricing source:  %s (cpu_per_hour=%.6f, mem_per_gb_hour=%.6f)\n\n",
		activeUsageRatesSource, activeUsageRates.CPUPerHour, activeUsageRates.MemPerGBHour)

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
		nodeCost, instanceType, usedInstanceOverride := calculateNodeHardwareCost(node)
		elecCost := cfg.Pricing.WattsPerNode / 1000.0 * 730 * cfg.Pricing.ElectricityRate
		if usedInstanceOverride {
			fmt.Printf("%s (%s): $%.2f (hardware) + $%.2f (electricity) = $%.2f/month\n",
				node.Name, instanceType, nodeCost, elecCost, nodeCost+elecCost)
			continue
		}
		fmt.Printf("%s: $%.2f (hardware) + $%.2f (electricity) = $%.2f/month\n",
			node.Name, nodeCost, elecCost, nodeCost+elecCost)
	}
}

func calculateNodeHardwareCost(node corev1.Node) (float64, string, bool) {
	instanceType := node.Labels[nodeInstanceTypeLabel]
	if instanceType != "" {
		if monthly, ok := cfg.Pricing.InstanceMonthlyByType[instanceType]; ok && monthly > 0 {
			return monthly, instanceType, true
		}
	}

	memGB := float64(node.Status.Capacity.Memory().Value()) / (1024 * 1024 * 1024)
	return memGB * cfg.Pricing.HardwareMonthlyPerGB, instanceType, false
}

func calculateContainerCost(cpu *resource.Quantity, mem *resource.Quantity) float64 {
	cpuCores := float64(cpu.MilliValue()) / 1000.0
	memGB := float64(mem.Value()) / (1024 * 1024 * 1024)

	monthlyCPUCost := cpuCores * 730 * activeUsageRates.CPUPerHour
	monthlyMemCost := memGB * 730 * activeUsageRates.MemPerGBHour

	return monthlyCPUCost + monthlyMemCost
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
		hardwareCost, _, _ := calculateNodeHardwareCost(node)
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
		hardwareCost, _, _ := calculateNodeHardwareCost(node)
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

func calculateClusterCosts(nodes []corev1.Node) (float64, float64, float64) {
	var hardwareCost float64
	for _, node := range nodes {
		nodeHardware, _, _ := calculateNodeHardwareCost(node)
		hardwareCost += nodeHardware
	}

	elecCost := float64(len(nodes)) * cfg.Pricing.WattsPerNode / 1000.0 * 730 * cfg.Pricing.ElectricityRate
	controlPlaneCost := 0.0
	if isEKSCluster(nodes) {
		controlPlaneCost = 730 * cfg.Pricing.EKS.ControlPlanePerHour
	}

	return hardwareCost, elecCost, controlPlaneCost
}

func isEKSCluster(nodes []corev1.Node) bool {
	for _, node := range nodes {
		for k := range node.Labels {
			if strings.HasPrefix(k, "eks.amazonaws.com/") {
				return true
			}
		}
		if strings.HasPrefix(node.Spec.ProviderID, "aws://") {
			return true
		}
	}
	return false
}

func resolveUsageRates() {
	base := pricing.NewStaticProvider(cfg.Pricing.Cloud.CPUPerHour, cfg.Pricing.Cloud.MemPerGBHour)
	activeUsageRatesSource = base.Source()

	cmd := strings.TrimSpace(cfg.Pricing.MCP.Command)
	if cmd == "" {
		rates, _ := base.UsageRates(context.Background())
		activeUsageRates = rates
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rates, err := pricing.NewMCPProvider(cmd, cfg.Pricing.MCP.Args).UsageRates(ctx)
	if err != nil {
		log.Printf("warning: mcp pricing failed, falling back to config rates: %v", err)
		rates, _ = base.UsageRates(context.Background())
		activeUsageRatesSource = base.Source()
		activeUsageRates = rates
		return
	}
	activeUsageRatesSource = "mcp"
	activeUsageRates = rates
}

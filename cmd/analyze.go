package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

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

	config, err := kubeconfig.ClientConfig()
	if err != nil {
		log.Fatalf("Failed to load kubeconfig: %v", err)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create kubernetes client: %v", err)
	}

	ctx := context.Background()

	// List all pods across all namespaces
	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Failed to list pods: %v", err)
	}

	fmt.Printf("Found %d pods\n\n", len(pods.Items))
	fmt.Printf("%-40s %-15s %-15s %-15s\n", "POD", "NAMESPACE", "CPU REQ", "MEM REQ")
	fmt.Println("================================================================================")

	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			cpu := container.Resources.Requests.Cpu().String()
			mem := container.Resources.Requests.Memory().String()
			fmt.Printf("%-40s %-15s %-15s %-15s\n", 
				container.Name[:min(40, len(container.Name))],
				pod.Namespace, cpu, mem)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

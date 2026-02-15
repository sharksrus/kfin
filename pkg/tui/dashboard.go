package tui

import (
	"fmt"

	"github.com/rivo/tview"
)

type PodInfo struct {
	Name      string
	Namespace string
	CPU       string
	Memory    string
	Cost      float64
}

type NodeInfo struct {
	Name         string
	MemoryGB     float64
	HardwareCost float64
	ElecCost     float64
	TotalCost    float64
}

type ReportData struct {
	PodCosts     []PodInfo
	TotalCost    float64
	HardwareCost float64
	ElecCost     float64
	Nodes        []NodeInfo
}

func ShowDashboard(data ReportData) {
	app := tview.NewApplication()

	// Group pods by namespace
	namespaces := getNamespaces(data.PodCosts)
	namespaces = append([]string{"all"}, namespaces...)

	// Create text input for namespace filter
	inputField := tview.NewInputField().
		SetLabel("Filter namespace: ").
		SetText("all").
		SetPlaceholder("all").
		SetFieldWidth(20)

	// Create tables
	podTable := tview.NewTable().SetBorders(true)
	nodeTable := tview.NewTable().SetBorders(true)

	// Node table header
	nodeHeaders := []string{"Node", "Memory (GB)", "Hardware", "Electricity", "Total"}
	for i, h := range nodeHeaders {
		c := tview.NewTableCell(h).SetAlign(tview.AlignCenter)
		nodeTable.SetCell(0, i, c)
	}

	// Node table data
	for i, node := range data.Nodes {
		row := i + 1
		nodeTable.SetCell(row, 0, tview.NewTableCell(node.Name).SetAlign(tview.AlignLeft))
		nodeTable.SetCell(row, 1, tview.NewTableCell(fmt.Sprintf("%.1f", node.MemoryGB)).SetAlign(tview.AlignRight))
		nodeTable.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("$%.2f", node.HardwareCost)).SetAlign(tview.AlignRight))
		nodeTable.SetCell(row, 3, tview.NewTableCell(fmt.Sprintf("$%.2f", node.ElecCost)).SetAlign(tview.AlignRight))
		nodeTable.SetCell(row, 4, tview.NewTableCell(fmt.Sprintf("$%.2f", node.TotalCost)).SetAlign(tview.AlignRight))
	}

	// Pod table header
	podHeaders := []string{"Pod", "Namespace", "CPU", "Memory", "Cost"}
	for i, h := range podHeaders {
		c := tview.NewTableCell(h).SetAlign(tview.AlignCenter)
		podTable.SetCell(0, i, c)
	}

	// Function to update pod table based on filter
	updatePodTable := func(ns string) {
		// Clear existing data rows
		for row := podTable.GetRowCount() - 1; row > 0; row-- {
			podTable.RemoveRow(row)
		}

		// Add filtered pods
		row := 1
		var totalCost float64
		for _, pod := range data.PodCosts {
			if ns == "all" || pod.Namespace == ns {
				podTable.SetCell(row, 0, tview.NewTableCell(pod.Name).SetAlign(tview.AlignLeft))
				podTable.SetCell(row, 1, tview.NewTableCell(pod.Namespace).SetAlign(tview.AlignLeft))
				podTable.SetCell(row, 2, tview.NewTableCell(pod.CPU).SetAlign(tview.AlignRight))
				podTable.SetCell(row, 3, tview.NewTableCell(pod.Memory).SetAlign(tview.AlignRight))
				podTable.SetCell(row, 4, tview.NewTableCell(fmt.Sprintf("$%.2f", pod.Cost)).SetAlign(tview.AlignRight))
				totalCost += pod.Cost
				row++
			}
		}

		// Add total row
		podTable.SetCell(row, 0, tview.NewTableCell("TOTAL").SetAlign(tview.AlignLeft))
		podTable.SetCell(row, 1, tview.NewTableCell("").SetAlign(tview.AlignLeft))
		podTable.SetCell(row, 2, tview.NewTableCell("").SetAlign(tview.AlignRight))
		podTable.SetCell(row, 3, tview.NewTableCell("").SetAlign(tview.AlignRight))
		podTable.SetCell(row, 4, tview.NewTableCell(fmt.Sprintf("$%.2f", totalCost)).SetAlign(tview.AlignRight))
	}

	updatePodTable("all")

	// Summary text
	summaryView := tview.NewTextView().SetWrap(true)
	updateSummary := func(ns string) {
		var podCount int
		var nsCost float64
		for _, pod := range data.PodCosts {
			if ns == "all" || pod.Namespace == ns {
				podCount++
				nsCost += pod.Cost
			}
		}

		summaryText := fmt.Sprintf(`
kfin Cost Analysis
==================

Total Monthly Cost: $%.2f
  Hardware (amortized): $%.2f
  Electricity:         $%.2f

Filtered Cost: $%.2f (%d pods)
Nodes: %d

Available namespaces: %v

Use arrow keys to navigate. Press ESC or Ctrl+C to exit.
`, data.TotalCost, data.HardwareCost, data.ElecCost, nsCost, podCount, len(data.Nodes), namespaces)
		summaryView.SetText(summaryText)
	}
	updateSummary("all")

	// Layout
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.AddItem(tview.NewTextView().SetText("kfin - Kubernetes Cost Analyzer"), 1, 0, false)
	flex.AddItem(inputField, 1, 0, true)
	flex.AddItem(summaryView, 0, 3, false)
	flex.AddItem(tview.NewTextView().SetText("Node Costs"), 1, 0, false)
	flex.AddItem(nodeTable, 0, 4, false)
	flex.AddItem(tview.NewTextView().SetText("Pod Costs (filtered)"), 1, 0, false)
	flex.AddItem(podTable, 0, 10, false)

	// Update on input change
	inputField.SetChangedFunc(func(text string) {
		if text == "" {
			text = "all"
		}
		updatePodTable(text)
		updateSummary(text)
		app.Draw()
	})

	if err := app.SetRoot(flex, true).Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
	}
}

func getNamespaces(pods []PodInfo) []string {
	seen := make(map[string]bool)
	var namespaces []string
	for _, pod := range pods {
		if !seen[pod.Namespace] {
			seen[pod.Namespace] = true
			namespaces = append(namespaces, pod.Namespace)
		}
	}
	return namespaces
}

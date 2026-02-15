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

	// Create tables
	podTable := tview.NewTable().SetBorders(true)
	nodeTable := tview.NewTable().SetBorders(true)

	// Pod table header
	headers := []string{"Pod", "Namespace", "CPU", "Memory", "Cost"}
	for i, h := range headers {
		c := tview.NewTableCell(h).SetAlign(tview.AlignCenter)
		podTable.SetCell(0, i, c)
	}

	// Pod table data
	for i, pod := range data.PodCosts {
		if pod.Cost > 0 {
			row := i + 1
			podTable.SetCell(row, 0, tview.NewTableCell(pod.Name).SetAlign(tview.AlignLeft))
			podTable.SetCell(row, 1, tview.NewTableCell(pod.Namespace).SetAlign(tview.AlignLeft))
			podTable.SetCell(row, 2, tview.NewTableCell(pod.CPU).SetAlign(tview.AlignRight))
			podTable.SetCell(row, 3, tview.NewTableCell(pod.Memory).SetAlign(tview.AlignRight))
			podTable.SetCell(row, 4, tview.NewTableCell(fmt.Sprintf("$%.2f", pod.Cost)).SetAlign(tview.AlignRight))
		}
	}

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

	// Summary text
	summaryText := fmt.Sprintf(`
kfin Cost Analysis
==================

Total Monthly Cost: $%.2f
  Hardware (amortized): $%.2f
  Electricity:         $%.2f

Pods with resource requests: %d
Nodes: %d

Use arrow keys to navigate. Press ESC or Ctrl+C to exit.
`, data.TotalCost, data.HardwareCost, data.ElecCost, len(data.PodCosts), len(data.Nodes))

	summaryView := tview.NewTextView().
		SetWrap(true)
	summaryView.SetText(summaryText)

	// Layout - use Flex
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.AddItem(summaryView, 0, 1, false)

	// Node section header
	nodeHeader := tview.NewTextView().SetText("\nNode Costs")
	flex.AddItem(nodeHeader, 1, 0, false)
	flex.AddItem(nodeTable, 0, 4, false)

	// Pod section header
	podHeader := tview.NewTextView().SetText("\nPod Costs")
	flex.AddItem(podHeader, 1, 0, false)
	flex.AddItem(podTable, 0, 10, false)

	if err := app.SetRoot(flex, true).Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
	}
}

package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
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

const (
	pageOverview    = "1"
	pagePods       = "2"
	pageNodes      = "3"
	pageNamespaces = "4"
)

func ShowDashboard(data ReportData) {
	app := tview.NewApplication()
	pages := tview.NewPages()

	// Build namespace filter options
	namespaces := getNamespaces(data.PodCosts)
	namespaces = append([]string{"all"}, namespaces...)

	// ========== MENU BAR (always visible) ==========
	menuBar := tview.NewTextView().
		SetText("[orange::b]kfin[/] | [::b]1[white]:Overview [::b]2[white]:Pods [::b]3[white]:Nodes [::b]4[white]:ByNamespace [::b]ESC[white]:Quit").
		SetDynamicColors(true)
	menuBar.SetBorder(false).SetBackgroundColor(tcell.ColorDarkBlue)

	// ========== OVERVIEW VIEW ==========
	overview := tview.NewFlex().SetDirection(tview.FlexRow)

	overviewText := fmt.Sprintf(`
kfin - Kubernetes Cost Analyzer

Monthly Cost: $%.2f
----------------------------------------------------------------
Hardware (amortized):  $%.2f
Electricity:           $%.2f

Total Pods: %d
Nodes: %d
Namespaces: %v
`, data.TotalCost, data.HardwareCost, data.ElecCost, len(data.PodCosts), len(data.Nodes), namespaces)

	overview.AddItem(tview.NewTextView().SetText(overviewText).SetDynamicColors(true), 0, 1, false)

	// ========== PODS VIEW ==========
	podsView := tview.NewFlex().SetDirection(tview.FlexRow)

	podTable := tview.NewTable().SetBorders(true)
	podHeaders := []string{"Namespace", "Pod", "CPU", "Memory", "Cost"}
	for i, h := range podHeaders {
		c := tview.NewTableCell(h).SetAlign(tview.AlignCenter)
		podTable.SetCell(0, i, c)
	}

	row := 1
	var totalCost float64
	for _, pod := range data.PodCosts {
		podTable.SetCell(row, 0, tview.NewTableCell(pod.Namespace).SetAlign(tview.AlignLeft))
		podTable.SetCell(row, 1, tview.NewTableCell(pod.Name).SetAlign(tview.AlignLeft))
		podTable.SetCell(row, 2, tview.NewTableCell(pod.CPU).SetAlign(tview.AlignRight))
		podTable.SetCell(row, 3, tview.NewTableCell(pod.Memory).SetAlign(tview.AlignRight))
		podTable.SetCell(row, 4, tview.NewTableCell(fmt.Sprintf("$%.2f", pod.Cost)).SetAlign(tview.AlignRight))
		totalCost += pod.Cost
		row++
	}

	// Total row
	podTable.SetCell(row, 0, tview.NewTableCell("TOTAL").SetAlign(tview.AlignLeft))
	podTable.SetCell(row, 1, tview.NewTableCell("").SetAlign(tview.AlignLeft))
	podTable.SetCell(row, 2, tview.NewTableCell("").SetAlign(tview.AlignRight))
	podTable.SetCell(row, 3, tview.NewTableCell("").SetAlign(tview.AlignRight))
	podTable.SetCell(row, 4, tview.NewTableCell(fmt.Sprintf("$%.2f", totalCost)).SetAlign(tview.AlignRight))

	podsView.AddItem(tview.NewTextView().SetText(fmt.Sprintf("\n Total Monthly Pod Cost: $%.2f\n", totalCost)), 1, 0, false)
	podsView.AddItem(podTable, 0, 1, false)

	// ========== NODES VIEW ==========
	nodesView := tview.NewFlex().SetDirection(tview.FlexRow)

	nodeTable := tview.NewTable().SetBorders(true)
	nodeHeaders := []string{"Node", "Memory (GB)", "Hardware", "Electricity", "Total"}
	for i, h := range nodeHeaders {
		c := tview.NewTableCell(h).SetAlign(tview.AlignCenter)
		nodeTable.SetCell(0, i, c)
	}

	var nodeTotal float64
	for i, node := range data.Nodes {
		row := i + 1
		nodeTable.SetCell(row, 0, tview.NewTableCell(node.Name).SetAlign(tview.AlignLeft))
		nodeTable.SetCell(row, 1, tview.NewTableCell(fmt.Sprintf("%.1f", node.MemoryGB)).SetAlign(tview.AlignRight))
		nodeTable.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("$%.2f", node.HardwareCost)).SetAlign(tview.AlignRight))
		nodeTable.SetCell(row, 3, tview.NewTableCell(fmt.Sprintf("$%.2f", node.ElecCost)).SetAlign(tview.AlignRight))
		nodeTable.SetCell(row, 4, tview.NewTableCell(fmt.Sprintf("$%.2f", node.TotalCost)).SetAlign(tview.AlignRight))
		nodeTotal += node.TotalCost
	}

	// Total row
	nodeTable.SetCell(len(data.Nodes)+1, 0, tview.NewTableCell("TOTAL").SetAlign(tview.AlignLeft))
	nodeTable.SetCell(len(data.Nodes)+1, 1, tview.NewTableCell("").SetAlign(tview.AlignRight))
	nodeTable.SetCell(len(data.Nodes)+1, 2, tview.NewTableCell("").SetAlign(tview.AlignRight))
	nodeTable.SetCell(len(data.Nodes)+1, 3, tview.NewTableCell("").SetAlign(tview.AlignRight))
	nodeTable.SetCell(len(data.Nodes)+1, 4, tview.NewTableCell(fmt.Sprintf("$%.2f", nodeTotal)).SetAlign(tview.AlignRight))

	nodesView.AddItem(tview.NewTextView().SetText(fmt.Sprintf("\n Total Monthly Node Cost: $%.2f\n", nodeTotal)), 1, 0, false)
	nodesView.AddItem(nodeTable, 0, 1, false)

	// ========== BY NAMESPACE VIEW ==========
	nsView := tview.NewFlex().SetDirection(tview.FlexRow)

	nsTable := tview.NewTable().SetBorders(true)
	nsHeaders := []string{"Namespace", "Pods", "Monthly Cost"}
	for i, h := range nsHeaders {
		c := tview.NewTableCell(h).SetAlign(tview.AlignCenter)
		nsTable.SetCell(0, i, c)
	}

	// Calculate costs by namespace
	type nsCostInfo struct {
		count int
		cost  float64
	}
	nsInfo := make(map[string]nsCostInfo)

	for _, pod := range data.PodCosts {
		info := nsInfo[pod.Namespace]
		info.count++
		info.cost += pod.Cost
		nsInfo[pod.Namespace] = info
	}

	row = 1
	var grandTotal float64
	for _, ns := range namespaces {
		if ns == "all" {
			continue
		}
		if info, ok := nsInfo[ns]; ok {
			nsTable.SetCell(row, 0, tview.NewTableCell(ns).SetAlign(tview.AlignLeft))
			nsTable.SetCell(row, 1, tview.NewTableCell(fmt.Sprintf("%d", info.count)).SetAlign(tview.AlignRight))
			nsTable.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("$%.2f", info.cost)).SetAlign(tview.AlignRight))
			grandTotal += info.cost
			row++
		}
	}

	// Total row
	nsTable.SetCell(row, 0, tview.NewTableCell("TOTAL").SetAlign(tview.AlignLeft))
	nsTable.SetCell(row, 1, tview.NewTableCell("").SetAlign(tview.AlignRight))
	nsTable.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("$%.2f", grandTotal)).SetAlign(tview.AlignRight))

	nsView.AddItem(tview.NewTextView().SetText(fmt.Sprintf("\n Total Monthly Cost: $%.2f\n", grandTotal)), 1, 0, false)
	nsView.AddItem(nsTable, 0, 1, false)

	// Add pages
	pages.AddPage(pageOverview, overview, true, true)
	pages.AddPage(pagePods, podsView, true, false)
	pages.AddPage(pageNodes, nodesView, true, false)
	pages.AddPage(pageNamespaces, nsView, true, false)

	// Main layout with menu bar on top
	mainLayout := tview.NewFlex().SetDirection(tview.FlexRow)
	mainLayout.AddItem(menuBar, 1, 0, false)
	mainLayout.AddItem(pages, 0, 1, true)

	// Key handler
	mainLayout.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			app.Stop()
		case tcell.KeyF1:
			pages.SwitchToPage(pageOverview)
		case tcell.KeyF2:
			pages.SwitchToPage(pagePods)
		case tcell.KeyF3:
			pages.SwitchToPage(pageNodes)
		case tcell.KeyF4:
			pages.SwitchToPage(pageNamespaces)
		}
		// Also handle number keys
		switch string(event.Rune()) {
		case "1":
			pages.SwitchToPage(pageOverview)
		case "2":
			pages.SwitchToPage(pagePods)
		case "3":
			pages.SwitchToPage(pageNodes)
		case "4":
			pages.SwitchToPage(pageNamespaces)
		}
		return event
	})

	if err := app.SetRoot(mainLayout, true).Run(); err != nil {
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

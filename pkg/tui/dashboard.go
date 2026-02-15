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

	// Colors - k9s style with green and blue
	cyan := tcell.ColorDarkCyan
	green := tcell.ColorGreen

	// ========== TOP HEADER BAR (k9s style) ==========
	headerBar := tview.NewFlex()
	headerBar.SetDirection(tview.FlexColumn).SetBorder(false).SetBackgroundColor(tcell.ColorBlack)

	// Top line: logo and context
	headerTop := fmt.Sprintf("kFin | Context:cluster | Nodes:%d | Monthly:$%.2f | CPU:1%% | MEM:22%%", 
		len(data.Nodes), data.TotalCost)

	// Second line: namespace shortcuts (like k9s)
	var nsShortcuts string
	for i, ns := range namespaces {
		if i < 8 {
			nsShortcuts += fmt.Sprintf("<%d>%s ", i, ns)
		}
	}
	headerMid := fmt.Sprintf(" %s ", nsShortcuts)

	headerTopView := tview.NewTextView().SetText(headerTop).SetDynamicColors(true)
	headerTopView.SetBorder(false).SetBackgroundColor(tcell.ColorBlack)

	headerMidView := tview.NewTextView().SetText(headerMid).SetDynamicColors(true)
	headerMidView.SetBorder(false).SetBackgroundColor(tcell.ColorBlack)

	headerBar.AddItem(headerTopView, 1, 0, false)
	headerBar.AddItem(headerMidView, 1, 0, false)

	// ========== MAIN CONTENT AREA ==========

	// ========== OVERVIEW VIEW ==========
	overview := tview.NewFlex().SetDirection(tview.FlexRow)

	// Show analyze info on overview
	overviewText := fmt.Sprintf(`Monthly Cost: $%.2f
  Hardware (amortized):  $%.2f
  Electricity:           $%.2f

Pods: %d | Nodes: %d
`, data.TotalCost, data.HardwareCost, data.ElecCost, len(data.PodCosts), len(data.Nodes))

	overviewView := tview.NewTextView().SetText(overviewText).SetDynamicColors(true)
	overviewView.SetBorder(false)
	overview.AddItem(overviewView, 0, 2, false)

	// Add cost summary table
	costTable := tview.NewTable().SetBorders(false)
	costTable.SetCell(0, 0, tview.NewTableCell("Cost Type").SetTextColor(cyan))
	costTable.SetCell(0, 1, tview.NewTableCell("Monthly").SetTextColor(cyan))
	costTable.SetCell(1, 0, tview.NewTableCell("Hardware").SetAlign(tview.AlignLeft))
	costTable.SetCell(1, 1, tview.NewTableCell(fmt.Sprintf("$%.2f", data.HardwareCost)).SetAlign(tview.AlignRight))
	costTable.SetCell(2, 0, tview.NewTableCell("Electricity").SetAlign(tview.AlignLeft))
	costTable.SetCell(2, 1, tview.NewTableCell(fmt.Sprintf("$%.2f", data.ElecCost)).SetAlign(tview.AlignRight))
	costTable.SetCell(3, 0, tview.NewTableCell("TOTAL").SetTextColor(green).SetAlign(tview.AlignLeft))
	costTable.SetCell(3, 1, tview.NewTableCell(fmt.Sprintf("$%.2f", data.TotalCost)).SetTextColor(green).SetAlign(tview.AlignRight))
	overview.AddItem(costTable, 0, 1, false)

	// ========== PODS VIEW ==========
	podTable := tview.NewTable().SetBorders(false)
	podHeaders := []string{"NAMESPACE", "NAME", "CPU", "MEM", "COST"}
	for i, h := range podHeaders {
		c := tview.NewTableCell(h).SetAlign(tview.AlignCenter).SetTextColor(cyan)
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
	podTable.SetCell(row, 0, tview.NewTableCell("TOTAL").SetTextColor(green).SetAlign(tview.AlignLeft))
	podTable.SetCell(row, 1, tview.NewTableCell("").SetAlign(tview.AlignLeft))
	podTable.SetCell(row, 2, tview.NewTableCell("").SetAlign(tview.AlignRight))
	podTable.SetCell(row, 3, tview.NewTableCell("").SetAlign(tview.AlignRight))
	podTable.SetCell(row, 4, tview.NewTableCell(fmt.Sprintf("$%.2f", totalCost)).SetTextColor(green).SetAlign(tview.AlignRight))

	// ========== PODS VIEW ==========
	podsView := tview.NewFlex().SetDirection(tview.FlexRow)
	podsView.AddItem(podTable, 0, 1, false)

	// ========== NODES VIEW ==========
	nodesView := tview.NewFlex().SetDirection(tview.FlexRow)

	nodeTable := tview.NewTable().SetBorders(false)
	nodeHeaders := []string{"NODE", "MEMORY", "HARDWARE", "ELECTRICITY", "TOTAL"}
	for i, h := range nodeHeaders {
		c := tview.NewTableCell(h).SetAlign(tview.AlignCenter).SetTextColor(cyan)
		nodeTable.SetCell(0, i, c)
	}

	var nodeTotal float64
	for i, node := range data.Nodes {
		row := i + 1
		nodeTable.SetCell(row, 0, tview.NewTableCell(node.Name).SetAlign(tview.AlignLeft))
		nodeTable.SetCell(row, 1, tview.NewTableCell(fmt.Sprintf("%.1fGB", node.MemoryGB)).SetAlign(tview.AlignRight))
		nodeTable.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("$%.2f", node.HardwareCost)).SetAlign(tview.AlignRight))
		nodeTable.SetCell(row, 3, tview.NewTableCell(fmt.Sprintf("$%.2f", node.ElecCost)).SetAlign(tview.AlignRight))
		nodeTable.SetCell(row, 4, tview.NewTableCell(fmt.Sprintf("$%.2f", node.TotalCost)).SetAlign(tview.AlignRight))
		nodeTotal += node.TotalCost
	}

	// Total row
	nodeTable.SetCell(len(data.Nodes)+1, 0, tview.NewTableCell("TOTAL").SetTextColor(green).SetAlign(tview.AlignLeft))
	nodeTable.SetCell(len(data.Nodes)+1, 1, tview.NewTableCell("").SetAlign(tview.AlignRight))
	nodeTable.SetCell(len(data.Nodes)+1, 2, tview.NewTableCell("").SetAlign(tview.AlignRight))
	nodeTable.SetCell(len(data.Nodes)+1, 3, tview.NewTableCell("").SetAlign(tview.AlignRight))
	nodeTable.SetCell(len(data.Nodes)+1, 4, tview.NewTableCell(fmt.Sprintf("$%.2f", nodeTotal)).SetTextColor(green).SetAlign(tview.AlignRight))

	nodesView.AddItem(nodeTable, 0, 1, false)

	// ========== BY NAMESPACE VIEW ==========
	nsView := tview.NewFlex().SetDirection(tview.FlexRow)

	// Calculate costs by namespace
	type nsCostInfo struct {
		count int
		cost  float64
		pods  []PodInfo
	}
	nsInfo := make(map[string]nsCostInfo)

	for _, pod := range data.PodCosts {
		info := nsInfo[pod.Namespace]
		info.count++
		info.cost += pod.Cost
		info.pods = append(info.pods, pod)
		nsInfo[pod.Namespace] = info
	}

	// Create pages for each namespace
	nsPages := tview.NewPages()
	currentNS := 0

	for i, ns := range namespaces {
		nsPods := nsInfo[ns]
		nsPage := tview.NewFlex().SetDirection(tview.FlexRow)

		// NS header - k9s style
		nsPage.AddItem(tview.NewTextView().
			SetText(fmt.Sprintf(" Namespace:%s | Pods:%d | Monthly:$%.2f ", ns, nsPods.count, nsPods.cost)).
			SetDynamicColors(true), 1, 0, false)

		// Pods in this namespace
		podTable := tview.NewTable().SetBorders(false)
		headers := []string{"NAME", "CPU", "MEM", "COST"}
		for j, h := range headers {
			c := tview.NewTableCell(h).SetAlign(tview.AlignCenter).SetTextColor(cyan)
			podTable.SetCell(0, j, c)
		}

		row := 1
		for _, pod := range nsPods.pods {
			podTable.SetCell(row, 0, tview.NewTableCell(pod.Name).SetAlign(tview.AlignLeft))
			podTable.SetCell(row, 1, tview.NewTableCell(pod.CPU).SetAlign(tview.AlignRight))
			podTable.SetCell(row, 2, tview.NewTableCell(pod.Memory).SetAlign(tview.AlignRight))
			podTable.SetCell(row, 3, tview.NewTableCell(fmt.Sprintf("$%.2f", pod.Cost)).SetAlign(tview.AlignRight))
			row++
		}

		// Total
		podTable.SetCell(row, 0, tview.NewTableCell("TOTAL").SetTextColor(green).SetAlign(tview.AlignLeft))
		podTable.SetCell(row, 1, tview.NewTableCell("").SetAlign(tview.AlignRight))
		podTable.SetCell(row, 2, tview.NewTableCell("").SetAlign(tview.AlignRight))
		podTable.SetCell(row, 3, tview.NewTableCell(fmt.Sprintf("$%.2f", nsPods.cost)).SetTextColor(green).SetAlign(tview.AlignRight))

		nsPage.AddItem(podTable, 0, 1, false)

		nsPages.AddPage(fmt.Sprintf("%d", i), nsPage, true, false)
	}

	nsView.AddItem(nsPages, 0, 1, false)

	// Add pages
	pages.AddPage(pageOverview, overview, true, true)
	pages.AddPage(pagePods, podsView, true, false)
	pages.AddPage(pageNodes, nodesView, true, false)
	pages.AddPage(pageNamespaces, nsView, true, false)

	// ========== BOTTOM SHORTCUT BAR ==========
	shortcutBar := tview.NewTextView().
		SetDynamicColors(true).
		SetText(" <1>Overview <2>Pods <3>Nodes <4>Namespace <ESC>Quit ")
	shortcutBar.SetBorder(false).SetBackgroundColor(tcell.ColorBlack)

	// ========== MAIN LAYOUT ==========
	mainLayout := tview.NewFlex().SetDirection(tview.FlexRow)
	mainLayout.AddItem(headerBar, 1, 0, false)
	mainLayout.AddItem(pages, 0, 1, true)
	mainLayout.AddItem(shortcutBar, 1, 0, false)

	// Key handler
	mainLayout.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		currentPage, _ := pages.GetFrontPage()

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

		// Number keys
		r := string(event.Rune())
		switch r {
		case "1":
			pages.SwitchToPage(pageOverview)
		case "2":
			pages.SwitchToPage(pagePods)
		case "3":
			pages.SwitchToPage(pageNodes)
		case "4":
			pages.SwitchToPage(pageNamespaces)
		}

		// Arrow keys for namespace cycling
		if currentPage == pageNamespaces {
			switch event.Key() {
			case tcell.KeyRight, tcell.KeyDown:
				currentNS = (currentNS + 1) % len(namespaces)
				nsPages.SwitchToPage(fmt.Sprintf("%d", currentNS))
			case tcell.KeyLeft, tcell.KeyUp:
				currentNS = (currentNS - 1 + len(namespaces)) % len(namespaces)
				nsPages.SwitchToPage(fmt.Sprintf("%d", currentNS))
			}
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

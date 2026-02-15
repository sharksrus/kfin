package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

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
	PodCosts         []PodInfo
	TotalCost        float64
	HardwareCost     float64
	ElecCost         float64
	ControlPlaneCost float64
	Nodes            []NodeInfo
	ContextName      string
	ClusterName      string
	PricingSource    string
	StatsFreshness   StatsFreshness
}

type StatsFreshness struct {
	Ready            bool
	BaseURL          string
	LookbackDuration time.Duration
	ObservedDuration time.Duration
	SampleCount      int
	LastSampleAt     time.Time
	Note             string
}

const (
	pageOverview   = "1"
	pageNamespaces = "2"
	pageNodes      = "3"
)

func ShowDashboard(data ReportData) {
	app := tview.NewApplication()
	pages := tview.NewPages()
	namespaces := getNamespaces(data.PodCosts)
	// Sort namespaces alphabetically
	sort.Strings(namespaces)
	nsInfo := buildNamespaceInfo(data.PodCosts)
	nsIndex := make(map[string]int, len(namespaces))
	for i, ns := range namespaces {
		nsIndex[ns] = i
	}

	cyan := tcell.ColorDarkCyan

	// ========== HEADER ==========
	headerBar := tview.NewFlex()
	headerBar.SetDirection(tview.FlexRow).SetBorder(false).SetBackgroundColor(tcell.ColorBlack)

	headerTop := fmt.Sprintf(
		"kFin | Context: %s | Cluster: %s | Nodes:%d | Monthly:$%.2f | Rates:%s",
		truncateString(data.ContextName, 28),
		truncateString(data.ClusterName, 28),
		len(data.Nodes),
		data.TotalCost,
		truncateString(data.PricingSource, 12),
	)
	headerMid := " [1] Overview  [2] Namespaces  [3] Nodes "

	logoView := tview.NewTextView().
		SetText(buildASCIIKFinLogo()).
		SetDynamicColors(true)
	logoView.SetBorder(false).SetBackgroundColor(tcell.ColorBlack)

	headerTopView := tview.NewTextView().SetText(" " + headerTop).SetDynamicColors(true)
	headerTopView.SetBorder(false).SetBackgroundColor(tcell.ColorBlack)

	headerMidView := tview.NewTextView().SetText(" " + headerMid).SetDynamicColors(true)
	headerMidView.SetBorder(false).SetBackgroundColor(tcell.ColorBlack)

	headerBar.AddItem(logoView, 5, 0, false)
	headerBar.AddItem(headerTopView, 1, 0, false)
	headerBar.AddItem(headerMidView, 1, 0, false)

	// ========== OVERVIEW VIEW ==========
	overview := tview.NewFlex().SetDirection(tview.FlexRow)
	snapshot := tview.NewTextView().SetDynamicColors(true)
	snapshot.SetBorder(true).SetTitle(" Cluster Snapshot ").SetTitleColor(cyan)
	tierLabel, tierColor := costTier(data.TotalCost)
	snapshot.SetText(fmt.Sprintf(
		" Pods:        %d\n Nodes:       %d\n Namespaces:  %d\n Monthly:     $%.2f\n Daily:       $%.2f\n Cost Tier:   [%s]%s[-]",
		len(data.PodCosts),
		len(data.Nodes),
		len(namespaces),
		data.TotalCost,
		data.TotalCost/30.0,
		tierColor,
		tierLabel,
	))

	var hardwarePct, elecPct, controlPlanePct float64
	if data.TotalCost > 0 {
		hardwarePct = (data.HardwareCost / data.TotalCost) * 100.0
		elecPct = (data.ElecCost / data.TotalCost) * 100.0
		controlPlanePct = (data.ControlPlaneCost / data.TotalCost) * 100.0
	}
	costBreakdown := tview.NewTextView().SetDynamicColors(true)
	costBreakdown.SetBorder(true).SetTitle(" Cost Breakdown ").SetTitleColor(cyan)
	costBreakdown.SetText(fmt.Sprintf(
		" Hardware:      $%.2f (%.1f%%)\n Electricity:   $%.2f (%.1f%%)\n Control Plane: $%.2f (%.1f%%)\n Allocation:\n [green]H[-] %s\n [yellow]E[-] %s\n [blue]C[-] %s",
		data.HardwareCost, hardwarePct,
		data.ElecCost, elecPct,
		data.ControlPlaneCost, controlPlanePct,
		renderCostBar(hardwarePct),
		renderCostBar(elecPct),
		renderCostBar(controlPlanePct),
	))

	topPods := tview.NewTable().SetBorders(false)
	topPods.SetSelectable(true, false)
	topPods.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorDarkCyan).Foreground(tcell.ColorBlack))
	topPods.SetBorder(true).SetTitle(" Top Pods By Monthly Cost ").SetTitleColor(cyan)
	topPodsHeaders := []string{"POD", "NS", "COST"}
	for i, h := range topPodsHeaders {
		topPods.SetCell(0, i, tview.NewTableCell(h).SetTextColor(cyan).SetAlign(tview.AlignLeft))
	}
	topPodItems := topPodsByCost(data.PodCosts, 8)
	for i, pod := range topPodItems {
		topPods.SetCell(i+1, 0, tview.NewTableCell(truncateString(pod.Name, 28)).SetAlign(tview.AlignLeft))
		topPods.SetCell(i+1, 1, tview.NewTableCell(truncateString(pod.Namespace, 16)).SetAlign(tview.AlignLeft))
		topPods.SetCell(i+1, 2, tview.NewTableCell(fmt.Sprintf("%s $%.2f", costBadge(pod.Cost), pod.Cost)).SetAlign(tview.AlignRight))
	}
	if len(topPodItems) > 0 {
		topPods.Select(1, 0)
	}

	topNS := tview.NewTable().SetBorders(false)
	topNS.SetSelectable(true, false)
	topNS.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorDarkCyan).Foreground(tcell.ColorBlack))
	topNS.SetBorder(true).SetTitle(" Top Namespaces By Monthly Cost ").SetTitleColor(cyan)
	topNSHeaders := []string{"NAMESPACE", "PODS", "COST"}
	for i, h := range topNSHeaders {
		topNS.SetCell(0, i, tview.NewTableCell(h).SetTextColor(cyan).SetAlign(tview.AlignLeft))
	}
	topNSItems := topNamespacesByCost(nsInfo, 8)
	for i, ns := range topNSItems {
		topNS.SetCell(i+1, 0, tview.NewTableCell(truncateString(ns.name, 28)).SetAlign(tview.AlignLeft))
		topNS.SetCell(i+1, 1, tview.NewTableCell(fmt.Sprintf("%d", ns.count)).SetAlign(tview.AlignRight))
		topNS.SetCell(i+1, 2, tview.NewTableCell(fmt.Sprintf("%s $%.2f", costBadge(ns.cost), ns.cost)).SetAlign(tview.AlignRight))
	}
	if len(topNSItems) > 0 {
		topNS.Select(1, 0)
	}

	topRow := tview.NewFlex().SetDirection(tview.FlexColumn)
	topRow.AddItem(snapshot, 0, 1, false)
	topRow.AddItem(costBreakdown, 0, 1, false)

	bottomRow := tview.NewFlex().SetDirection(tview.FlexColumn)
	bottomRow.AddItem(topPods, 0, 1, false)
	bottomRow.AddItem(topNS, 0, 1, false)

	overview.AddItem(topRow, 8, 0, false)
	overview.AddItem(bottomRow, 0, 1, false)
	activeOverviewTable := 0
	updateOverviewFocus := func() {
		if activeOverviewTable == 0 {
			topPods.SetTitle(" Top Pods By Monthly Cost * ").SetBorderColor(cyan)
			topNS.SetTitle(" Top Namespaces By Monthly Cost ").SetBorderColor(tcell.ColorGray)
			return
		}
		topPods.SetTitle(" Top Pods By Monthly Cost ").SetBorderColor(tcell.ColorGray)
		topNS.SetTitle(" Top Namespaces By Monthly Cost * ").SetBorderColor(cyan)
	}
	updateOverviewFocus()

	// ========== NODES VIEW ==========
	nodesView := tview.NewFlex().SetDirection(tview.FlexRow)
	nodesList := tview.NewTextView().
		SetDynamicColors(true).
		SetText(buildNodesListText(data.Nodes))
	nodesList.SetBorder(false)
	nodesView.AddItem(nodesList, 0, 1, false)

	// ========== BY NAMESPACE VIEW ==========
	nsView := tview.NewFlex().SetDirection(tview.FlexRow)

	nsPages := tview.NewPages()
	currentNS := 0

	for i, ns := range namespaces {
		nsPods := nsInfo[ns]
		nsPage := tview.NewFlex().SetDirection(tview.FlexRow)
		nsList := tview.NewTextView().
			SetDynamicColors(true).
			SetText(buildNamespaceListText(nsPods, true))
		nsList.SetBorder(false)
		nsPage.AddItem(nsList, 0, 1, false)
		nsPages.AddPage(fmt.Sprintf("%d", i), nsPage, true, false)
	}
	nsView.AddItem(nsPages, 0, 1, false)

	// Add pages
	pages.AddPage(pageOverview, overview, true, true)
	pages.AddPage(pageNamespaces, nsView, true, false)
	pages.AddPage(pageNodes, nodesView, true, false)

	updateHeaderNav := func() {
		currentPage, _ := pages.GetFrontPage()
		overviewLabel := "[1] Overview"
		nsLabel := "[2] Namespaces"
		nodesLabel := "[3] Nodes"
		switch currentPage {
		case pageOverview:
			overviewLabel = "[darkcyan][1] Overview[-]"
		case pageNamespaces:
			nsLabel = "[darkcyan][2] Namespaces[-]"
		case pageNodes:
			nodesLabel = "[darkcyan][3] Nodes[-]"
		}
		headerMidView.SetText(fmt.Sprintf(" %s  %s  %s ", overviewLabel, nsLabel, nodesLabel))
	}
	updateHeaderNav()

	// Show first namespace by default
	if len(namespaces) > 0 {
		nsPages.SwitchToPage("0")
	}

	pageTitleView := tview.NewTextView().SetDynamicColors(true)
	pageTitleView.SetBorder(false).SetBackgroundColor(tcell.ColorBlack)

	updatePageTitle := func() {
		currentPage, _ := pages.GetFrontPage()
		switch currentPage {
		case pageOverview:
			pageTitleView.SetText(" [darkcyan]OVERVIEW[-]  |  [gray]Tab/Left/Right switch tables, Up/Down move row, Enter pod details[-]")
		case pageNodes:
			pageTitleView.SetText(" [darkcyan]NODES[-]  |  [gray]Cluster monthly hardware + electricity by node[-]")
		case pageNamespaces:
			if len(namespaces) == 0 {
				pageTitleView.SetText(" [darkcyan]NAMESPACES[-]")
				return
			}
			ns := namespaces[currentNS]
			info := nsInfo[ns]
			pageTitleView.SetText(fmt.Sprintf(" [darkcyan]NAMESPACES[-]  >  [white]%s[-]  |  Pods:%d  Cost:$%.2f", ns, info.count, info.cost))
		}
	}
	updatePageTitle()

	contentFrame := tview.NewFlex().SetDirection(tview.FlexColumn)
	contentFrame.AddItem(tview.NewBox().SetBackgroundColor(tcell.ColorBlack), 2, 0, false)
	contentFrame.AddItem(pages, 0, 1, true)
	contentFrame.AddItem(tview.NewBox().SetBackgroundColor(tcell.ColorBlack), 1, 0, false)

	podDetailView := tview.NewTextView().SetDynamicColors(true)
	podDetailView.SetBorder(true).SetTitle(" Pod Details ").SetTitleColor(cyan)
	podDetailView.SetBackgroundColor(tcell.ColorBlack)
	podModalFrame := tview.NewFlex().SetDirection(tview.FlexRow)
	podModalFrame.AddItem(tview.NewBox(), 0, 1, false)
	podModalFrame.AddItem(
		tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(tview.NewBox(), 0, 1, false).
			AddItem(podDetailView, 80, 0, true).
			AddItem(tview.NewBox(), 0, 1, false),
		12, 0, true,
	)
	podModalFrame.AddItem(tview.NewBox(), 0, 1, false)
	const pagePodDetail = "pod-detail"
	pages.AddPage(pagePodDetail, podModalFrame, true, false)
	podModalVisible := false

	showPodDetail := func(pod PodInfo) {
		podDetailView.SetText(buildPodDetailText(pod, data.StatsFreshness))
		pages.ShowPage(pagePodDetail)
		podModalVisible = true
	}

	hidePodDetail := func() {
		pages.HidePage(pagePodDetail)
		podModalVisible = false
	}

	footerNavView := tview.NewTextView().SetDynamicColors(true)
	footerNavView.SetBorder(false).SetBackgroundColor(tcell.ColorBlack)
	updateFooterNav := func() {
		currentPage, _ := pages.GetFrontPage()
		overviewLabel := "[1] Overview"
		nsLabel := "[2] Namespaces"
		nodesLabel := "[3] Nodes"
		switch currentPage {
		case pageOverview:
			overviewLabel = "[darkcyan][1] Overview[-]"
		case pageNamespaces:
			nsLabel = "[darkcyan][2] Namespaces[-]"
		case pageNodes:
			nodesLabel = "[darkcyan][3] Nodes[-]"
		}
		footerNavView.SetText(fmt.Sprintf(" %s  %s  %s  |  Left/Right: Cycle NS  Esc: Back  : Command ", overviewLabel, nsLabel, nodesLabel))
	}
	updateFooterNav()

	mainLayout := tview.NewFlex().SetDirection(tview.FlexRow)
	mainLayout.AddItem(headerBar, 7, 0, false)
	mainLayout.AddItem(pageTitleView, 1, 0, false)
	mainLayout.AddItem(contentFrame, 0, 1, true)
	mainLayout.AddItem(footerNavView, 1, 0, false)

	pageHistory := []string{}
	commandMode := false
	commandBuffer := ""

	switchToPage := func(page string) {
		currentPage, _ := pages.GetFrontPage()
		if currentPage == page {
			return
		}
		pageHistory = append(pageHistory, currentPage)
		pages.SwitchToPage(page)
	}

	updateCommandFooter := func() {
		footerNavView.SetText(fmt.Sprintf(" [darkcyan]Command[-] %s[gray]   (Enter to run, Esc to cancel)[-]", commandBuffer))
	}

	mainLayout.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if podModalVisible {
			switch event.Key() {
			case tcell.KeyEsc, tcell.KeyEnter:
				hidePodDetail()
			}
			return nil
		}

		if commandMode {
			switch event.Key() {
			case tcell.KeyEsc:
				commandMode = false
				commandBuffer = ""
				updateFooterNav()
				return nil
			case tcell.KeyEnter:
				cmd := strings.TrimSpace(commandBuffer)
				commandMode = false
				commandBuffer = ""
				switch cmd {
				case ":q", ":quit":
					app.Stop()
					return nil
				case "":
					updateFooterNav()
					return nil
				default:
					footerNavView.SetText(fmt.Sprintf(" [yellow]Unknown command:[-] %s  [gray](supported: :q, :quit)[-]", cmd))
					return nil
				}
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				if len(commandBuffer) > 1 {
					commandBuffer = commandBuffer[:len(commandBuffer)-1]
				}
				updateCommandFooter()
				return nil
			}

			if event.Rune() != 0 {
				commandBuffer += string(event.Rune())
				updateCommandFooter()
				return nil
			}
			return nil
		}

		switch event.Key() {
		case tcell.KeyEsc:
			if len(pageHistory) > 0 {
				prev := pageHistory[len(pageHistory)-1]
				pageHistory = pageHistory[:len(pageHistory)-1]
				pages.SwitchToPage(prev)
			}
			updateHeaderNav()
			updateFooterNav()
			updatePageTitle()
			return nil
		}
		r := string(event.Rune())
		switch r {
		case "1":
			switchToPage(pageOverview)
		case "2":
			switchToPage(pageNamespaces)
		case "3":
			switchToPage(pageNodes)
		case ":":
			commandMode = true
			commandBuffer = ":"
			updateCommandFooter()
			return nil
		}
		currentPage, _ := pages.GetFrontPage()
		if currentPage == pageOverview {
			switch event.Key() {
			case tcell.KeyTAB, tcell.KeyRight, tcell.KeyLeft:
				if activeOverviewTable == 0 {
					activeOverviewTable = 1
				} else {
					activeOverviewTable = 0
				}
				updateOverviewFocus()
			case tcell.KeyDown:
				if activeOverviewTable == 0 {
					r, _ := topPods.GetSelection()
					if len(topPodItems) > 0 {
						next := r + 1
						if next > len(topPodItems) {
							next = 1
						}
						topPods.Select(next, 0)
					}
				} else {
					r, _ := topNS.GetSelection()
					if len(topNSItems) > 0 {
						next := r + 1
						if next > len(topNSItems) {
							next = 1
						}
						topNS.Select(next, 0)
					}
				}
			case tcell.KeyUp:
				if activeOverviewTable == 0 {
					r, _ := topPods.GetSelection()
					if len(topPodItems) > 0 {
						prev := r - 1
						if prev < 1 {
							prev = len(topPodItems)
						}
						topPods.Select(prev, 0)
					}
				} else {
					r, _ := topNS.GetSelection()
					if len(topNSItems) > 0 {
						prev := r - 1
						if prev < 1 {
							prev = len(topNSItems)
						}
						topNS.Select(prev, 0)
					}
				}
			case tcell.KeyEnter:
				if len(namespaces) == 0 {
					break
				}
				if activeOverviewTable == 0 {
					r, _ := topPods.GetSelection()
					if len(topPodItems) > 0 && r >= 1 && r <= len(topPodItems) {
						showPodDetail(topPodItems[r-1])
					}
				} else {
					r, _ := topNS.GetSelection()
					if len(topNSItems) > 0 && r >= 1 && r <= len(topNSItems) {
						ns := topNSItems[r-1].name
						if idx, ok := nsIndex[ns]; ok {
							currentNS = idx
							nsPages.SwitchToPage(fmt.Sprintf("%d", currentNS))
							switchToPage(pageNamespaces)
						}
					}
				}
			}
		}
		if currentPage == pageNamespaces {
			if len(namespaces) == 0 {
				return event
			}
			switch event.Key() {
			case tcell.KeyRight:
				currentNS = (currentNS + 1) % len(namespaces)
				nsPages.SwitchToPage(fmt.Sprintf("%d", currentNS))
			case tcell.KeyLeft:
				currentNS = (currentNS - 1 + len(namespaces)) % len(namespaces)
				nsPages.SwitchToPage(fmt.Sprintf("%d", currentNS))
			}
		}
		updateHeaderNav()
		updateFooterNav()
		updatePageTitle()
		return event
	})

	if err := app.SetRoot(mainLayout, true).Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
	}
}

func buildASCIIKFinLogo() string {
	return `[green] _    _____ ___ _   _
| | _|  ___|_ _| \ | |
| |/ / |_   | ||  \| |
|   <|  _|  | || . ` + "`" + ` |
|_|\_\_|   |___|_|\__|[-]`
}

type nsCostInfo struct {
	count int
	cost  float64
	pods  []PodInfo
}

type nsSummary struct {
	name  string
	count int
	cost  float64
}

func buildNamespaceInfo(pods []PodInfo) map[string]nsCostInfo {
	nsInfo := make(map[string]nsCostInfo)
	for _, pod := range pods {
		info := nsInfo[pod.Namespace]
		info.count++
		info.cost += pod.Cost
		info.pods = append(info.pods, pod)
		nsInfo[pod.Namespace] = info
	}
	return nsInfo
}

func topPodsByCost(pods []PodInfo, n int) []PodInfo {
	var filtered []PodInfo
	for _, pod := range pods {
		if pod.Cost > 0 {
			filtered = append(filtered, pod)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Cost > filtered[j].Cost
	})
	if len(filtered) > n {
		return filtered[:n]
	}
	return filtered
}

func topNamespacesByCost(nsInfo map[string]nsCostInfo, n int) []nsSummary {
	summaries := make([]nsSummary, 0, len(nsInfo))
	for name, info := range nsInfo {
		summaries = append(summaries, nsSummary{name: name, count: info.count, cost: info.cost})
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].cost > summaries[j].cost
	})
	if len(summaries) > n {
		return summaries[:n]
	}
	return summaries
}

func renderCostBar(percent float64) string {
	const width = 18
	filled := int((percent / 100.0) * width)
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "|"
		} else {
			bar += "."
		}
	}
	return bar
}

func costTier(totalCost float64) (string, string) {
	switch {
	case totalCost >= 3000:
		return "LARGE", "red"
	case totalCost >= 1000:
		return "MEDIUM", "yellow"
	default:
		return "SMALL", "green"
	}
}

func costBadge(cost float64) string {
	switch {
	case cost >= 50:
		return "[red]H[-]"
	case cost >= 10:
		return "[yellow]M[-]"
	default:
		return "[green]C[-]"
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func buildNamespaceListText(nsPods nsCostInfo, hideZeroCost bool) string {
	const leftPad = "  "
	header := fmt.Sprintf("[darkcyan]%-30s %10s %10s %12s[-]", "NAME", "CPU", "MEM", "COST")
	separator := "--------------------------------------------------------------------"
	lines := []string{leftPad + header, leftPad + separator}

	pods := append([]PodInfo(nil), nsPods.pods...)
	sort.Slice(pods, func(i, j int) bool { return pods[i].Cost > pods[j].Cost })
	rowCount := 0
	for _, pod := range pods {
		if hideZeroCost && pod.Cost == 0 {
			continue
		}
		lines = append(lines, leftPad+fmt.Sprintf(
			"%-30s %10s %10s %12s",
			truncateString(pod.Name, 30),
			pod.CPU,
			pod.Memory,
			fmt.Sprintf("$%.2f", pod.Cost),
		))
		rowCount++
	}
	if rowCount == 0 {
		lines = append(lines, leftPad+"[gray]No non-zero cost pods in this namespace[-]")
	}

	lines = append(lines, leftPad+separator)
	lines = append(lines, leftPad+fmt.Sprintf("[green]%-30s %10s %10s %12s[-]", "TOTAL", "", "", fmt.Sprintf("$%.2f", nsPods.cost)))
	return strings.Join(lines, "\n")
}

func buildNodesListText(nodes []NodeInfo) string {
	const leftPad = "  "
	header := fmt.Sprintf("[darkcyan]%-12s %10s %12s %12s %12s[-]", "NODE", "MEMORY", "HARDWARE", "ELECTRICITY", "TOTAL")
	separator := "--------------------------------------------------------------------------"
	lines := []string{leftPad + header, leftPad + separator}
	var total float64

	for _, node := range nodes {
		lines = append(lines, leftPad+fmt.Sprintf(
			"%-12s %10s %12s %12s %12s",
			truncateString(node.Name, 12),
			fmt.Sprintf("%.1fGB", node.MemoryGB),
			fmt.Sprintf("$%.2f", node.HardwareCost),
			fmt.Sprintf("$%.2f", node.ElecCost),
			fmt.Sprintf("$%.2f", node.TotalCost),
		))
		total += node.TotalCost
	}
	lines = append(lines, leftPad+separator)
	lines = append(lines, leftPad+fmt.Sprintf("[green]%-12s %10s %12s %12s %12s[-]", "TOTAL", "", "", "", fmt.Sprintf("$%.2f", total)))
	return strings.Join(lines, "\n")
}

func buildPodDetailText(pod PodInfo, freshness StatsFreshness) string {
	lines := []string{
		fmt.Sprintf("  Pod:        [white]%s[-]", pod.Name),
		fmt.Sprintf("  Namespace:  [white]%s[-]", pod.Namespace),
		fmt.Sprintf("  CPU Req:    [white]%s[-]", pod.CPU),
		fmt.Sprintf("  Mem Req:    [white]%s[-]", pod.Memory),
		fmt.Sprintf("  Monthly:    [white]$%.2f[-]", pod.Cost),
		"",
		"  [darkcyan]Prometheus Data Freshness[-]",
	}

	if !freshness.Ready {
		lines = append(lines,
			fmt.Sprintf("  Status:     [yellow]Unavailable[-] (%s)", truncateString(freshness.Note, 64)),
			"",
			"  [gray]Tip: configure stats.base_url to show scrape-history confidence here.[-]",
			"",
			"  [gray]Esc/Enter to close[-]",
		)
		return strings.Join(lines, "\n")
	}

	confidenceLabel, confidenceColor := freshnessConfidence(freshness.ObservedDuration)
	lastAge := "now"
	if !freshness.LastSampleAt.IsZero() {
		age := time.Since(freshness.LastSampleAt).Round(time.Minute)
		if age > 0 {
			lastAge = fmt.Sprintf("%s ago", age)
		}
	}

	lines = append(lines,
		fmt.Sprintf("  Source:     %s", truncateString(freshness.BaseURL, 56)),
		fmt.Sprintf("  Coverage:   [white]%s[-] observed of [white]%s[-] lookback", formatShortDuration(freshness.ObservedDuration), formatShortDuration(freshness.LookbackDuration)),
		fmt.Sprintf("  Samples:    [white]%d[-] points", freshness.SampleCount),
		fmt.Sprintf("  Last Seen:  [white]%s[-]", lastAge),
		fmt.Sprintf("  Confidence: [%s]%s[-]", confidenceColor, confidenceLabel),
		"",
		"  [gray]Esc/Enter to close[-]",
	)

	return strings.Join(lines, "\n")
}

func freshnessConfidence(observed time.Duration) (string, string) {
	switch {
	case observed < 30*time.Minute:
		return "Very low (early scrape history)", "red"
	case observed < 2*time.Hour:
		return "Low (still warming up)", "yellow"
	case observed < 6*time.Hour:
		return "Moderate", "yellow"
	default:
		return "High", "green"
	}
}

func formatShortDuration(d time.Duration) string {
	if d <= 0 {
		return "0m"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if m == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh%dm", h, m)
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

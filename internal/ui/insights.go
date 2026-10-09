package ui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"heimdall/internal/core"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	insightsGroupCount         = 4
	insightsMinimumWidth       = 40
	insightsMinimumHeight      = 16
	insightsMinimumPanelHeight = 9
	insightsMinimumTableHeight = 3
	insightsTableHeadingRows   = 2
	insightsFrameRows          = 2
	insightsPanelColumns       = 4
	insightsSummaryHeight      = 7
	insightsSummaryColumns     = 3
	insightsSplitWidth         = 112
	insightsDetailWidth        = 32
	insightsColumnGap          = 3
	insightsValueWidth         = 12
	insightsCountWidth         = 7
	insightsStepWidth          = 10
	insightsRankWidth          = 4
	insightsSmallestCost       = 0.0001
	insightsBarWidth           = 8
	insightsBarGap             = 2
	insightsChartMinimumWidth  = 56
)

var insightsCaptionColor = lipgloss.Color("#A6B0BC")

type reportExportMsg struct {
	Path string
	Err  error
}

func exportAnalysisReport(report core.SessionAnalysisReport, directory string) (string, error) {
	file, err := os.CreateTemp(directory, "heimdall-report-"+time.Now().Format("20060102T150405")+"-*.json")
	if err != nil {
		return "", err
	}
	writeErr := core.WriteAnalysisReport(file, report, "json")
	closeErr := file.Close()
	if writeErr != nil {
		_ = os.Remove(file.Name())
		return "", writeErr
	}
	if closeErr != nil {
		_ = os.Remove(file.Name())
		return "", closeErr
	}
	return file.Name(), nil
}

func (m Model) exportAnalysisReportCmd() tea.Cmd {
	report := m.analysisReport
	return func() tea.Msg { path, err := exportAnalysisReport(report, "."); return reportExportMsg{path, err} }
}

func (m Model) insightItems() (string, []core.RankedObservation) {
	switch m.insightsGroup {
	case 1:
		return "Task output · observed tokens", m.analysisReport.TaskOutputs
	case 2:
		return "Task thinking · observed tokens", m.analysisReport.TaskThinking
	case 3:
		return "Tool outputs · local estimates", m.analysisReport.ToolOutputs
	default:
		return "Task usage · observed processing", m.analysisReport.TaskUsage
	}
}

func (m Model) renderInsightsView() string {
	width, height := max(insightsMinimumWidth, m.width), max(insightsMinimumHeight, m.height-insightsFrameRows)
	if !m.reportReady {
		lines := []string{TitleStyle.Render("SESSION INSIGHTS"), "", "Preparing your session analysis…", "Ranked observations will appear when source data is ready."}
		if m.sourceHealth.State == core.MonitorDegraded {
			lines[2] = "Session data is currently unavailable."
		}
		return insightPanel(lines, width, height)
	}
	summary := m.insightSummary(width)
	panelHeight := max(insightsMinimumPanelHeight, height-insightsSummaryHeight)
	contentWidth := max(1, width-insightsPanelColumns)
	contentHeight := max(1, panelHeight-insightsFrameRows)
	title, items := m.insightItems()
	limit := min(len(items), core.InsightsRankingLimit)
	selected := max(0, min(m.insightsIndex, limit-1))
	tabs := []string{"Task usage", "Task output", "Task thinking", "Tool outputs"}
	for i, tab := range tabs {
		style := lipgloss.NewStyle().Foreground(ColorLightText)
		if i == m.insightsGroup {
			style = style.Bold(true).Foreground(ColorSecondary).Underline(true)
		}
		tabs[i] = style.Render(tab)
	}
	lines := []string{strings.Join(tabs, "   "), ""}
	header := insightPair(TitleStyle.Render(title), fmt.Sprintf("Top %d / %d", limit, len(items)), contentWidth)
	lines = append(lines, header)
	available := max(1, contentHeight-len(lines))
	if limit == 0 {
		lines = append(lines, "", "No comparable observations yet.")
		lines = append(lines, wrapText(insightDescription(m.insightsGroup), contentWidth)...)
	} else if width >= insightsSplitWidth {
		listWidth := contentWidth - insightsDetailWidth - insightsColumnGap
		table := m.insightTable(items, selected, listWidth, available)
		detail := m.insightDetails(items[selected], insightsDetailWidth)
		for i := 0; i < available; i++ {
			left, right := "", ""
			if i < len(table) {
				left = table[i]
			}
			if i < len(detail) {
				right = detail[i]
			}
			lines = append(lines, insightCell(left, listWidth, false)+strings.Repeat(" ", insightsColumnGap)+insightCell(right, insightsDetailWidth, false))
		}
	} else {
		detail := m.insightCompactDetails(items[selected], contentWidth)
		tableHeight := max(insightsMinimumTableHeight, available-len(detail)-1)
		lines = append(lines, m.insightTable(items, selected, contentWidth, tableHeight)...)
		lines = append(lines, "")
		lines = append(lines, detail...)
	}
	return lipgloss.JoinVertical(lipgloss.Left, summary, insightPanel(lines, width, panelHeight))
}

func (m Model) insightSummary(width int) string {
	available := max(1, width-insightsPanelColumns)
	report := m.analysisReport
	stats := report.Metrics.TotalStats
	input, output, cost := "—", "—", "—"
	if report.Coverage.CompleteInput > 0 {
		input = formatCommas(stats.TotalProcessedTokenSum)
	}
	for _, generation := range report.Generations {
		if generation.ThinkingOutput.Available || generation.ContentOutput.Available {
			output = formatCommas(stats.TotalOutputTokenSum)
			break
		}
	}
	if stats.HasEstimatedCost {
		cost = insightCost(stats.EstimatedCostUSD)
	}
	if report.Coverage.CostPartial && stats.HasEstimatedCost {
		cost += " *"
	}
	labels, values := []string{"INPUT TOKENS", "OUTPUT TOKENS", "REFERENCE COST"}, []string{input, output, cost}
	columnWidth := available / insightsSummaryColumns
	labelLine, valueLine := "", ""
	for i, label := range labels {
		cellWidth := columnWidth
		if i == len(labels)-1 {
			cellWidth = available - columnWidth*i
		}
		labelLine += insightCell(lipgloss.NewStyle().Foreground(insightsCaptionColor).Render(label), cellWidth, false)
		valueLine += insightCell(lipgloss.NewStyle().Bold(true).Foreground(ColorLightText).Render(values[i]), cellWidth, false)
	}
	health := "Ready"
	healthColor := ColorSuccess
	if m.sourceLoading {
		health, healthColor = "Loading", ColorWarning
	} else if m.sourceHealth.State == core.MonitorDegraded {
		health, healthColor = "Read interrupted", ColorWarning
	} else if m.sourceHealth.State == core.MonitorStopped {
		health, healthColor = "Stopped", ColorMuted
	}
	readTime := "—"
	if !m.sourceHealth.LastSuccess.IsZero() {
		readTime = m.sourceHealth.LastSuccess.Format(time.TimeOnly)
	}
	status := lipgloss.NewStyle().Foreground(healthColor).Render(health) + " · " + readTime
	coverage := fmt.Sprintf("Usage %d/%d turns · Priced %d/%d", report.Coverage.CompleteInput, report.Coverage.Generations, report.Coverage.PricedGenerations, report.Coverage.Generations)
	if report.Coverage.CostPartial {
		coverage += " · Partial estimate"
	}
	return insightPanel([]string{insightPair(TitleStyle.Render("SESSION INSIGHTS"), status, available), "", labelLine, valueLine, lipgloss.NewStyle().Foreground(insightsCaptionColor).Render(coverage)}, width, insightsSummaryHeight)
}

func (m Model) insightTable(items []core.RankedObservation, selected, width, height int) []string {
	limit := min(len(items), core.InsightsRankingLimit)
	count := min(limit, max(1, height-insightsTableHeadingRows))
	start := max(0, min(selected-count+1, limit-count))
	valueWidth := insightsValueWidth
	labelWidth := max(1, width-insightsRankWidth-insightsStepWidth-insightsCountWidth-valueWidth)
	chartWidth := 0
	if width >= insightsChartMinimumWidth {
		chartWidth = insightsBarWidth + insightsBarGap
		labelWidth -= chartWidth
	}
	label := "USER TASK"
	value := "TOKENS"
	stepLabel := "USER STEP"
	if m.insightsGroup == 3 {
		label = "TOOL / ACTION"
		stepLabel = "STEP"
	}
	row := func(rank, step, name, chart, count, value string) string {
		return insightCell(rank, insightsRankWidth, false) + insightCell(step, insightsStepWidth, false) + insightCell(name, labelWidth, false) + insightCell(chart, chartWidth, false) + insightCell(count, insightsCountWidth, true) + insightCell(value, valueWidth, true)
	}
	lines := []string{lipgloss.NewStyle().Foreground(insightsCaptionColor).Render(row("", stepLabel, label, "RELATIVE", "STEPS", value)), lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", width))}
	for i := start; i < start+count; i++ {
		item := items[i]
		step := "—"
		if item.HasStepIndex {
			step = fmt.Sprintf("#%d", item.StepIndex)
		}
		amount := formatCommas(int(item.Amount.Value))
		stepCount := "1"
		if item.Task != nil {
			stepCount = formatCommas(item.Task.StepCount)
		}
		name := item.Label
		rank := fmt.Sprintf("%d", i+1)
		style := lipgloss.NewStyle().Foreground(ColorLightText)
		if i == selected {
			rank = "›" + rank
		}
		chart := ""
		if chartWidth > 0 && items[0].Amount.Available && items[0].Amount.Value > 0 && item.Amount.Available {
			chart = renderProportionBar(item.Amount.Value/items[0].Amount.Value, insightsBarWidth, ColorSecondary)
		}
		identity := insightCell(rank, insightsRankWidth, false) + insightCell(step, insightsStepWidth, false) + insightCell(name, labelWidth, false)
		comparison := insightCell(chart, chartWidth, false) + insightCell(stepCount, insightsCountWidth, true) + insightCell(amount, valueWidth, true)
		if i == selected {
			selectedStyle := style.Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(ColorPrimary)
			lines = append(lines, selectedStyle.Render(identity)+style.Render(comparison))
			continue
		}
		lines = append(lines, style.Render(row(rank, step, name, chart, stepCount, amount)))
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return lines
}

func (m Model) insightDetails(item core.RankedObservation, width int) []string {
	source, locator := insightSource(item)
	lines := []string{TitleStyle.Render("SELECTED EVIDENCE"), "", lipgloss.NewStyle().Bold(true).Foreground(ColorLightText).Render(item.Label)}
	lines = append(lines, wrapText(insightMeasurementLabel(item), width)...)
	lines = append(lines, "", lipgloss.NewStyle().Foreground(insightsCaptionColor).Render("SOURCE"))
	lines = append(lines, wrapVisualLines(source, width)...)
	lines = append(lines, wrapVisualLines(locator, width)...)
	if item.Task != nil {
		lines = append(lines, "")
		lines = append(lines, taskInsightDetailLines(*item.Task, item.StepIndex, width)...)
	}
	lines = append(lines, "")
	lines = append(lines, wrapText(insightDescription(m.insightsGroup), width)...)
	lines = append(lines, "", "Bars compare with the largest", "item in this ranking.")
	lines = append(lines, "", KeyStyle.Render("Enter")+" Inspect in History")
	return lines
}

func (m Model) insightCompactDetails(item core.RankedObservation, width int) []string {
	source, locator := insightSource(item)
	measurement := insightMeasurementLabel(item)
	if item.Amount.Kind == core.MeasurementEstimated {
		measurement = "Local estimate"
	}
	lines := []string{
		lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", width)),
		insightPair(TitleStyle.Render("SELECTED EVIDENCE"), measurement, width),
		insightCell(source+" · "+locator, width, false),
		lipgloss.NewStyle().Foreground(insightsCaptionColor).Render(insightCell(insightCompactDescription(m.insightsGroup), width, false)),
	}
	if item.Task != nil {
		lines = append(lines, insightCell(compactTaskInsight(*item.Task, item.StepIndex), width, false))
	}
	return lines
}

func insightCompactDescription(group int) string {
	switch group {
	case 1:
		return "Visible output is summed across model calls in one user task."
	case 2:
		return "Thinking is summed across model calls in one user task."
	case 3:
		return "Local tool-result size; not model output or billing."
	default:
		return "Repeated input processing is counted for every model call."
	}
}

func insightSource(item core.RankedObservation) (string, string) {
	if len(item.Evidence) == 0 {
		return "Source location unavailable", "Enter opens the linked step, when available."
	}
	evidence := item.Evidence[0]
	return filepath.Base(evidence.Source.Path), evidence.Locator
}

func insightMeasurementLabel(item core.RankedObservation) string {
	if item.Amount.Kind == core.MeasurementEstimated {
		return "Local estimate · " + item.Amount.Method
	}
	if item.Task != nil && item.Task.PartialUsage {
		return "Partial task total · " + item.Amount.Method
	}
	return "Calculated from recorded usage"
}

func insightDescription(group int) string {
	switch group {
	case 1:
		return "Visible content output is summed across every observed model call in one user task."
	case 2:
		return "Thinking output is summed across every observed model call in one user task."
	case 3:
		return "Local tool-result text estimate. It excludes model thinking and visible model output."
	default:
		return "All observed model-call input and output usage in one user task. Repeated context is counted again."
	}
}

func taskInsightDetailLines(task core.TaskObservationDetails, startStep, width int) []string {
	partial := "complete"
	if task.PartialUsage {
		partial = "partial"
	}
	lines := []string{
		fmt.Sprintf("Task steps #%d–#%d", startStep, task.EndStepIndex),
		fmt.Sprintf("History steps %d", task.StepCount),
		fmt.Sprintf("Model calls %d · usage %d (%s)", task.ModelCalls, task.UsageCalls, partial),
		fmt.Sprintf("Compactions %d", task.Compactions),
		fmt.Sprintf("Input %s", insightMeasurementValue(task.InputTokens)),
		fmt.Sprintf("Thinking %s", insightMeasurementValue(task.ThinkingTokens)),
		fmt.Sprintf("Content %s", insightMeasurementValue(task.ContentOutputTokens)),
		fmt.Sprintf("Tool output %s", insightMeasurementValue(task.ToolOutputTokens)),
		"Context start / peak / end",
		fmt.Sprintf("%s / %s / %s", insightMeasurementValue(task.StartingContextTokens), insightMeasurementValue(task.PeakContextTokens), insightMeasurementValue(task.EndingContextTokens)),
	}
	wrapped := make([]string, 0, len(lines))
	for _, line := range lines {
		wrapped = append(wrapped, wrapText(line, width)...)
	}
	return wrapped
}

func compactTaskInsight(task core.TaskObservationDetails, startStep int) string {
	partial := ""
	if task.PartialUsage {
		partial = " · partial"
	}
	return fmt.Sprintf("Task #%d–#%d · %d steps · %d calls · %d compact%s", startStep, task.EndStepIndex, task.StepCount, task.ModelCalls, task.Compactions, partial)
}

func insightMeasurementValue(measurement core.Measurement) string {
	if !measurement.Available {
		return "—"
	}
	return formatCommas(int(measurement.Value))
}

func insightCost(value float64) string {
	if value > 0 && value < insightsSmallestCost {
		return "<$0.0001"
	}
	return fmt.Sprintf("$%.4f", value)
}

func insightCell(text string, width int, right bool) string {
	text = truncateVisualWidth(text, max(0, width))
	space := strings.Repeat(" ", max(0, width-lipgloss.Width(text)))
	if right {
		return space + text
	}
	return text + space
}

func insightPair(left, right string, width int) string {
	right = truncateVisualWidth(right, width/2)
	leftWidth := max(0, width-lipgloss.Width(right)-1)
	return insightCell(left, leftWidth, false) + " " + right
}

func insightPanel(lines []string, width, height int) string {
	inside := max(1, width-insightsPanelColumns)
	output := make([]string, 0, height-insightsFrameRows)
	for i := 0; i < height-insightsFrameRows; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		output = append(output, " "+insightCell(line, inside, false)+" ")
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(ColorBorder).Render(strings.Join(output, "\n"))
}

func (m *Model) updateInsightsKey(key string) {
	_, items := m.insightItems()
	limit := min(len(items), core.InsightsRankingLimit)
	switch key {
	case "l", "right":
		m.insightsGroup = (m.insightsGroup + 1) % insightsGroupCount
		m.insightsIndex = 0
	case "h", "left":
		m.insightsGroup = (m.insightsGroup + insightsGroupCount - 1) % insightsGroupCount
		m.insightsIndex = 0
	case "j", "down":
		m.insightsIndex = min(max(0, limit-1), m.insightsIndex+1)
	case "k", "up":
		m.insightsIndex = max(0, m.insightsIndex-1)
	case "enter":
		if m.insightsIndex < 0 || m.insightsIndex >= limit {
			return
		}
		item := items[m.insightsIndex]
		if !item.HasStepIndex {
			m.statusMessage = "No unambiguous transcript step for this generation"
			m.statusMessageTime = time.Now()
			return
		}
		key := fmt.Sprintf("%s:%d", m.sessionID, item.StepIndex)
		index, exists := m.historyIndex[key]
		if !exists {
			m.statusMessage = "The referenced step is unavailable"
			m.statusMessageTime = time.Now()
			return
		}
		m.activeView, m.focusPane = ViewHistory, FocusDetail
		m.historyTypeFilter, m.historyCacheFilter, m.historyStepQuery = TypeFilterAll, CacheFilterAll, ""
		m.selectedIdx, m.detailScroll = len(m.history)-1-index, 0
		m.historyOffset = m.selectedIdx
	}
}

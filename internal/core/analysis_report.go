package core

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
)

const (
	ReportSchemaVersion    = 2
	InsightsRankingLimit   = 10
	MeasurementObserved    = "observed"
	MeasurementDerived     = "derived"
	MeasurementEstimated   = "local-estimate"
	MeasurementUnavailable = "unavailable"
	generationNumberBase   = 10
	generationNumberBits   = 64
)

// Measurement retains missingness and the method behind every reported value.
type Measurement struct {
	Value     float64 `json:"value"`
	Available bool    `json:"available"`
	Kind      string  `json:"kind"`
	Method    string  `json:"method"`
}

func observed(value int, available bool, method string) Measurement {
	if !available {
		return Measurement{Kind: MeasurementUnavailable, Method: method}
	}
	return Measurement{Value: float64(value), Available: true, Kind: MeasurementObserved, Method: method}
}

func estimatedTextTokens(content string) Measurement {
	method := defaultEncoding
	if _, err := GetTokenizer(); err != nil {
		method = "byte-length / 4 fallback"
	}
	return Measurement{Value: float64(CountTokens(content)), Available: true, Kind: MeasurementEstimated, Method: method}
}

// GenerationAnalysis keeps billing evidence separate from readable content.
type GenerationAnalysis struct {
	ID             string       `json:"id"`
	StepIndex      int          `json:"step_index"`
	HasStepIndex   bool         `json:"has_step_index"`
	Model          string       `json:"model"`
	Provider       ProviderName `json:"pricing_provider"`
	Context        Measurement  `json:"context"`
	UncachedInput  Measurement  `json:"uncached_input"`
	CachedInput    Measurement  `json:"cached_input"`
	ThinkingOutput Measurement  `json:"thinking_output"`
	ContentOutput  Measurement  `json:"content_output"`
	TotalOutput    Measurement  `json:"total_output"`
	TotalTokens    Measurement  `json:"total_tokens"`
	EstimatedCost  Measurement  `json:"estimated_cost_usd"`
	CostPartial    bool         `json:"cost_partial"`
	Pricing        *ModelInfo   `json:"pricing,omitempty"`
	Evidence       []Evidence   `json:"evidence"`
}

type RankedObservation struct {
	StepIndex            int                     `json:"step_index"`
	HasStepIndex         bool                    `json:"has_step_index"`
	GenerationID         string                  `json:"generation_id,omitempty"`
	PreviousGenerationID string                  `json:"previous_generation_id,omitempty"`
	Label                string                  `json:"label"`
	Amount               Measurement             `json:"amount"`
	Task                 *TaskObservationDetails `json:"task,omitempty"`
	Evidence             []Evidence              `json:"evidence"`
}

// TaskObservationDetails describes one user-request span without treating a
// compaction boundary as a new user task.
type TaskObservationDetails struct {
	EndStepIndex          int         `json:"end_step_index"`
	StepCount             int         `json:"step_count"`
	ModelCalls            int         `json:"model_calls"`
	UsageCalls            int         `json:"usage_calls"`
	ThinkingCalls         int         `json:"thinking_calls"`
	ContentOutputCalls    int         `json:"content_output_calls"`
	Compactions           int         `json:"compactions"`
	PartialUsage          bool        `json:"partial_usage"`
	InputTokens           Measurement `json:"input_tokens"`
	ThinkingTokens        Measurement `json:"thinking_tokens"`
	ContentOutputTokens   Measurement `json:"content_output_tokens"`
	ToolOutputTokens      Measurement `json:"tool_output_tokens"`
	StartingContextTokens Measurement `json:"starting_context_tokens"`
	PeakContextTokens     Measurement `json:"peak_context_tokens"`
	EndingContextTokens   Measurement `json:"ending_context_tokens"`
}

type ReportCoverage struct {
	Generations       int  `json:"generations"`
	CompleteInput     int  `json:"complete_input"`
	PricedGenerations int  `json:"priced_generations"`
	CostPartial       bool `json:"cost_partial"`
	ParserDiagnostics int  `json:"parser_diagnostics"`
}

// SessionAnalysisReport deliberately contains no prompts or tool output text.
type SessionAnalysisReport struct {
	SchemaVersion int                     `json:"schema_version"`
	Session       SessionRef              `json:"session"`
	Revision      uint64                  `json:"revision"`
	Health        MonitorHealth           `json:"health"`
	Coverage      ReportCoverage          `json:"coverage"`
	Metrics       SessionAggregateMetrics `json:"metrics"`
	Generations   []GenerationAnalysis    `json:"generations"`
	TaskUsage     []RankedObservation     `json:"task_usage"`
	TaskThinking  []RankedObservation     `json:"task_thinking"`
	ToolOutputs   []RankedObservation     `json:"tool_outputs"`
	TaskOutputs   []RankedObservation     `json:"task_outputs"`
	Notes         []string                `json:"notes"`
}

func canonicalGenerations(generations []Generation) []Generation {
	result := make([]Generation, 0, len(generations))
	positions := make(map[string]int)
	for _, generation := range generations {
		if index, exists := positions[generation.ID]; exists && generation.ID != "" {
			result[index] = generation
			continue
		}
		positions[generation.ID] = len(result)
		result = append(result, generation)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].StepIndex != result[j].StepIndex {
			return result[i].StepIndex < result[j].StepIndex
		}
		return generationIDLess(result[i].ID, result[j].ID)
	})
	return result
}

func generationIDLess(left, right string) bool {
	l, le := strconv.ParseUint(left, generationNumberBase, generationNumberBits)
	r, re := strconv.ParseUint(right, generationNumberBase, generationNumberBits)
	if le == nil && re == nil && l != r {
		return l < r
	}
	return left < right
}

func generationMetrics(generations []Generation) SessionAggregateMetrics {
	events := make([]UnifiedAgentEvent, 0, len(generations))
	for _, generation := range canonicalGenerations(generations) {
		events = append(events, UnifiedAgentEvent{Usage: projectUsageObservation(generation)})
	}
	return ComputeSessionAggregateMetrics(events)
}

func BuildSessionAnalysisReport(session Session) SessionAnalysisReport {
	report := SessionAnalysisReport{
		SchemaVersion: ReportSchemaVersion, Session: session.Ref, Revision: session.Revision,
		Health: MonitorHealth{State: MonitorSnapshot}, Metrics: generationMetrics(session.Generations),
		Generations: []GenerationAnalysis{}, TaskUsage: []RankedObservation{}, TaskThinking: []RankedObservation{},
		TaskOutputs: []RankedObservation{}, ToolOutputs: []RankedObservation{},
		Notes: []string{
			"Tool output sizes are local text estimates, not per-tool invoices or proof of repeated transmission.",
			"Task rankings span one user step through the step before the next user step.",
			"Task usage sums repeated model-call input processing; it is not unique context size.",
			"Missing cache scalars in complete input usage are inferred zero; other missing fields remain unavailable.",
			"Compaction stays inside its user task and is counted, but unpersisted compactor usage remains unavailable.",
			"Starting, peak and ending context are observed states, not attribution of context to one message.",
		},
	}
	steps := make(map[int]Step, len(session.Steps))
	for _, step := range session.Steps {
		steps[step.Index] = step
	}
	for _, step := range session.Steps {
		event := UnifiedAgentEvent{Type: StepType(step.Kind), Scope: step.Scope}
		if !event.IsLocalStep() || step.Content == "" {
			continue
		}
		label := "unknown tool (" + step.Kind + ")"
		if len(step.ToolCalls) == 1 {
			label = step.ToolCalls[0].Name
		} else if parent, ok := steps[step.ParentStepIndex]; ok && len(parent.ToolCalls) == 1 {
			label = parent.ToolCalls[0].Name
		}
		report.ToolOutputs = append(report.ToolOutputs, RankedObservation{StepIndex: step.Index, HasStepIndex: true, Label: label,
			Amount: estimatedTextTokens(step.Content), Evidence: append([]Evidence(nil), step.Evidence...)})
	}
	for _, generation := range canonicalGenerations(session.Generations) {
		item := analyzeGeneration(generation)
		report.Coverage.Generations++
		if item.UncachedInput.Available {
			report.Coverage.CompleteInput++
		}
		if item.EstimatedCost.Available {
			report.Coverage.PricedGenerations++
		}
		if item.CostPartial || !item.EstimatedCost.Available {
			report.Coverage.CostPartial = true
		}
		report.Generations = append(report.Generations, item)
	}
	report.TaskUsage, report.TaskThinking, report.TaskOutputs = buildTaskRankings(session.Steps, report.Generations)
	for _, ranking := range [][]RankedObservation{report.TaskUsage, report.TaskOutputs, report.TaskThinking, report.ToolOutputs} {
		sortRanking(ranking)
	}
	report.Coverage.ParserDiagnostics = session.DiagnosticCount
	return report
}

func analyzeGeneration(g Generation) GenerationAnalysis {
	u := g.Usage
	r := GenerationAnalysis{ID: g.ID, StepIndex: g.StepIndex, HasStepIndex: g.HasStepIndex || g.StepIndex > 0, Model: g.ModelID, Provider: g.Provider,
		Context:        observed(u.ObservedContextTokens, u.HasObservedContextTokens, "persisted context-state field"),
		UncachedInput:  observed(u.UncachedInputTokens, u.HasUncachedInputTokens, "persisted usage field"),
		CachedInput:    observed(u.CachedInputTokens, u.HasCachedInputTokens, "persisted usage field"),
		ThinkingOutput: observed(u.ThinkingOutputTokens, u.HasThinkingOutputTokens || u.ThinkingOutputTokens > 0, "persisted usage field"),
		ContentOutput:  observed(u.OutputContentTokens, u.HasOutputContentTokens || u.OutputContentTokens > 0, "persisted usage field"),
		TotalOutput:    observed(u.TotalOutputTokens, u.HasTotalOutputTokens, "persisted aggregate usage field"),
		TotalTokens:    Measurement{Kind: MeasurementUnavailable, Method: "sum of available usage components"},
		EstimatedCost:  Measurement{Kind: MeasurementUnavailable, Method: "catalog reference estimate"}, Evidence: append([]Evidence(nil), g.Evidence...),
	}
	if u.HasUncachedInputTokens && !u.HasCachedInputTokens {
		r.CachedInput = Measurement{Available: true, Kind: MeasurementDerived, Method: "omitted cache scalar inferred zero for complete input usage"}
	}
	if r.Provider == "" {
		r.Provider = ProviderVertexAI
	}
	output := r.TotalOutput
	if !output.Available && r.ThinkingOutput.Available && r.ContentOutput.Available {
		output = Measurement{Value: r.ThinkingOutput.Value + r.ContentOutput.Value, Available: true, Kind: MeasurementDerived, Method: "sum of observed output components"}
	}
	r.CostPartial = !r.UncachedInput.Available || !output.Available
	if r.UncachedInput.Available {
		r.TotalTokens = Measurement{Value: r.UncachedInput.Value + r.CachedInput.Value + output.Value, Available: true, Kind: MeasurementDerived, Method: "sum of complete input and available aggregate output"}
		if price, ok := ResolveModelInfoByString(r.Provider, r.Model); ok {
			r.Pricing = &price
			r.EstimatedCost = Measurement{Value: price.CalculateCost(int(r.UncachedInput.Value), int(r.CachedInput.Value), int(output.Value)), Available: true, Kind: MeasurementDerived, Method: "catalog reference rates; omitted output is excluded"}
		}
	}
	return r
}

type taskRankingAccumulator struct {
	startStep      Step
	endStepIndex   int
	nextStartIndex int
	details        TaskObservationDetails
	processed      float64
	input          float64
	thinking       float64
	contentOutput  float64
	evidence       []Evidence
	firstGenID     string
	lastGenID      string
	contextSeen    bool
}

func buildTaskRankings(steps []Step, generations []GenerationAnalysis) ([]RankedObservation, []RankedObservation, []RankedObservation) {
	tasks := taskAccumulators(steps)
	if len(tasks) == 0 {
		return []RankedObservation{}, []RankedObservation{}, []RankedObservation{}
	}
	taskIndex := 0
	for _, generation := range generations {
		if !generation.HasStepIndex {
			continue
		}
		for taskIndex+1 < len(tasks) && generation.StepIndex >= tasks[taskIndex+1].startStep.Index {
			taskIndex++
		}
		task := &tasks[taskIndex]
		if generation.StepIndex < task.startStep.Index || task.nextStartIndex > 0 && generation.StepIndex >= task.nextStartIndex {
			continue
		}
		accumulateTaskGeneration(task, generation)
	}

	usage := make([]RankedObservation, 0, len(tasks))
	thinking := make([]RankedObservation, 0, len(tasks))
	content := make([]RankedObservation, 0, len(tasks))
	for index := range tasks {
		task := &tasks[index]
		finalizeTaskDetails(task)
		if task.details.UsageCalls > 0 {
			usage = append(usage, taskRanking(*task, Measurement{Value: task.processed, Available: true, Kind: MeasurementDerived, Method: "sum of available generation usage within user-step span"}))
		}
		if task.details.ThinkingCalls > 0 {
			thinking = append(thinking, taskRanking(*task, task.details.ThinkingTokens))
		}
		if task.details.ContentOutputCalls > 0 {
			content = append(content, taskRanking(*task, task.details.ContentOutputTokens))
		}
	}
	return usage, thinking, content
}

func taskAccumulators(steps []Step) []taskRankingAccumulator {
	ordered := append([]Step(nil), steps...)
	sort.SliceStable(ordered, func(left, right int) bool { return ordered[left].Index < ordered[right].Index })
	tasks := make([]taskRankingAccumulator, 0)
	for _, step := range ordered {
		isUser := StepType(step.Kind) == StepTypeUserInput || step.Scope == ScopeUserInteraction
		if isUser {
			if len(tasks) > 0 {
				tasks[len(tasks)-1].nextStartIndex = step.Index
			}
			tasks = append(tasks, taskRankingAccumulator{startStep: step, endStepIndex: step.Index, details: TaskObservationDetails{StepCount: 1}, evidence: append([]Evidence(nil), step.Evidence...)})
			continue
		}
		if len(tasks) == 0 {
			continue
		}
		task := &tasks[len(tasks)-1]
		task.endStepIndex = step.Index
		task.details.StepCount++
		if StepType(step.Kind) == StepTypeCheckpoint || step.Scope == ScopeSystemCompaction {
			task.details.Compactions++
		}
		event := UnifiedAgentEvent{Type: StepType(step.Kind), Scope: step.Scope}
		if event.IsLocalStep() && step.Content != "" {
			measurement := estimatedTextTokens(step.Content)
			task.details.ToolOutputTokens.Value += measurement.Value
			task.details.ToolOutputTokens.Available = true
			task.details.ToolOutputTokens.Kind = MeasurementEstimated
			task.details.ToolOutputTokens.Method = measurement.Method
		}
	}
	return tasks
}

func accumulateTaskGeneration(task *taskRankingAccumulator, generation GenerationAnalysis) {
	task.details.ModelCalls++
	if task.firstGenID == "" {
		task.firstGenID = generation.ID
	}
	task.lastGenID = generation.ID
	if len(task.evidence) == 0 {
		task.evidence = append([]Evidence(nil), generation.Evidence...)
	}
	if generation.TotalTokens.Available {
		task.processed += generation.TotalTokens.Value
		task.details.UsageCalls++
	}
	if generation.UncachedInput.Available {
		task.input += generation.UncachedInput.Value + generation.CachedInput.Value
	}
	if generation.ThinkingOutput.Available {
		task.thinking += generation.ThinkingOutput.Value
		task.details.ThinkingCalls++
	}
	if generation.ContentOutput.Available {
		task.contentOutput += generation.ContentOutput.Value
		task.details.ContentOutputCalls++
	}
	if generation.CostPartial {
		task.details.PartialUsage = true
	}
	if generation.Context.Available {
		if !task.contextSeen {
			task.details.StartingContextTokens = generation.Context
			task.details.PeakContextTokens = generation.Context
			task.contextSeen = true
		}
		if generation.Context.Value > task.details.PeakContextTokens.Value {
			task.details.PeakContextTokens = generation.Context
		}
		task.details.EndingContextTokens = generation.Context
	}
}

func finalizeTaskDetails(task *taskRankingAccumulator) {
	task.details.EndStepIndex = task.endStepIndex
	task.details.PartialUsage = task.details.PartialUsage || task.details.UsageCalls < task.details.ModelCalls
	if task.details.UsageCalls > 0 {
		task.details.InputTokens = Measurement{Value: task.input, Available: true, Kind: MeasurementDerived, Method: "sum of available cached and uncached input usage"}
	}
	if task.details.ThinkingCalls > 0 {
		task.details.ThinkingTokens = Measurement{Value: task.thinking, Available: true, Kind: MeasurementDerived, Method: "sum of observed thinking output within user-step span"}
	}
	if task.details.ContentOutputCalls > 0 {
		task.details.ContentOutputTokens = Measurement{Value: task.contentOutput, Available: true, Kind: MeasurementDerived, Method: "sum of observed content output within user-step span"}
	}
}

func taskRanking(task taskRankingAccumulator, amount Measurement) RankedObservation {
	details := task.details
	return RankedObservation{
		StepIndex: task.startStep.Index, HasStepIndex: true, GenerationID: task.lastGenID, PreviousGenerationID: task.firstGenID,
		Label: taskLabel(task.startStep), Amount: amount, Task: &details, Evidence: append([]Evidence(nil), task.evidence...),
	}
}

func taskLabel(step Step) string {
	return fmt.Sprintf("User task #%d", step.Index)
}

func sortRanking(items []RankedObservation) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Amount.Value != items[j].Amount.Value {
			return items[i].Amount.Value > items[j].Amount.Value
		}
		if items[i].StepIndex != items[j].StepIndex {
			return items[i].StepIndex < items[j].StepIndex
		}
		return generationIDLess(items[i].GenerationID, items[j].GenerationID)
	})
}

// WriteAnalysisReport shares one wire contract between CLI and TUI exports.
func WriteAnalysisReport(writer io.Writer, report SessionAnalysisReport, format string) error {
	switch format {
	case "json":
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		return encoder.Encode(report)
	case "text":
		cost := "unavailable"
		if report.Metrics.TotalStats.HasEstimatedCost {
			cost = fmt.Sprintf("$%.6f USD", report.Metrics.TotalStats.EstimatedCostUSD)
		}
		if _, err := fmt.Fprintf(writer, "HEIMDALL SESSION REPORT\nSession: %s  Revision: %d\nSource: %s\nGenerations: %d  Complete input: %d  Priced: %d  Partial cost: %t\nReference cost: %s\n", report.Session.SessionID, report.Revision, report.Health.State, report.Coverage.Generations, report.Coverage.CompleteInput, report.Coverage.PricedGenerations, report.Coverage.CostPartial, cost); err != nil {
			return err
		}
		groups := []struct {
			name  string
			items []RankedObservation
		}{{"TASK USAGE (OBSERVED TOKEN PROCESSING)", report.TaskUsage}, {"TASK OUTPUT (OBSERVED OUTPUT)", report.TaskOutputs}, {"TASK THINKING (OBSERVED OUTPUT)", report.TaskThinking}, {"TOOL OUTPUTS (LOCAL TOKEN ESTIMATES)", report.ToolOutputs}}
		for _, group := range groups {
			if _, err := fmt.Fprintln(writer, "\n"+group.name); err != nil {
				return err
			}
			for _, item := range group.items {
				if _, err := fmt.Fprintf(writer, "step=%d linked=%t generation=%s %s: %.6f [%s; %s]\n", item.StepIndex, item.HasStepIndex, item.GenerationID, item.Label, item.Amount.Value, item.Amount.Kind, item.Amount.Method); err != nil {
					return err
				}
			}
		}
		for _, note := range report.Notes {
			if _, err := fmt.Fprintln(writer, note); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported report format %q (use text or json)", format)
	}
}

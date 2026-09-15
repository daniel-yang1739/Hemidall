package core

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
)

const (
	ReportSchemaVersion    = 1
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
	TotalTokens    Measurement  `json:"total_tokens"`
	EstimatedCost  Measurement  `json:"estimated_cost_usd"`
	CostPartial    bool         `json:"cost_partial"`
	Pricing        *ModelInfo   `json:"pricing,omitempty"`
	Evidence       []Evidence   `json:"evidence"`
}

type RankedObservation struct {
	StepIndex            int         `json:"step_index"`
	HasStepIndex         bool        `json:"has_step_index"`
	GenerationID         string      `json:"generation_id,omitempty"`
	PreviousGenerationID string      `json:"previous_generation_id,omitempty"`
	Label                string      `json:"label"`
	Amount               Measurement `json:"amount"`
	Evidence             []Evidence  `json:"evidence"`
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
	SchemaVersion  int                     `json:"schema_version"`
	Session        SessionRef              `json:"session"`
	Revision       uint64                  `json:"revision"`
	Health         MonitorHealth           `json:"health"`
	Coverage       ReportCoverage          `json:"coverage"`
	Metrics        SessionAggregateMetrics `json:"metrics"`
	Generations    []GenerationAnalysis    `json:"generations"`
	ToolOutputs    []RankedObservation     `json:"tool_outputs"`
	ContextGrowth  []RankedObservation     `json:"context_growth"`
	TokenConsumers []RankedObservation     `json:"token_consumers"`
	CostConsumers  []RankedObservation     `json:"cost_consumers"`
	Notes          []string                `json:"notes"`
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
		Generations: []GenerationAnalysis{}, ToolOutputs: []RankedObservation{}, ContextGrowth: []RankedObservation{},
		TokenConsumers: []RankedObservation{}, CostConsumers: []RankedObservation{},
		Notes: []string{
			"Tool output sizes are local text estimates, not per-tool invoices or proof of repeated transmission.",
			"Costs use the recorded catalog rates and pricing-provider assumption; they are not an invoice.",
			"Missing cache scalars in complete input usage are inferred zero; other missing fields remain unavailable.",
			"Context growth compares adjacent linked generations of the same model and source, without crossing checkpoints.",
			"Compaction changes do not establish compressor savings or task quality.",
		},
	}
	steps := make(map[int]Step, len(session.Steps))
	checkpoints := []int{}
	for _, step := range session.Steps {
		steps[step.Index] = step
		if StepType(step.Kind) == StepTypeCheckpoint || step.Scope == ScopeSystemCompaction {
			checkpoints = append(checkpoints, step.Index)
		}
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
		method := defaultEncoding
		if _, err := GetTokenizer(); err != nil {
			method = "byte-length / 4 fallback"
		}
		report.ToolOutputs = append(report.ToolOutputs, RankedObservation{StepIndex: step.Index, HasStepIndex: true, Label: label,
			Amount: Measurement{Value: float64(CountTokens(step.Content)), Available: true, Kind: MeasurementEstimated, Method: method}, Evidence: append([]Evidence(nil), step.Evidence...)})
	}
	var previous *GenerationAnalysis
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
		if previous != nil && comparableContext(*previous, item, checkpoints) {
			growth := item.Context.Value - previous.Context.Value
			if growth > 0 {
				report.ContextGrowth = append(report.ContextGrowth, RankedObservation{StepIndex: item.StepIndex, HasStepIndex: item.HasStepIndex,
					GenerationID: item.ID, PreviousGenerationID: previous.ID, Label: item.Model,
					Amount: Measurement{Value: growth, Available: true, Kind: MeasurementDerived, Method: "difference of observed context-state values"}, Evidence: item.Evidence})
			}
		}
		report.Generations = append(report.Generations, item)
		previous = &report.Generations[len(report.Generations)-1]
		if item.TotalTokens.Available {
			report.TokenConsumers = append(report.TokenConsumers, generationRanking(item, item.TotalTokens))
		}
		if item.EstimatedCost.Available {
			report.CostConsumers = append(report.CostConsumers, generationRanking(item, item.EstimatedCost))
		}
	}
	for _, ranking := range [][]RankedObservation{report.ToolOutputs, report.ContextGrowth, report.TokenConsumers, report.CostConsumers} {
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
		TotalTokens:    Measurement{Kind: MeasurementUnavailable, Method: "sum of available usage components"},
		EstimatedCost:  Measurement{Kind: MeasurementUnavailable, Method: "catalog reference estimate"}, Evidence: append([]Evidence(nil), g.Evidence...),
	}
	if u.HasUncachedInputTokens && !u.HasCachedInputTokens {
		r.CachedInput = Measurement{Available: true, Kind: MeasurementDerived, Method: "omitted cache scalar inferred zero for complete input usage"}
	}
	if r.Provider == "" {
		r.Provider = ProviderVertexAI
	}
	r.CostPartial = !r.UncachedInput.Available || !r.ThinkingOutput.Available || !r.ContentOutput.Available
	if r.UncachedInput.Available {
		r.TotalTokens = Measurement{Value: r.UncachedInput.Value + r.CachedInput.Value + r.ThinkingOutput.Value + r.ContentOutput.Value, Available: true, Kind: MeasurementDerived, Method: "sum of available usage components; see field availability"}
		if price, ok := ResolveModelInfoByString(r.Provider, r.Model); ok {
			r.Pricing = &price
			r.EstimatedCost = Measurement{Value: price.CalculateCost(int(r.UncachedInput.Value), int(r.CachedInput.Value), int(r.ThinkingOutput.Value+r.ContentOutput.Value)), Available: true, Kind: MeasurementDerived, Method: "catalog reference rates; omitted output is excluded"}
		}
	}
	return r
}

func comparableContext(previous, current GenerationAnalysis, checkpoints []int) bool {
	if !previous.Context.Available || !current.Context.Available || !previous.HasStepIndex || !current.HasStepIndex || previous.Model == "" || current.Model != previous.Model || current.Provider != previous.Provider || current.StepIndex < previous.StepIndex {
		return false
	}
	previousID, previousErr := strconv.ParseUint(previous.ID, generationNumberBase, generationNumberBits)
	currentID, currentErr := strconv.ParseUint(current.ID, generationNumberBase, generationNumberBits)
	if previousErr == nil && currentErr == nil && currentID-previousID != 1 {
		return false
	}
	if len(previous.Evidence) > 0 && len(current.Evidence) > 0 && previous.Evidence[0].Source != current.Evidence[0].Source {
		return false
	}
	for _, index := range checkpoints {
		if index > previous.StepIndex && index <= current.StepIndex {
			return false
		}
	}
	return true
}

func generationRanking(g GenerationAnalysis, amount Measurement) RankedObservation {
	return RankedObservation{StepIndex: g.StepIndex, HasStepIndex: g.HasStepIndex, GenerationID: g.ID, Label: g.Model, Amount: amount, Evidence: g.Evidence}
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
		}{{"TOOL OUTPUTS (LOCAL TOKEN ESTIMATES)", report.ToolOutputs}, {"CONTEXT GROWTH (OBSERVED DIFFERENCES)", report.ContextGrowth}, {"TOKEN CONSUMERS (AVAILABLE USAGE)", report.TokenConsumers}, {"COST CONSUMERS (REFERENCE USD)", report.CostConsumers}}
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

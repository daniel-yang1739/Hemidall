package antigravity

import (
	"testing"
)

func TestCleanModelNameAndExtraction(t *testing.T) {
	sampleLine := "The user changed setting `Model Selection` from None to Gemini 3.7 Flash (High). No need to comment on this change if the user doesn't ask about it."
	matches := modelSelectRegex.FindStringSubmatch(sampleLine)
	if len(matches) < 2 {
		t.Fatalf("Expected regex to match Model Selection in sample line, got: %v", matches)
	}
	model := CleanModelName(matches[1])
	if model != "Gemini 3.7 Flash (High)" {
		t.Fatalf("Expected 'Gemini 3.7 Flash (High)', got '%s'", model)
	}
}

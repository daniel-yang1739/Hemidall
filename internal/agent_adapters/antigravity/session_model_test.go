package antigravity

import (
	"testing"

	"heimdall/internal/agent_adapters"
)

func TestBuildSession_CreatesDynamicModelTable(t *testing.T) {
	session := buildSession(agents.SessionRef{SessionID: fixtureSessionID}, nil, []agents.Generation{
		{ID: "generation-a", ModelID: "claude-sonnet-4-6", Evidence: []agents.Evidence{{Level: agents.EvidenceArtifactEquality}}},
		{ID: "generation-b", ModelID: "claude-sonnet-4-6"},
		{ID: "generation-c", ModelID: "gemini-3.7-flash"},
		{ID: "generation-d"},
	}, nil, 0)

	requireEqual(t, len(session.Models.Records), 2)
	requireEqual(t, session.Models.Records[0].ModelID, "claude-sonnet-4-6")
	requireEqual(t, len(session.Models.Records[0].GenerationIDs), 2)
	requireEqual(t, session.Models.Records[1].ModelID, "gemini-3.7-flash")
}

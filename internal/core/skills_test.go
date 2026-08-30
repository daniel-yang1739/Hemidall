package core

import (
	"path/filepath"
	"testing"
)

func TestDiscoverSkillDefinitions_ReadsExplicitRootsInStableOrder(t *testing.T) {
	fixtureRoot := filepath.Join("testdata", "skills")
	skills := DiscoverSkillDefinitions(fixtureRoot, filepath.Join("testdata", "missing"))

	if len(skills) != 2 {
		t.Fatalf("expected two fixture skills, got %d", len(skills))
	}
	if skills[0].Name != "alpha" || skills[0].Description != "Alpha fixture skill" {
		t.Errorf("unexpected first skill: %#v", skills[0])
	}
	if skills[1].Name != "zeta-skill" || skills[1].Description != "Zeta fixture skill" {
		t.Errorf("unexpected second skill: %#v", skills[1])
	}
}

func TestFindNearestDirectory_ResolvesAncestorRoot(t *testing.T) {
	start := filepath.Join("testdata", "skills", "alpha")
	resolved := FindNearestDirectory(start, "testdata")
	expected := "testdata"
	if resolved != expected {
		t.Errorf("expected %q, got %q", expected, resolved)
	}
}

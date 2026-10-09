package ui

import (
	"strings"
	"testing"
)

func TestInsightsGuideIsEmbeddedAndSearchableInBothLocales(t *testing.T) {
	for _, language := range []string{"en", "zh"} {
		t.Run(language, func(t *testing.T) {
			items := loadDocDefinitions(language)
			expectedCategory := "Insights - Reading Your Session"
			if language == "zh" {
				expectedCategory = "Insights 使用指南"
			}
			if len(items) == 0 || items[0].Category != expectedCategory {
				t.Fatal("Insights guide is not available at the start of Docs")
			}
			guide := parsedGuideText(items, language)
			expectations := []string{"Task usage", "Task output", "Task thinking", "Tool outputs", "STEPS", "Compaction", "RELATIVE", "5,000", "2,500", "Partial task", ":report", "Privacy"}
			if language == "zh" {
				expectations = []string{"Insights 使用指南", "Task usage", "Task output", "Task thinking", "Tool outputs", "STEPS", "Compaction", "5,000", "2,500", "Partial task", ":report", "分享前請先檢查"}
			}
			for _, expected := range expectations {
				t.Run(expected, func(t *testing.T) {
					if !strings.Contains(guide, expected) {
						t.Fatalf("guide parsing or search lost %q", expected)
					}
				})
			}
			if language == "zh" && strings.Contains(guide, "Start here") {
				t.Fatal("English Insights guide leaked into Traditional Chinese Docs")
			}
			if (DocItem{Category: "Unrelated", Title: "Other topic"}).matchesQuery("insights") {
				t.Fatal("Insights search includes unrelated topics")
			}
		})
	}
}

func parsedGuideText(items []DocItem, language string) string {
	query := "insights"
	if language == "zh" {
		query = "使用指南"
	}
	var text []string
	for _, item := range items {
		if item.matchesQuery(query) {
			text = append(text, item.Category, item.Title, item.Desc)
			for _, bullet := range item.Bullets {
				text = append(text, bullet.Key, bullet.Text)
			}
		}
	}
	return strings.Join(strings.Fields(strings.Join(text, " ")), " ")
}

package ui_test

import (
	"testing"

	"heimdall/internal/ui"
)

func TestStripAnsi(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: ""},
		{name: "plain", input: "Plain text with no ANSI escape sequences 12345!?", want: "Plain text with no ANSI escape sequences 12345!?"},
		{name: "basic colors and bold", input: "\x1b[31;1mRed Bold\x1b[0m and \x1b[32mGreen\x1b[0m", want: "Red Bold and Green"},
		{name: "256 color and true color", input: "\x1b[38;5;196m256Color\x1b[0m \x1b[48;2;255;0;128mRGBBackground\x1b[0m", want: "256Color RGBBackground"},
		{name: "OSC sequence", input: "\x1b]52;c;dGVzdA==\x07Hello OSC", want: "Hello OSC"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ui.StripAnsi(test.input); got != test.want {
				t.Errorf("StripAnsi(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

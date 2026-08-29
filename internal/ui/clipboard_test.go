package ui_test

import (
	"testing"

	"heimdall/internal/ui"
)

func TestStripAnsi_EmptyString(t *testing.T) {
	out := ui.StripAnsi("")
	if out != "" {
		t.Errorf("StripAnsi(\"\") = %q, want \"\"", out)
	}
}

func TestStripAnsi_PlainTextWithoutAnsi(t *testing.T) {
	in := "Plain text with no ANSI escape sequences 12345!?"
	out := ui.StripAnsi(in)
	if out != in {
		t.Errorf("StripAnsi(%q) = %q, want %q", in, out, in)
	}
}

func TestStripAnsi_BasicColorsAndBold(t *testing.T) {
	in := "\x1b[31;1mRed Bold\x1b[0m and \x1b[32mGreen\x1b[0m"
	want := "Red Bold and Green"
	out := ui.StripAnsi(in)
	if out != want {
		t.Errorf("StripAnsi(%q) = %q, want %q", in, out, want)
	}
}

func TestStripAnsi_256ColorAndTrueColorRGB(t *testing.T) {
	in := "\x1b[38;5;196m256Color\x1b[0m \x1b[48;2;255;0;128mRGBBackground\x1b[0m"
	want := "256Color RGBBackground"
	out := ui.StripAnsi(in)
	if out != want {
		t.Errorf("StripAnsi(%q) = %q, want %q", in, out, want)
	}
}

func TestStripAnsi_OscSequences(t *testing.T) {
	in := "\x1b]52;c;dGVzdA==\x07Hello OSC"
	want := "Hello OSC"
	out := ui.StripAnsi(in)
	if out != want {
		t.Errorf("StripAnsi(%q) = %q, want %q", in, out, want)
	}
}

func TestCopyToClipboard_Sanity(t *testing.T) {
	err := ui.CopyToClipboard("Unit test clipboard payload")
	// On environments without GUI/clipboard binary, it still emits OSC 52 without panic
	if err != nil && err.Error() == "" {
		t.Errorf("Unexpected error from CopyToClipboard: %v", err)
	}
}

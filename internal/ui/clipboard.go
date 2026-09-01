package ui

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

const (
	// OS identifier constants from runtime.GOOS
	osDarwin  = "darwin"
	osLinux   = "linux"
	osWindows = "windows"

	// macOS clipboard binaries
	binDarwinPbcopy     = "pbcopy"
	binDarwinPbcopyPath = "/usr/bin/pbcopy"

	// Linux clipboard binaries & arguments
	binLinuxWlCopy = "wl-copy"
	binLinuxXclip  = "xclip"
	binLinuxXsel   = "xsel"
	flagSelection  = "-selection"
	flagClipboard  = "clipboard"
	flagXselClip   = "--clipboard"
	flagXselIn     = "--input"

	// Windows clipboard binaries
	binWindowsClipExe = "clip.exe"
	binWindowsClip    = "clip"

	// ANSI OSC 52 sequence template
	osc52ClipboardFormat = "\x1b]52;c;%s\x07"
)

var (
	ansiRegex                = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\].*?\x07`)
	osStderrWriter io.Writer = os.Stderr
)

// StripAnsi removes ANSI escape sequences from strings
func StripAnsi(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// CopyToClipboard copies text to the OS clipboard using pbcopy, xclip, wl-copy, and OSC 52
func CopyToClipboard(text string) error {
	cleanText := StripAnsi(text)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case osDarwin:
		if path, err := exec.LookPath(binDarwinPbcopy); err == nil {
			cmd = exec.Command(path)
		} else {
			cmd = exec.Command(binDarwinPbcopyPath)
		}
	case osLinux:
		if path, err := exec.LookPath(binLinuxWlCopy); err == nil {
			cmd = exec.Command(path)
		} else if path, err := exec.LookPath(binLinuxXclip); err == nil {
			cmd = exec.Command(path, flagSelection, flagClipboard)
		} else if path, err := exec.LookPath(binLinuxXsel); err == nil {
			cmd = exec.Command(path, flagXselClip, flagXselIn)
		}
	case osWindows:
		if path, err := exec.LookPath(binWindowsClipExe); err == nil {
			cmd = exec.Command(path)
		} else {
			cmd = exec.Command(binWindowsClip)
		}
	}

	var err error
	if cmd != nil {
		cmd.Stdin = strings.NewReader(cleanText)
		err = cmd.Run()
	}

	// Emit ANSI OSC 52 sequence to stderr in interactive terminal sessions (silenced during unit tests)
	if flag.Lookup("test.v") == nil && osStderrWriter != nil {
		b64 := base64.StdEncoding.EncodeToString([]byte(cleanText))
		_, _ = fmt.Fprintf(osStderrWriter, osc52ClipboardFormat, b64)
	}

	return err
}

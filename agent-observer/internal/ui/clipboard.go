package ui

import (
	"os/exec"
	"runtime"
	"strings"
)

// CopyToClipboard writes the specified text to the OS system clipboard
func CopyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command("wl-copy")
		} else {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		}
	case "windows":
		cmd = exec.Command("clip")
	default:
		cmd = exec.Command("pbcopy")
	}

	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

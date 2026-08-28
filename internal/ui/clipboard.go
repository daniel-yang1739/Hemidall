package ui

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// CopyToClipboard copies text to the OS clipboard using pbcopy, xclip, wl-copy, and OSC 52
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
	}

	if cmd != nil {
		cmd.Stdin = strings.NewReader(text)
		_ = cmd.Run()
	}

	// Also emit ANSI OSC 52 sequence to stdout (works over SSH and modern terminals)
	b64 := base64.StdEncoding.EncodeToString([]byte(text))
	fmt.Printf("\x1b]52;c;%s\x07", b64)

	return nil
}

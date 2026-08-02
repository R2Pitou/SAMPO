//go:build !windows

package host

import (
	"fmt"
	"os/exec"
	"runtime"
)

func OpenBrowser(url string) error {
	command := "xdg-open"
	if runtime.GOOS == "darwin" {
		command = "open"
	}
	if err := exec.Command(command, url).Start(); err != nil {
		return fmt.Errorf("open default browser: %w", err)
	}
	return nil
}

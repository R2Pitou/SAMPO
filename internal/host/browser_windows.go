//go:build windows

package host

import (
	"fmt"

	"golang.org/x/sys/windows"
)

func OpenBrowser(url string) error {
	verb, _ := windows.UTF16PtrFromString("open")
	file, err := windows.UTF16PtrFromString(url)
	if err != nil {
		return fmt.Errorf("encode browser URL: %w", err)
	}
	if err := windows.ShellExecute(0, verb, file, nil, nil, windows.SW_SHOWNORMAL); err != nil {
		return fmt.Errorf("open default browser: %w", err)
	}
	return nil
}

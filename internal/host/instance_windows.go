//go:build windows

package host

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows"
)

type InstanceLock struct{ handle windows.Handle }

func AcquireInstance() (*InstanceLock, error) {
	name, _ := windows.UTF16PtrFromString(`Local\sampo-control-plane`)
	handle, err := windows.CreateMutex(nil, false, name)
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		_ = windows.CloseHandle(handle)
		return nil, errors.New("SAMPO is already running for this Windows session")
	}
	if err != nil {
		return nil, fmt.Errorf("create single-instance mutex: %w", err)
	}
	return &InstanceLock{handle: handle}, nil
}

func (l *InstanceLock) Close() error { return windows.CloseHandle(l.handle) }

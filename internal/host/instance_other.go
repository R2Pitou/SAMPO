//go:build !windows

package host

type InstanceLock struct{}

func AcquireInstance() (*InstanceLock, error) { return &InstanceLock{}, nil }
func (l *InstanceLock) Close() error          { return nil }

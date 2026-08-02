//go:build !windows

package observer

import "os"

func nativeIdentityForOpenFile(_ *os.File) (string, error) { return "", nil }
func nativeIdentityForPath(_ string) (string, error)       { return "", nil }

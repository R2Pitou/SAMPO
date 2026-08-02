package observer

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

var ErrUnstable = errors.New("file changed while being hashed")

type HashResult struct {
	DigestHex      string
	ByteLength     int64
	ModifiedAt     time.Time
	NativeIdentity string
}

type evidence struct {
	size       int64
	modifiedAt time.Time
	identity   string
}

func HashStable(path string, maxRetries int) (HashResult, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		result, err := hashOnce(path)
		if err == nil {
			return result, nil
		}
		if !errors.Is(err, ErrUnstable) {
			return HashResult{}, err
		}
		lastErr = err
		if attempt < maxRetries {
			time.Sleep(time.Duration(attempt+1) * 20 * time.Millisecond)
		}
	}
	return HashResult{}, lastErr
}

func hashOnce(path string) (HashResult, error) {
	return hashOnceWithHook(path, nil)
}

func hashOnceWithHook(path string, afterEvidence func()) (HashResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return HashResult{}, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	before, err := evidenceForOpenFile(f)
	if err != nil {
		return HashResult{}, err
	}
	if afterEvidence != nil {
		afterEvidence()
	}

	h := sha256.New()
	written, err := io.CopyBuffer(h, f, make([]byte, 1024*1024))
	if err != nil {
		return HashResult{}, fmt.Errorf("read complete content: %w", err)
	}

	after, err := evidenceForOpenFile(f)
	if err != nil {
		return HashResult{}, err
	}
	pathInfo, err := os.Stat(path)
	if err != nil {
		return HashResult{}, fmt.Errorf("restat path: %w", err)
	}
	pathIdentity, err := nativeIdentityForPath(path)
	if err != nil {
		return HashResult{}, fmt.Errorf("restat identity: %w", err)
	}

	if written != before.size || !sameEvidence(before, after) ||
		pathInfo.Size() != after.size || !pathInfo.ModTime().Equal(after.modifiedAt) ||
		(before.identity != "" && pathIdentity != before.identity) {
		return HashResult{}, ErrUnstable
	}

	return HashResult{
		DigestHex:      hex.EncodeToString(h.Sum(nil)),
		ByteLength:     written,
		ModifiedAt:     after.modifiedAt,
		NativeIdentity: after.identity,
	}, nil
}

func evidenceForOpenFile(f *os.File) (evidence, error) {
	info, err := f.Stat()
	if err != nil {
		return evidence{}, fmt.Errorf("stat open file: %w", err)
	}
	identity, err := nativeIdentityForOpenFile(f)
	if err != nil {
		return evidence{}, fmt.Errorf("read native identity: %w", err)
	}
	return evidence{size: info.Size(), modifiedAt: info.ModTime(), identity: identity}, nil
}

func sameEvidence(a, b evidence) bool {
	if a.size != b.size || !a.modifiedAt.Equal(b.modifiedAt) {
		return false
	}
	return a.identity == "" || b.identity == "" || a.identity == b.identity
}

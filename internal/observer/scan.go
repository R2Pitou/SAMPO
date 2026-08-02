package observer

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"sampo/internal/domain"
)

type Scanner struct {
	HashRetries int
}

func (s Scanner) Scan(ctx context.Context, root string) (domain.ProviderScan, error) {
	started := time.Now().UTC()
	root, err := filepath.Abs(root)
	if err != nil {
		return domain.ProviderScan{}, fmt.Errorf("absolute provider root: %w", err)
	}
	info, err := filepath.EvalSymlinks(root)
	if err != nil {
		return domain.ProviderScan{}, fmt.Errorf("resolve provider root: %w", err)
	}
	root = info

	result := domain.ProviderScan{StartedAt: started}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			result.Issues = append(result.Issues, issue(root, path, "walk", walkErr))
			if entry != nil && entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if path == root {
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if entry.IsDir() || !entry.Type().IsRegular() {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			result.Issues = append(result.Issues, issue(root, path, "locator", errors.New("path escaped provider root")))
			return nil
		}
		hashed, err := HashStable(path, s.HashRetries)
		if err != nil {
			code := "hash"
			if errors.Is(err, ErrUnstable) {
				code = "unstable"
				result.Unstable++
			}
			result.Issues = append(result.Issues, issue(root, path, code, err))
			return nil
		}
		result.Observations = append(result.Observations, domain.FileObservation{
			Locator:        filepath.ToSlash(rel),
			DisplayName:    entry.Name(),
			DigestHex:      hashed.DigestHex,
			ByteLength:     hashed.ByteLength,
			ModifiedAt:     hashed.ModifiedAt,
			NativeIdentity: hashed.NativeIdentity,
		})
		return nil
	})
	result.EndedAt = time.Now().UTC()
	if err != nil {
		return result, fmt.Errorf("scan provider: %w", err)
	}
	return result, nil
}

func issue(root, path, code string, err error) domain.ScanIssue {
	rel, relErr := filepath.Rel(root, path)
	if relErr != nil {
		rel = path
	}
	return domain.ScanIssue{Locator: filepath.ToSlash(rel), Code: code, Message: err.Error()}
}

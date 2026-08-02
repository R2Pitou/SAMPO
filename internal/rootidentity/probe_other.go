//go:build !windows

package rootidentity

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"sampo/internal/domain"
)

func (SystemProber) Probe(submittedLocator string) (domain.ProviderRoot, error) {
	if !filepath.IsAbs(submittedLocator) {
		return domain.ProviderRoot{}, errors.New("provider root must be an absolute path")
	}
	resolved, err := filepath.EvalSymlinks(submittedLocator)
	if err != nil {
		return domain.ProviderRoot{}, fmt.Errorf("resolve provider root: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return domain.ProviderRoot{}, fmt.Errorf("inspect provider root: %w", err)
	}
	if !info.IsDir() {
		return domain.ProviderRoot{}, errors.New("provider root must be a directory")
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return domain.ProviderRoot{}, fmt.Errorf("resolve provider root: %w", err)
	}
	return domain.ProviderRoot{
		SubmittedLocator:   submittedLocator,
		OperationalLocator: resolved,
		FinalPathEvidence:  resolved,
		IdentityConfidence: domain.RootIdentityWeak,
		CatalogueOnly:      true,
	}, nil
}

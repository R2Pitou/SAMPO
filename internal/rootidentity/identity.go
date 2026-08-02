package rootidentity

import (
	"errors"
	"fmt"
	"strings"

	"sampo/internal/domain"
)

var ErrRootIdentityChanged = errors.New("provider root identity changed")

type Prober interface {
	Probe(submittedLocator string) (domain.ProviderRoot, error)
}

type SystemProber struct{}

func Verify(enrolled domain.ProviderRoot, current domain.ProviderRoot) error {
	if enrolled.IdentityConfidence == domain.RootIdentityWeak || current.IdentityConfidence == domain.RootIdentityWeak {
		if enrolled.IdentityConfidence == domain.RootIdentityWeak &&
			current.IdentityConfidence == domain.RootIdentityWeak &&
			equalPathEvidence(enrolled.FinalPathEvidence, current.FinalPathEvidence) {
			return nil
		}
		return fmt.Errorf("%w: enrolled=%s observed=%s", ErrRootIdentityChanged,
			describe(enrolled), describe(current))
	}
	if enrolled.PhysicalIdentity != "" && current.PhysicalIdentity == enrolled.PhysicalIdentity {
		return nil
	}
	if enrolled.FallbackIdentity != "" && current.FallbackIdentity == enrolled.FallbackIdentity {
		return nil
	}
	return fmt.Errorf("%w: enrolled=%s observed=%s", ErrRootIdentityChanged,
		describe(enrolled), describe(current))
}

func equalPathEvidence(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if windowsPath(a) || windowsPath(b) {
		return strings.EqualFold(strings.TrimRight(a, `\/`), strings.TrimRight(b, `\/`))
	}
	return strings.TrimRight(a, "/") == strings.TrimRight(b, "/")
}

func windowsPath(path string) bool {
	return strings.HasPrefix(path, `\\`) || (len(path) >= 2 && path[1] == ':')
}

func describe(root domain.ProviderRoot) string {
	if root.PhysicalIdentity != "" {
		return root.PhysicalIdentity
	}
	if root.FallbackIdentity != "" {
		return root.FallbackIdentity
	}
	if root.FinalPathEvidence != "" {
		return root.IdentityConfidence + ":" + root.FinalPathEvidence
	}
	return root.IdentityConfidence + ":unknown"
}

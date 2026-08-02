package rootidentity

import (
	"errors"
	"testing"

	"sampo/internal/domain"
)

func TestVerifyUsesPhysicalEvidenceAndFailsClosedOnMismatch(t *testing.T) {
	enrolled := domain.ProviderRoot{
		PhysicalIdentity: "physical-one", FallbackIdentity: "fallback-one",
		IdentityConfidence: domain.RootIdentityStrong,
	}
	matching := domain.ProviderRoot{
		PhysicalIdentity: "physical-one", FallbackIdentity: "fallback-one",
		IdentityConfidence: domain.RootIdentityStrong,
	}
	if err := Verify(enrolled, matching); err != nil {
		t.Fatal(err)
	}
	replacement := matching
	replacement.PhysicalIdentity = "physical-two"
	replacement.FallbackIdentity = "fallback-two"
	if err := Verify(enrolled, replacement); !errors.Is(err, ErrRootIdentityChanged) {
		t.Fatalf("replacement verification error = %v", err)
	}
}

func TestVerifyAllowsOnlyMatchingWeakPathEvidence(t *testing.T) {
	enrolled := domain.ProviderRoot{
		FinalPathEvidence: `\\server\share\Root`, IdentityConfidence: domain.RootIdentityWeak,
	}
	matching := domain.ProviderRoot{
		FinalPathEvidence: `\\SERVER\SHARE\root\`, PhysicalIdentity: "unreliable-zero",
		FallbackIdentity: "unreliable-zero", IdentityConfidence: domain.RootIdentityWeak,
	}
	if err := Verify(enrolled, matching); err != nil {
		t.Fatal(err)
	}
	matching.FinalPathEvidence = `\\server\share\other`
	if err := Verify(enrolled, matching); !errors.Is(err, ErrRootIdentityChanged) {
		t.Fatalf("redirected weak root verification error = %v", err)
	}
}

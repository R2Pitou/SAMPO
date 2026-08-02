package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"sampo/internal/catalog"
	"sampo/internal/domain"
)

func TestMilestoneOneJourneyLeavesSourceUntouched(t *testing.T) {
	ctx := context.Background()
	providerParent := t.TempDir()
	providerRoot := filepath.Join(providerParent, "provider")
	if err := os.Mkdir(providerRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte("five idiot copies")
	for _, name := range []string{"one.txt", "two.txt", "three.txt", "four.txt", "five.txt"} {
		if err := os.WriteFile(filepath.Join(providerRoot, name), content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	store, err := catalog.Open(ctx, filepath.Join(t.TempDir(), "home", "catalogue.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := New(store)

	provider, err := service.EnrollFilesystem(ctx, "Source", providerRoot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Scan(ctx, provider.ID); err != nil {
		t.Fatal(err)
	}
	results, err := service.Search(ctx, "one.txt", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || len(results[0].Appearances) != 5 {
		t.Fatalf("duplicate grouping = %#v", results)
	}
	want := sha256.Sum256(content)
	if results[0].Content.DigestHex != hex.EncodeToString(want[:]) {
		t.Fatal("catalogue did not use the complete SHA-256 digest")
	}
	if _, err := os.Stat(filepath.Join(providerRoot, ".sampo")); !os.IsNotExist(err) {
		t.Fatalf("enrollment or scan modified provider: %v", err)
	}
	for _, name := range []string{"one.txt", "two.txt", "three.txt", "four.txt", "five.txt"} {
		got, err := os.ReadFile(filepath.Join(providerRoot, name))
		if err != nil || string(got) != string(content) {
			t.Fatalf("source %s changed: %q, %v", name, got, err)
		}
	}

	disconnected := filepath.Join(providerParent, "disconnected")
	if err := os.Rename(providerRoot, disconnected); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Scan(ctx, provider.ID); err == nil {
		t.Fatal("scan unexpectedly succeeded while provider was unavailable")
	}
	unavailable, err := service.Search(ctx, "one.txt", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(unavailable) != 1 || len(unavailable[0].Appearances) != 5 {
		t.Fatalf("unavailable provider lost catalogue history: %#v", unavailable)
	}
	for _, appearance := range unavailable[0].Appearances {
		if appearance.Availability != "unavailable" {
			t.Fatalf("appearance remained %q while provider was unavailable", appearance.Availability)
		}
	}
	if err := os.Rename(disconnected, providerRoot); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Scan(ctx, provider.ID); err != nil {
		t.Fatal(err)
	}
	reconnected, err := service.Search(ctx, "one.txt", 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, appearance := range reconnected[0].Appearances {
		if appearance.Availability != "available" {
			t.Fatalf("appearance remained %q after successful reconnect scan", appearance.Availability)
		}
	}
}

type fixedRootProber struct {
	root domain.ProviderRoot
	err  error
}

func (p fixedRootProber) Probe(submitted string) (domain.ProviderRoot, error) {
	root := p.root
	root.SubmittedLocator = submitted
	return root, p.err
}

func TestWeakProviderIdentityIsCatalogueOnly(t *testing.T) {
	ctx := context.Background()
	store, err := catalog.Open(ctx, filepath.Join(t.TempDir(), "catalogue.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := newWithRootProber(store, fixedRootProber{root: domain.ProviderRoot{
		OperationalLocator: `\\server\share\root`,
		FinalPathEvidence:  `\\server\share\root`,
		IdentityConfidence: domain.RootIdentityWeak,
		CatalogueOnly:      true,
	}})
	provider, err := service.EnrollFilesystem(ctx, "Weak remote", `\\server\share\root`)
	if err != nil {
		t.Fatal(err)
	}
	if !provider.CatalogueOnly || provider.IdentityConfidence != domain.RootIdentityWeak {
		t.Fatalf("weak provider = %#v", provider.ProviderRoot)
	}
}

type sequenceRootProber struct {
	mu    sync.Mutex
	roots []domain.ProviderRoot
	next  int
}

func (p *sequenceRootProber) Probe(submitted string) (domain.ProviderRoot, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	index := p.next
	if index >= len(p.roots) {
		index = len(p.roots) - 1
	}
	p.next++
	root := p.roots[index]
	root.SubmittedLocator = submitted
	return root, nil
}

func TestScanRejectsRootIdentityChangeDuringEnumeration(t *testing.T) {
	ctx := context.Background()
	rootPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(rootPath, "untrusted.txt"), []byte("bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	base := domain.ProviderRoot{
		OperationalLocator: rootPath, FinalPathEvidence: rootPath,
		PhysicalIdentity: "physical-before", FallbackIdentity: "fallback-before",
		IdentityConfidence: domain.RootIdentityStrong,
	}
	after := base
	after.PhysicalIdentity = "physical-after"
	after.FallbackIdentity = "fallback-after"
	prober := &sequenceRootProber{roots: []domain.ProviderRoot{base, base, after}}
	store, err := catalog.Open(ctx, filepath.Join(t.TempDir(), "catalogue.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := newWithRootProber(store, prober)
	provider, err := service.EnrollFilesystem(ctx, "Swap test", rootPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Scan(ctx, provider.ID); err == nil {
		t.Fatal("scan accepted a root identity change during enumeration")
	}
	results, err := service.Search(ctx, "untrusted.txt", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("mid-scan replacement observations were reconciled: %#v", results)
	}
}

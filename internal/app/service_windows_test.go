//go:build windows

package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"sampo/internal/catalog"
	"sampo/internal/seshat"
)

func TestWindowsEnrollmentRejectsAliasesAndOverlappingRoots(t *testing.T) {
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "provider")
	child := filepath.Join(root, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	store := openServiceTestStore(t)
	service := New(store)
	if _, err := service.EnrollFilesystem(ctx, "Root", root); err != nil {
		t.Fatal(err)
	}
	if _, err := service.EnrollFilesystem(ctx, "Extended alias", `\\?\`+root); !errors.Is(err, seshat.ErrProviderRootDuplicate) {
		t.Fatalf("extended alias error = %v, want duplicate", err)
	}
	if _, err := service.EnrollFilesystem(ctx, "Child", child); !errors.Is(err, seshat.ErrProviderRootOverlap) {
		t.Fatalf("child error = %v, want overlap", err)
	}
}

func TestWindowsConcurrentAliasEnrollmentCommitsOnce(t *testing.T) {
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "provider")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	store := openServiceTestStore(t)
	service := New(store)
	paths := []string{root, `\\?\` + root}
	var wg sync.WaitGroup
	errs := make(chan error, len(paths))
	for _, path := range paths {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			_, err := service.EnrollFilesystem(ctx, "Concurrent", path)
			errs <- err
		}(path)
	}
	wg.Wait()
	close(errs)
	var succeeded, rejected int
	for err := range errs {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, seshat.ErrProviderRootDuplicate):
			rejected++
		default:
			t.Fatalf("unexpected enrollment result: %v", err)
		}
	}
	if succeeded != 1 || rejected != 1 {
		t.Fatalf("succeeded=%d rejected=%d, want 1/1", succeeded, rejected)
	}
}

func TestWindowsScanRefusesReplacedProviderRoot(t *testing.T) {
	ctx := context.Background()
	parent := t.TempDir()
	root := filepath.Join(parent, "provider")
	displaced := filepath.Join(parent, "original-provider")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "original.txt"), []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := openServiceTestStore(t)
	service := New(store)
	provider, err := service.EnrollFilesystem(ctx, "Replace test", root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Scan(ctx, provider.ID); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(root, displaced); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "replacement.txt"), []byte("replacement"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Scan(ctx, provider.ID); err == nil || !strings.Contains(err.Error(), "provider root identity changed") {
		t.Fatalf("replacement scan error = %v", err)
	}
	original, err := service.Search(ctx, "original.txt", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(original) != 1 || original[0].Appearances[0].Availability != "unavailable" {
		t.Fatalf("original appearance after replacement = %#v", original)
	}
	replacement, err := service.Search(ctx, "replacement.txt", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(replacement) != 0 {
		t.Fatalf("replacement root bytes were indexed: %#v", replacement)
	}
}

func TestWindowsScanRefusesRedirectedProviderRoot(t *testing.T) {
	ctx := context.Background()
	parent := t.TempDir()
	first := filepath.Join(parent, "first")
	second := filepath.Join(parent, "second")
	link := filepath.Join(parent, "provider-link")
	if err := os.Mkdir(first, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(second, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(first, link); err != nil {
		t.Skipf("symbolic-link privilege unavailable: %v", err)
	}
	store := openServiceTestStore(t)
	service := New(store)
	provider, err := service.EnrollFilesystem(ctx, "Redirect test", link)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(second, link); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Scan(ctx, provider.ID); err == nil || !strings.Contains(err.Error(), "provider root identity changed") {
		t.Fatalf("redirected scan error = %v", err)
	}
}

func openServiceTestStore(t *testing.T) *catalog.Store {
	t.Helper()
	store, err := catalog.Open(context.Background(), filepath.Join(t.TempDir(), "catalogue.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

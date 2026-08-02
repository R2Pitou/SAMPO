package catalog

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"sampo/internal/domain"
)

func TestOpenUsesDurableRollbackConfiguration(t *testing.T) {
	store := openTestStore(t)
	checks := []struct {
		query string
		want  string
	}{
		{"PRAGMA journal_mode", "delete"},
		{"PRAGMA synchronous", "2"},
		{"PRAGMA foreign_keys", "1"},
		{"PRAGMA user_version", "1"},
		{"SELECT sqlite_version()", "3.53.3"},
	}
	for _, check := range checks {
		var got string
		if err := store.db.QueryRow(check.query).Scan(&got); err != nil {
			t.Fatalf("%s: %v", check.query, err)
		}
		if got != check.want {
			t.Fatalf("%s = %q, want %q", check.query, got, check.want)
		}
	}
}

func TestOpenRejectsCorruptCatalogue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catalogue.sqlite3")
	if err := os.WriteFile(path, []byte("not a sqlite database"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := Open(context.Background(), path)
	if err == nil {
		store.Close()
		t.Fatal("corrupt catalogue was accepted")
	}
}

func TestReconcileGroupsDuplicatesAndPreservesRenameContinuity(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	provider, err := store.AddFilesystemProvider(ctx, "Test", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	first := testScan(
		observation("A/one.txt", "aaa", 3, "file-1"),
		observation("B/two.txt", "aaa", 3, "file-2"),
		observation("C/three.txt", "bbb", 9, "file-3"),
	)
	if err := store.ReconcileScan(ctx, provider.ID, first); err != nil {
		t.Fatal(err)
	}
	results, err := store.Search(ctx, "", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d content groups, want 2", len(results))
	}
	duplicate := resultByDigest(t, results, "aaa")
	if len(duplicate.Appearances) != 2 {
		t.Fatalf("got %d duplicate appearances, want 2", len(duplicate.Appearances))
	}
	for _, appearance := range duplicate.Appearances {
		if appearance.Custody != domain.CustodyUserOwned {
			t.Fatalf("custody = %q, want user-owned", appearance.Custody)
		}
	}
	originalID := appearanceByLocator(t, duplicate.Appearances, "A/one.txt").ID

	second := testScan(
		observation("Moved/one.txt", "aaa", 3, "file-1"),
		observation("B/two.txt", "aaa", 3, "file-2"),
		observation("C/three.txt", "bbb", 9, "file-3"),
	)
	if err := store.ReconcileScan(ctx, provider.ID, second); err != nil {
		t.Fatal(err)
	}
	renamed, err := store.Search(ctx, "Moved", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(renamed) != 1 || len(renamed[0].Appearances) != 2 {
		t.Fatalf("unexpected renamed result: %#v", renamed)
	}
	renamedAppearance := appearanceByLocator(t, renamed[0].Appearances, "Moved/one.txt")
	if renamedAppearance.ID != originalID {
		t.Fatal("native-ID rename did not preserve Appearance identity")
	}
	if renamedAppearance.Continuity != "confirmed-rename" {
		t.Fatalf("continuity = %q, want confirmed-rename", renamedAppearance.Continuity)
	}
}

func TestReconcileDoesNotReuseAppearanceForChangedBytes(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	provider, err := store.AddFilesystemProvider(ctx, "Test", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ReconcileScan(ctx, provider.ID, testScan(observation("work.txt", "old", 3, "same-file"))); err != nil {
		t.Fatal(err)
	}
	before, err := store.Search(ctx, "work.txt", 100)
	if err != nil {
		t.Fatal(err)
	}
	oldID := before[0].Appearances[0].ID

	if err := store.ReconcileScan(ctx, provider.ID, testScan(observation("work.txt", "new", 4, "same-file"))); err != nil {
		t.Fatal(err)
	}
	after, err := store.Search(ctx, "work.txt", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 2 {
		t.Fatalf("got %d content records, want old and new Content", len(after))
	}
	var available, unavailable domain.Appearance
	for _, result := range after {
		for _, appearance := range result.Appearances {
			if appearance.Availability == "available" {
				available = appearance
			} else {
				unavailable = appearance
			}
		}
	}
	if available.ID == "" || unavailable.ID == "" {
		t.Fatalf("expected available and unavailable appearances: %#v", after)
	}
	if available.ID == oldID || unavailable.ID != oldID {
		t.Fatal("changed bytes erased or reused the old Appearance identity")
	}
}

func TestReconcileUsesOneToOneExactHashAsProbableRename(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	provider, err := store.AddFilesystemProvider(ctx, "Test", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ReconcileScan(ctx, provider.ID, testScan(observation("old.bin", "hash", 12, ""))); err != nil {
		t.Fatal(err)
	}
	before, _ := store.Search(ctx, "old.bin", 10)
	oldID := before[0].Appearances[0].ID

	next := testScan(observation("new.bin", "hash", 12, ""))
	next.StartedAt = next.StartedAt.Add(time.Minute)
	next.EndedAt = next.EndedAt.Add(time.Minute)
	if err := store.ReconcileScan(ctx, provider.ID, next); err != nil {
		t.Fatal(err)
	}
	after, _ := store.Search(ctx, "new.bin", 10)
	appearance := after[0].Appearances[0]
	if appearance.ID != oldID || appearance.Continuity != "probable-rename" {
		t.Fatalf("probable rename = %#v, want id %s", appearance, oldID)
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(context.Background(), filepath.Join(t.TempDir(), "catalogue.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func testScan(observations ...domain.FileObservation) domain.ProviderScan {
	now := time.Now().UTC()
	return domain.ProviderScan{StartedAt: now, EndedAt: now.Add(time.Second), Observations: observations}
}

func observation(locator, digest string, size int64, identity string) domain.FileObservation {
	return domain.FileObservation{
		Locator: locator, DisplayName: filepath.Base(locator), DigestHex: digest,
		ByteLength: size, ModifiedAt: time.Now().UTC(), NativeIdentity: identity,
	}
}

func resultByDigest(t *testing.T, results []domain.SearchResult, digest string) domain.SearchResult {
	t.Helper()
	for _, result := range results {
		if result.Content.DigestHex == digest {
			return result
		}
	}
	t.Fatalf("digest %q not found", digest)
	return domain.SearchResult{}
}

func appearanceByLocator(t *testing.T, appearances []domain.Appearance, locator string) domain.Appearance {
	t.Helper()
	for _, appearance := range appearances {
		if appearance.Locator == locator {
			return appearance
		}
	}
	t.Fatalf("locator %q not found", locator)
	return domain.Appearance{}
}

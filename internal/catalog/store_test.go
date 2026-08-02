package catalog

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sync"
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
		{"PRAGMA user_version", "2"},
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

func TestProviderRootEvidencePersistsSeparatelyAndExactly(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	root := domain.ProviderRoot{
		SubmittedLocator:   `\\?\C:\Submitted Provider Root `,
		OperationalLocator: `\\?\C:\Operational Provider Root`,
		FinalPathEvidence:  `\\?\Volume{test}\Final Provider Root`,
		PhysicalIdentity:   "windows:file-id128:volume:file",
		FallbackIdentity:   "windows:file-index64:volume:file",
		IdentityConfidence: domain.RootIdentityStrong,
	}
	created, err := store.AddFilesystemProvider(ctx, "Evidence", root)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Provider(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ProviderRoot != root {
		t.Fatalf("persisted root evidence = %#v, want %#v", loaded.ProviderRoot, root)
	}
	if loaded.RootLocator != root.SubmittedLocator {
		t.Fatalf("compatibility root locator = %q, want submitted locator %q", loaded.RootLocator, root.SubmittedLocator)
	}
}

func TestReconcileGroupsDuplicatesAndPreservesRenameContinuity(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	provider, err := store.AddFilesystemProvider(ctx, "Test", testProviderRoot(t.TempDir()))
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
	provider, err := store.AddFilesystemProvider(ctx, "Test", testProviderRoot(t.TempDir()))
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
	provider, err := store.AddFilesystemProvider(ctx, "Test", testProviderRoot(t.TempDir()))
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

func TestProviderEnrollmentRejectsPhysicalDuplicateAndOverlap(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	first := domain.ProviderRoot{
		SubmittedLocator: `C:\Data`, OperationalLocator: `\\?\C:\Data`,
		FinalPathEvidence: `\\?\Volume{test}\Data`, PhysicalIdentity: "physical-root",
		FallbackIdentity: "fallback-root", IdentityConfidence: domain.RootIdentityStrong,
	}
	if _, err := store.AddFilesystemProvider(ctx, "First", first); err != nil {
		t.Fatal(err)
	}

	alias := first
	alias.SubmittedLocator = `\\?\C:\Data`
	alias.OperationalLocator = alias.SubmittedLocator
	if _, err := store.AddFilesystemProvider(ctx, "Alias", alias); !errors.Is(err, ErrProviderRootDuplicate) {
		t.Fatalf("physical alias error = %v, want duplicate", err)
	}

	child := domain.ProviderRoot{
		SubmittedLocator: `C:\Data\Child`, OperationalLocator: `\\?\C:\Data\Child`,
		FinalPathEvidence: `\\?\Volume{test}\Data\Child`, PhysicalIdentity: "physical-child",
		FallbackIdentity: "fallback-child", IdentityConfidence: domain.RootIdentityStrong,
	}
	if _, err := store.AddFilesystemProvider(ctx, "Child", child); !errors.Is(err, ErrProviderRootOverlap) {
		t.Fatalf("child overlap error = %v, want overlap", err)
	}

	parent := domain.ProviderRoot{
		SubmittedLocator: `C:\`, OperationalLocator: `\\?\C:\`,
		FinalPathEvidence: `\\?\Volume{test}\`, PhysicalIdentity: "physical-parent",
		FallbackIdentity: "fallback-parent", IdentityConfidence: domain.RootIdentityStrong,
	}
	if _, err := store.AddFilesystemProvider(ctx, "Parent", parent); !errors.Is(err, ErrProviderRootOverlap) {
		t.Fatalf("parent overlap error = %v, want overlap", err)
	}
}

func TestProviderEnrollmentSerializesConcurrentPhysicalAliases(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	roots := []domain.ProviderRoot{
		{
			SubmittedLocator: `C:\Concurrent`, OperationalLocator: `\\?\C:\Concurrent`,
			FinalPathEvidence: `\\?\Volume{test}\Concurrent`, PhysicalIdentity: "same-physical",
			FallbackIdentity: "same-fallback", IdentityConfidence: domain.RootIdentityStrong,
		},
		{
			SubmittedLocator: `\\?\C:\Concurrent`, OperationalLocator: `\\?\C:\Concurrent`,
			FinalPathEvidence: `\\?\Volume{test}\Concurrent`, PhysicalIdentity: "same-physical",
			FallbackIdentity: "same-fallback", IdentityConfidence: domain.RootIdentityStrong,
		},
	}
	var wg sync.WaitGroup
	errs := make(chan error, len(roots))
	for i, root := range roots {
		wg.Add(1)
		go func(i int, root domain.ProviderRoot) {
			defer wg.Done()
			_, err := store.AddFilesystemProvider(ctx, "Concurrent", root)
			errs <- err
		}(i, root)
	}
	wg.Wait()
	close(errs)
	var succeeded, rejected int
	for err := range errs {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, ErrProviderRootDuplicate):
			rejected++
		default:
			t.Fatalf("unexpected enrollment error: %v", err)
		}
	}
	if succeeded != 1 || rejected != 1 {
		t.Fatalf("succeeded=%d rejected=%d, want 1/1", succeeded, rejected)
	}
}

func TestWeakRemoteEvidenceDoesNotCreateFalsePhysicalDuplicates(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	for _, locator := range []string{`\\server\share\one`, `\\server\share\two`} {
		root := domain.ProviderRoot{
			SubmittedLocator: locator, OperationalLocator: locator, FinalPathEvidence: locator,
			PhysicalIdentity:   "windows:file-id128:0000000000000000:00000000000000000000000000000000",
			FallbackIdentity:   "windows:file-index64:00000000:0000000000000000",
			IdentityConfidence: domain.RootIdentityWeak, CatalogueOnly: true,
		}
		if _, err := store.AddFilesystemProvider(ctx, "Weak", root); err != nil {
			t.Fatalf("weak root %q was rejected using unreliable physical evidence: %v", locator, err)
		}
	}
}

func TestMigrationMarksExistingProviderIdentityLegacyAndCatalogueOnly(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "catalogue.sqlite3")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(migrationV1); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO providers
        (id, kind, display_name, root_locator, created_at_ns, scan_status)
        VALUES ('legacy', 'filesystem', 'Legacy', 'C:\Legacy', 1, 'never-scanned')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("PRAGMA user_version=1"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	provider, err := store.Provider(ctx, "legacy")
	if err != nil {
		t.Fatal(err)
	}
	if provider.IdentityConfidence != domain.RootIdentityLegacy || !provider.CatalogueOnly {
		t.Fatalf("migrated provider identity = %#v", provider.ProviderRoot)
	}
	if provider.SubmittedLocator != `C:\Legacy` || provider.OperationalLocator != `C:\Legacy` {
		t.Fatalf("migrated locators = %#v", provider.ProviderRoot)
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

func testProviderRoot(path string) domain.ProviderRoot {
	path = filepath.Clean(path)
	return domain.ProviderRoot{
		SubmittedLocator: path, OperationalLocator: path, FinalPathEvidence: path,
		PhysicalIdentity: "test:" + path, FallbackIdentity: "test-fallback:" + path,
		IdentityConfidence: domain.RootIdentityStrong,
	}
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

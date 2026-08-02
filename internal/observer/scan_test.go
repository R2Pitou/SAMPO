package observer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestScanReadsCompleteContentWithoutChangingProvider(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "A", "one.txt"), []byte("same bytes"))
	writeTestFile(t, filepath.Join(root, "B", "two.txt"), []byte("same bytes"))
	writeTestFile(t, filepath.Join(root, "unique.bin"), []byte{0, 1, 2, 3})
	before := snapshotTree(t, root)

	result, err := (Scanner{HashRetries: 2}).Scan(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Issues) != 0 {
		t.Fatalf("unexpected scan issues: %#v", result.Issues)
	}
	if len(result.Observations) != 3 {
		t.Fatalf("observed %d files, want 3", len(result.Observations))
	}
	after := snapshotTree(t, root)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("provider tree changed\nbefore: %#v\nafter:  %#v", before, after)
	}
	if _, err := os.Stat(filepath.Join(root, ".sampo")); !os.IsNotExist(err) {
		t.Fatalf("scanner created provider metadata: %v", err)
	}

	want := sha256.Sum256([]byte("same bytes"))
	var duplicateDigests []string
	for _, observation := range result.Observations {
		if observation.Locator == "A/one.txt" || observation.Locator == "B/two.txt" {
			duplicateDigests = append(duplicateDigests, observation.DigestHex)
		}
	}
	if len(duplicateDigests) != 2 || duplicateDigests[0] != hex.EncodeToString(want[:]) || duplicateDigests[1] != duplicateDigests[0] {
		t.Fatalf("duplicate complete hashes = %#v", duplicateDigests)
	}
}

func TestHashStableReturnsHandleIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.txt")
	writeTestFile(t, path, []byte("content"))
	result, err := HashStable(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	if result.ByteLength != 7 {
		t.Fatalf("length = %d, want 7", result.ByteLength)
	}
	if result.NativeIdentity == "" {
		t.Fatal("Windows scan did not record native file identity")
	}
}

func TestHashRejectsFileChangedAfterInitialEvidence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "changing.txt")
	writeTestFile(t, path, []byte("before"))
	_, err := hashOnceWithHook(path, func() {
		if writeErr := os.WriteFile(path, []byte("different length"), 0o644); writeErr != nil {
			t.Fatal(writeErr)
		}
	})
	if !errors.Is(err, ErrUnstable) {
		t.Fatalf("changed file error = %v, want ErrUnstable", err)
	}
}

func writeTestFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func snapshotTree(t *testing.T, root string) []string {
	t.Helper()
	var snapshot []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			snapshot = append(snapshot, "dir:"+filepath.ToSlash(rel))
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(content)
		snapshot = append(snapshot, "file:"+filepath.ToSlash(rel)+":"+hex.EncodeToString(digest[:]))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(snapshot)
	return snapshot
}

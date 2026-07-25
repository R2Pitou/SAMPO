package staff_test

import (
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mash/internal/config"
	"mash/internal/event"
	"mash/internal/librarian"
	"mash/internal/staff"
)

func TestMASHStaffIntegration(t *testing.T) {
	// 1. Setup temporary directories for providers
	tmpDir := t.TempDir()
	primaryPath := filepath.Join(tmpDir, "primary")
	backupPath := filepath.Join(tmpDir, "backup")

	if err := os.MkdirAll(primaryPath, 0755); err != nil {
		t.Fatalf("failed to create primary dir: %v", err)
	}
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		t.Fatalf("failed to create backup dir: %v", err)
	}

	// 2. Setup Event Bus and Catalog (Seshat)
	bus := event.NewBus()
	catalog := librarian.NewSeshat(bus, nil)

	// Register Storage Providers
	prov1 := &config.StorageProvider{
		ID:   "prov-primary",
		Type: "local",
		Path: primaryPath,
	}
	prov2 := &config.StorageProvider{
		ID:   "prov-backup",
		Type: "local",
		Path: backupPath,
	}
	catalog.AddProvider(prov1)
	catalog.AddProvider(prov2)

	// Define replication policy requiring 2 replicas for all objects
	policies := []config.Policy{
		{
			ID:       "policy-replicate",
			Type:     "replicate",
			Target:   "object",
			Value:    "2",
			Priority: 1,
		},
	}

	// Track published events of interest for verification
	var jobCompletedReceived bool
	var copyCorruptedReceived bool
	var providerOfflineReceived bool

	bus.Subscribe(event.EventJobCompleted, func(e event.Event) {
		log.Printf("[Test] EventJobCompleted received: %+v", e)
		jobCompletedReceived = true
	})

	bus.Subscribe(event.EventCopyCorrupted, func(e event.Event) {
		log.Printf("[Test] EventCopyCorrupted received: %+v", e)
		copyCorruptedReceived = true
	})

	bus.Subscribe(event.EventProviderOffline, func(e event.Event) {
		log.Printf("[Test] EventProviderOffline received: %+v", e)
		providerOfflineReceived = true
	})

	// 3. Initialize Staff Services
	_ = staff.NewTuoni(catalog, bus, policies)
	_ = staff.NewBoatman(catalog, bus)
	observer := staff.NewObserver(catalog, bus)
	caretaker := staff.NewCaretaker(catalog, bus)

	// 4. Create a test file in the Primary Storage Provider
	testFileName := "photo.jpg"
	testFileContent := "dummy image bytes"
	testFilePath := filepath.Join(primaryPath, testFileName)

	if err := os.WriteFile(testFilePath, []byte(testFileContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// 5. Trigger Observer Scan (simulating manual or interval scan)
	observer.ScanProviders()

	// Wait up to 2 seconds for asynchronous event orchestration (Observer -> Tuoni -> Boatman)
	deadline := time.Now().Add(2 * time.Second)
	var obj *librarian.Object
	var err error
	for time.Now().Before(deadline) {
		obj, err = catalog.GetObject(testFileName)
		if err == nil {
			obj.RLock()
			// We expect 2 copies because of the replication policy: primary and backup
			if len(obj.Versions) > 0 && len(obj.Versions[0].Copies) >= 2 {
				obj.RUnlock()
				break
			}
			obj.RUnlock()
		}
		time.Sleep(50 * time.Millisecond)
	}

	if err != nil {
		t.Fatalf("file not discovered in catalog: %v", err)
	}

	obj.RLock()
	if len(obj.Versions) == 0 {
		obj.RUnlock()
		t.Fatalf("no versions found for object")
	}
	copies := obj.Versions[0].Copies
	obj.RUnlock()

	if len(copies) < 2 {
		t.Fatalf("replication failed to meet policy: expected at least 2 copies, got %d", len(copies))
	}

	// Verify physical replication actually happened
	replicatedFilePath := filepath.Join(backupPath, testFileName)
	if _, err := os.Stat(replicatedFilePath); os.IsNotExist(err) {
		t.Fatalf("replicated file does not exist at backup path %s", replicatedFilePath)
	}

	replicatedBytes, err := os.ReadFile(replicatedFilePath)
	if err != nil {
		t.Fatalf("failed to read replicated file: %v", err)
	}
	if string(replicatedBytes) != testFileContent {
		t.Errorf("replicated file content mismatch: expected '%s', got '%s'", testFileContent, string(replicatedBytes))
	}

	if !jobCompletedReceived {
		t.Errorf("expected EventJobCompleted to have been published")
	}

	// 6. Test Caretaker Integrity Check - Healthy State
	caretaker.PerformIntegrityCheck()

	obj, _ = catalog.GetObject(testFileName)
	obj.RLock()
	for _, cp := range obj.Versions[0].Copies {
		if cp.State != "healthy" {
			t.Errorf("expected copy %+v to be healthy, got %s", cp, cp.State)
		}
	}
	obj.RUnlock()

	// 7. Test Caretaker Integrity Check - Corrupted State
	// Let's modify/corrupt the replicated file
	if err := os.WriteFile(replicatedFilePath, []byte("corrupted content bytes"), 0644); err != nil {
		t.Fatalf("failed to corrupt file: %v", err)
	}

	caretaker.PerformIntegrityCheck()

	// Wait briefly for Event Bus asynchronously
	time.Sleep(100 * time.Millisecond)

	obj, _ = catalog.GetObject(testFileName)
	obj.RLock()
	var foundCorrupted bool
	for _, cp := range obj.Versions[0].Copies {
		if cp.ProviderID == "prov-backup" && cp.State == "corrupted" {
			foundCorrupted = true
		}
	}
	obj.RUnlock()

	if !foundCorrupted {
		t.Errorf("expected caretaker to mark the backup copy as corrupted")
	}

	if !copyCorruptedReceived {
		t.Errorf("expected EventCopyCorrupted to have been published")
	}

	// 8. Test Caretaker Integrity Check - Missing/Offline State
	// Let's delete the replicated file entirely
	if err := os.Remove(replicatedFilePath); err != nil {
		t.Fatalf("failed to delete backup file: %v", err)
	}

	caretaker.PerformIntegrityCheck()

	// Wait briefly for Event Bus asynchronously
	time.Sleep(100 * time.Millisecond)

	obj, _ = catalog.GetObject(testFileName)
	obj.RLock()
	var foundMissing bool
	for _, cp := range obj.Versions[0].Copies {
		if cp.ProviderID == "prov-backup" && cp.State == "missing" {
			foundMissing = true
		}
	}
	obj.RUnlock()

	if !foundMissing {
		t.Errorf("expected caretaker to mark the backup copy as missing")
	}

	if !providerOfflineReceived {
		t.Errorf("expected EventProviderOffline to have been published")
	}
}

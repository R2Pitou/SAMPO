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

func TestMASHStaffControlAndTiering(t *testing.T) {
	tmpDir := t.TempDir()
	ssdReadOnlyPath := filepath.Join(tmpDir, "ssd-readonly")
	ssdFullPath := filepath.Join(tmpDir, "ssd-full")
	hddFullPath := filepath.Join(tmpDir, "hdd-full")

	_ = os.MkdirAll(ssdReadOnlyPath, 0755)
	_ = os.MkdirAll(ssdFullPath, 0755)
	_ = os.MkdirAll(hddFullPath, 0755)

	bus := event.NewBus()
	catalog := librarian.NewSeshat(bus, nil)

	// Register the 3 providers
	provSSDReadOnly := &config.StorageProvider{
		ID:   "prov-ssd-readonly",
		Type: "local",
		Path: ssdReadOnlyPath,
		Capabilities: map[string]string{
			"latency":    "low",
			"drive_type": "ssd",
			"control":    "index_observe",
			"read_only":  "true",
		},
	}
	provSSDFull := &config.StorageProvider{
		ID:   "prov-ssd-full",
		Type: "local",
		Path: ssdFullPath,
		Capabilities: map[string]string{
			"latency":    "low",
			"drive_type": "ssd",
			"control":    "full",
			"read_only":  "false",
		},
	}
	provHDDFull := &config.StorageProvider{
		ID:   "prov-hdd-full",
		Type: "local",
		Path: hddFullPath,
		Capabilities: map[string]string{
			"latency":    "high",
			"drive_type": "hdd",
			"control":    "full",
			"read_only":  "false",
		},
	}

	catalog.AddProvider(provSSDReadOnly)
	catalog.AddProvider(provSSDFull)
	catalog.AddProvider(provHDDFull)

	// Define policies: replicate target = 2, and migrate
	policies := []config.Policy{
		{
			ID:       "policy-replicate",
			Type:     "replicate",
			Target:   "object",
			Value:    "2",
			Priority: 1,
		},
		{
			ID:       "tier-by-frequency",
			Type:     "migrate",
			Target:   "object",
			Value:    "ssd-for-frequent-hdd-for-infrequent",
			Priority: 10,
		},
	}

	_ = staff.NewTuoni(catalog, bus, policies)
	_ = staff.NewBoatman(catalog, bus)
	observer := staff.NewObserver(catalog, bus)

	// Test 1: "index_observe" read-only exclusion logic.
	// We write a file to ssd-full. It should replicate to hdd-full, but NEVER to ssd-readonly.
	testFile1 := "doc1.txt"
	_ = os.WriteFile(filepath.Join(ssdFullPath, testFile1), []byte("content of doc1"), 0644)

	observer.ScanProviders()

	// Wait for replication to trigger and finish
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		obj, err := catalog.GetObject(testFile1)
		if err == nil {
			obj.RLock()
			if len(obj.Versions) > 0 && len(obj.Versions[0].Copies) >= 2 {
				obj.RUnlock()
				break
			}
			obj.RUnlock()
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Verify that prov-ssd-readonly has NO copies of doc1.txt
	obj1, err := catalog.GetObject(testFile1)
	if err != nil {
		t.Fatalf("doc1.txt not found in catalog: %v", err)
	}
	obj1.RLock()
	for _, cp := range obj1.Versions[0].Copies {
		if cp.ProviderID == "prov-ssd-readonly" {
			t.Errorf("Error: read-only provider prov-ssd-readonly received replicated copy!")
		}
	}
	obj1.RUnlock()

	// Verify that it actually replicated to hdd-full
	replicatedPath := filepath.Join(hddFullPath, testFile1)
	if _, err := os.Stat(replicatedPath); os.IsNotExist(err) {
		t.Errorf("Expected replication of doc1.txt to hdd-full")
	}

	// Test 2: Hot file tiering.
	// Create an object that is hot (access_frequency: high) on hdd-full.
	// Tuoni should migrate/replicate it to ssd-full.
	testFileHot := "hotfile.txt"
	hotPath := filepath.Join(hddFullPath, testFileHot)
	_ = os.WriteFile(hotPath, []byte("hot bytes"), 0644)

	// Discovered by observer (defaults to low access frequency, let's update catalog entry to high access frequency)
	observer.ScanProviders()

	// Wait for discovery
	var hotObj *librarian.Object
	for time.Now().Before(time.Now().Add(1 * time.Second)) {
		hotObj, err = catalog.GetObject(testFileHot)
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if hotObj == nil {
		t.Fatalf("Failed to discover hotfile.txt")
	}

	// Manually set access frequency to high
	hotObj.Lock()
	hotObj.Metadata["access_frequency"] = "high"
	hotObj.Unlock()

	// Re-evaluate policies
	bus.Publish(event.Event{
		ID:        "trigger-policy-eval",
		Type:      event.EventPolicyChanged,
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"policies": policies,
		},
	})

	// Wait up to 2 seconds for migration to SSD
	deadline = time.Now().Add(2 * time.Second)
	var foundSSD bool
	for time.Now().Before(deadline) {
		hotObj, _ = catalog.GetObject(testFileHot)
		hotObj.RLock()
		if len(hotObj.Versions) > 0 {
			for _, cp := range hotObj.Versions[0].Copies {
				if cp.ProviderID == "prov-ssd-full" && cp.State == "healthy" {
					foundSSD = true
					break
				}
			}
		}
		hotObj.RUnlock()
		if foundSSD {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if !foundSSD {
		t.Errorf("Expected hotfile.txt to be tiered/copied to prov-ssd-full")
	}

	// Check that the file actually exists on ssd-full
	if _, err := os.Stat(filepath.Join(ssdFullPath, testFileHot)); os.IsNotExist(err) {
		t.Errorf("Expected physical file hotfile.txt on ssd-full")
	}
}

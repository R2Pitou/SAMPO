package librarian_test

import (
	"os"
	"testing"
	"time"

	"mash/internal/config"
	"mash/internal/event"
	"mash/internal/librarian"
)

func TestSQLitePersistence(t *testing.T) {
	dbFile := "test_mash.db"
	defer os.Remove(dbFile)

	// Create SQLite store
	store, err := librarian.NewSQLiteStore(dbFile)
	if err != nil {
		t.Fatalf("failed to create sqlite store: %v", err)
	}
	defer store.Close()

	// 1. Save and Load Provider
	prov := &config.StorageProvider{
		ID:   "p-test-1",
		Type: "local",
		Path: "/test/path/primary",
		Capabilities: map[string]string{
			"speed": "fast",
		},
	}
	if err := store.SaveProvider(prov); err != nil {
		t.Fatalf("failed to save provider: %v", err)
	}

	providers, err := store.LoadProviders()
	if err != nil {
		t.Fatalf("failed to load providers: %v", err)
	}
	if len(providers) != 1 || providers[0].ID != "p-test-1" || providers[0].Capabilities["speed"] != "fast" {
		t.Errorf("loaded provider does not match: %+v", providers)
	}

	// 2. Save and Load Project
	proj := &librarian.Project{
		ID:   "proj-test-1",
		Name: "Test Project",
		Policies: []config.Policy{
			{
				ID:    "pol-1",
				Type:  "replicate",
				Value: "2",
			},
		},
		Metadata: map[string]interface{}{
			"owner": "admin",
		},
	}
	if err := store.SaveProject(proj); err != nil {
		t.Fatalf("failed to save project: %v", err)
	}

	projects, err := store.LoadProjects()
	if err != nil {
		t.Fatalf("failed to load projects: %v", err)
	}
	if len(projects) != 1 || projects[0].ID != "proj-test-1" || projects[0].Name != "Test Project" {
		t.Errorf("loaded project does not match: %+v", projects)
	}

	// 3. Save and Load Object with Versions and Copies
	cp := librarian.Copy{
		ID:         "copy-1",
		ProviderID: "p-test-1",
		Path:       "/test/path/primary/file.txt",
		State:      "healthy",
		VerifiedAt: time.Now().Truncate(time.Second),
	}

	ver := librarian.Version{
		ID:        "v1",
		Hash:      "hash-12345",
		Timestamp: time.Now().Truncate(time.Second),
		Copies:    []librarian.Copy{cp},
	}

	obj := &librarian.Object{
		ID:        "obj-test-1",
		Hash:      "hash-12345",
		ProjectID: "proj-test-1",
		Metadata: map[string]interface{}{
			"type": "text",
		},
		Versions: []librarian.Version{ver},
	}

	if err := store.SaveObject(obj); err != nil {
		t.Fatalf("failed to save object: %v", err)
	}

	objects, err := store.LoadObjects()
	if err != nil {
		t.Fatalf("failed to load objects: %v", err)
	}
	if len(objects) != 1 || objects[0].ID != "obj-test-1" || len(objects[0].Versions) != 1 || len(objects[0].Versions[0].Copies) != 1 {
		t.Fatalf("loaded object structurally mismatched: %+v", objects)
	}

	loadedObj := objects[0]
	if loadedObj.Versions[0].Copies[0].Path != "/test/path/primary/file.txt" {
		t.Errorf("loaded copy path does not match: %s", loadedObj.Versions[0].Copies[0].Path)
	}

	// 4. Update Copy State
	store.UpdateCopyState("obj-test-1", "v1", "copy-1", "corrupted")
	objectsUpdated, err := store.LoadObjects()
	if err != nil {
		t.Fatalf("failed to load objects after copy update: %v", err)
	}
	if objectsUpdated[0].Versions[0].Copies[0].State != "corrupted" {
		t.Errorf("expected copy state to be updated to 'corrupted', got %s", objectsUpdated[0].Versions[0].Copies[0].State)
	}
}

func TestSeshatWithSQLiteIntegration(t *testing.T) {
	dbFile := "test_seshat_integration.db"
	defer os.Remove(dbFile)

	store, err := librarian.NewSQLiteStore(dbFile)
	if err != nil {
		t.Fatalf("failed to create sqlite store: %v", err)
	}

	bus := event.NewBus()
	catalog := librarian.NewSeshat(bus, store)

	// Save data via catalog methods
	prov := &config.StorageProvider{
		ID:   "p-int-1",
		Type: "local",
		Path: "/int/path",
	}
	catalog.AddProvider(prov)

	obj := &librarian.Object{
		ID:        "obj-int-1",
		Hash:      "hash-int-99",
		ProjectID: "proj-int-1",
		Versions: []librarian.Version{
			{
				ID:        "v1",
				Hash:      "hash-int-99",
				Timestamp: time.Now(),
			},
		},
	}
	catalog.PutObject(obj)

	// Close database
	store.Close()

	// Load again from database using a new Seshat instance
	newStore, err := librarian.NewSQLiteStore(dbFile)
	if err != nil {
		t.Fatalf("failed to reload sqlite store: %v", err)
	}
	defer newStore.Close()

	newCatalog := librarian.NewSeshat(bus, newStore)

	// Confirm that state was perfectly restored from DB
	restoredObj, err := newCatalog.GetObject("obj-int-1")
	if err != nil {
		t.Fatalf("failed to retrieve object from reloaded catalog: %v", err)
	}
	if restoredObj.Hash != "hash-int-99" {
		t.Errorf("expected restored object hash 'hash-int-99', got %s", restoredObj.Hash)
	}

	restoredProv, err := newCatalog.GetProvider("p-int-1")
	if err != nil {
		t.Fatalf("failed to retrieve provider from reloaded catalog: %v", err)
	}
	if restoredProv.Path != "/int/path" {
		t.Errorf("expected restored provider path '/int/path', got %s", restoredProv.Path)
	}
}

package librarian_test

import (
	"testing"

	"mash/internal/config"
	"mash/internal/event"
	"mash/internal/librarian"
)

func TestSeshatCatalogue(t *testing.T) {
	bus := event.NewBus()
	catalog := librarian.NewSeshat(bus, nil)

	prov := &config.StorageProvider{
		ID:   "p1",
		Type: "local",
		Path: "/tmp/p1",
	}

	catalog.AddProvider(prov)

	p, err := catalog.GetProvider("p1")
	if err != nil {
		t.Fatalf("unexpected error getting provider: %v", err)
	}

	if p.Path != "/tmp/p1" {
		t.Errorf("expected provider path /tmp/p1, got %s", p.Path)
	}

	// Add an object
	obj := &librarian.Object{
		ID:        "obj-1",
		Hash:      "sha256-hash",
		ProjectID: "proj-1",
	}

	catalog.PutObject(obj)

	retrieved, err := catalog.GetObject("obj-1")
	if err != nil {
		t.Fatalf("unexpected error getting object: %v", err)
	}

	if retrieved.Hash != "sha256-hash" {
		t.Errorf("expected object hash sha256-hash, got %s", retrieved.Hash)
	}
}

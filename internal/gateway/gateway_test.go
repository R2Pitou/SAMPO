package gateway_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mash/internal/config"
	"mash/internal/event"
	"mash/internal/gateway"
	"mash/internal/librarian"
)

func TestHandleObjects(t *testing.T) {
	bus := event.NewBus()
	catalog := librarian.NewSeshat(bus, nil)
	gw := gateway.NewGateway(catalog, 8080)

	// Add an initial object
	initialObj := &librarian.Object{
		ID:        "obj-test",
		Hash:      "hash-123",
		ProjectID: "proj-123",
		Metadata:  map[string]interface{}{"note": "hello"},
	}
	catalog.PutObject(initialObj)

	t.Run("GET - List Objects", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/objects", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		gw.HandleObjects(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status OK, got %v", rr.Code)
		}

		var objs []librarian.Object
		if err := json.Unmarshal(rr.Body.Bytes(), &objs); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(objs) != 1 {
			t.Fatalf("expected 1 object, got %d", len(objs))
		}

		if objs[0].ID != "obj-test" || objs[0].Hash != "hash-123" {
			t.Errorf("unexpected object content: %+v", objs[0])
		}
	})

	t.Run("POST - Create Object", func(t *testing.T) {
		newObj := librarian.Object{
			ID:        "obj-post",
			Hash:      "hash-post",
			ProjectID: "proj-post",
			Metadata:  map[string]interface{}{"status": "new"},
		}

		bodyBytes, _ := json.Marshal(newObj)
		req, err := http.NewRequest(http.MethodPost, "/objects", bytes.NewBuffer(bodyBytes))
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		gw.HandleObjects(rr, req)

		if rr.Code != http.StatusCreated {
			t.Errorf("expected status Created, got %v", rr.Code)
		}

		var returnedObj librarian.Object
		if err := json.Unmarshal(rr.Body.Bytes(), &returnedObj); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if returnedObj.ID != "obj-post" {
			t.Errorf("expected ID 'obj-post', got '%s'", returnedObj.ID)
		}

		// Verify it was added to the catalog
		catObj, err := catalog.GetObject("obj-post")
		if err != nil {
			t.Fatalf("object was not saved to catalog: %v", err)
		}
		if catObj.Hash != "hash-post" {
			t.Errorf("expected hash-post, got %s", catObj.Hash)
		}
	})

	t.Run("POST - Bad Payload", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, "/objects", bytes.NewBufferString("{invalid json"))
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		gw.HandleObjects(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status BadRequest, got %v", rr.Code)
		}
	})

	t.Run("POST - Unsafe/Invalid Object IDs", func(t *testing.T) {
		unsafeIDs := []string{
			"../../etc/passwd",
			"/etc/passwd",
			"..",
			"C:\\windows\\win.ini",
			"foo/../../bar",
			"\\absolute\\path",
			"",
		}

		for _, id := range unsafeIDs {
			t.Run("ID: "+id, func(t *testing.T) {
				badObj := librarian.Object{
					ID:   id,
					Hash: "hash-bad",
				}
				bodyBytes, _ := json.Marshal(badObj)
				req, err := http.NewRequest(http.MethodPost, "/objects", bytes.NewBuffer(bodyBytes))
				if err != nil {
					t.Fatalf("failed to create request: %v", err)
				}

				rr := httptest.NewRecorder()
				gw.HandleObjects(rr, req)

				if rr.Code != http.StatusBadRequest {
					t.Errorf("expected status BadRequest for ID '%s', got %v", id, rr.Code)
				}
			})
		}
	})

	t.Run("POST - Safe Subdirectory Object ID", func(t *testing.T) {
		safeObj := librarian.Object{
			ID:   "docs/sub/my-doc.txt",
			Hash: "hash-safe",
		}
		bodyBytes, _ := json.Marshal(safeObj)
		req, err := http.NewRequest(http.MethodPost, "/objects", bytes.NewBuffer(bodyBytes))
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		gw.HandleObjects(rr, req)

		if rr.Code != http.StatusCreated {
			t.Errorf("expected status Created for safe ID, got %v", rr.Code)
		}
	})

	t.Run("PUT - Method Not Allowed", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPut, "/objects", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		gw.HandleObjects(rr, req)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status MethodNotAllowed, got %v", rr.Code)
		}
	})
}

func TestHandleProviders(t *testing.T) {
	bus := event.NewBus()
	catalog := librarian.NewSeshat(bus, nil)
	gw := gateway.NewGateway(catalog, 8080)

	catalog.AddProvider(&config.StorageProvider{
		ID:   "p-gw",
		Type: "local",
		Path: "/gw/path",
	})

	t.Run("GET - List Providers", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/providers", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		gw.HandleProviders(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status OK, got %v", rr.Code)
		}

		var provs []config.StorageProvider
		if err := json.Unmarshal(rr.Body.Bytes(), &provs); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(provs) != 1 {
			t.Fatalf("expected 1 provider, got %d", len(provs))
		}

		if provs[0].ID != "p-gw" || provs[0].Path != "/gw/path" {
			t.Errorf("unexpected provider content: %+v", provs[0])
		}
	})

	t.Run("POST - Method Not Allowed", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, "/providers", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		gw.HandleProviders(rr, req)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status MethodNotAllowed, got %v", rr.Code)
		}
	})
}

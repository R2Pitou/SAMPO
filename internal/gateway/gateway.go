package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strings"

	"mash/internal/librarian"
)

// Gateway provides external clients with REST interfaces to interact with Seshat catalogue.
type Gateway struct {
	catalog *librarian.Seshat
	port    int
}

// NewGateway creates a Gateway instance.
func NewGateway(catalog *librarian.Seshat, port int) *Gateway {
	return &Gateway{
		catalog: catalog,
		port:    port,
	}
}

// Start boots the HTTP REST server.
func (g *Gateway) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/objects", g.HandleObjects)
	mux.HandleFunc("/providers", g.HandleProviders)

	addr := fmt.Sprintf(":%d", g.port)
	return http.ListenAndServe(addr, mux)
}

// HandleObjects lists all metadata tracked objects or registers custom files.
func (g *Gateway) HandleObjects(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodGet {
		objects := g.catalog.ListObjects()
		_ = json.NewEncoder(w).Encode(objects)
		return
	}

	if r.Method == http.MethodPost {
		var obj librarian.Object
		if err := json.NewDecoder(r.Body).Decode(&obj); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}
		if !isValidObjectID(obj.ID) {
			http.Error(w, "Invalid or unsafe object ID", http.StatusBadRequest)
			return
		}
		g.catalog.PutObject(&obj)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(obj)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// isValidObjectID returns true if the object ID is safe and valid (not empty, no path traversal).
func isValidObjectID(id string) bool {
	if id == "" {
		return false
	}
	// Normalize backslashes to slashes to handle Windows/Unix styles
	normalized := strings.ReplaceAll(id, "\\", "/")

	// Clean the path using path.Clean (works with forward slashes)
	cleaned := path.Clean(normalized)

	// If it's empty, "." or contains drive indicators (e.g., ":"), it's invalid
	if cleaned == "" || cleaned == "." || strings.Contains(cleaned, ":") {
		return false
	}

	// Must not escape directory (not start with "../", "/" or look like an absolute/relative escape)
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.HasPrefix(cleaned, "/") {
		return false
	}

	return true
}

// HandleProviders lists storage provider status.
func (g *Gateway) HandleProviders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodGet {
		providers := g.catalog.ListProviders()
		_ = json.NewEncoder(w).Encode(providers)
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

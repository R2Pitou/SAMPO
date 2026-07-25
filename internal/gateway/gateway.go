package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"

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
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		g.catalog.PutObject(&obj)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(obj)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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

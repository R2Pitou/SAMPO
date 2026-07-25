package librarian

import (
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"mash/internal/config"
	"mash/internal/event"
)

// Object represents a logical file or dataset managed by MAS-H.
type Object struct {
	mu        sync.RWMutex
	ID        string                 `json:"id"`
	Hash      string                 `json:"hash"`      // latest content hash
	ProjectID string                 `json:"projectId"` // associated project (optional)
	Metadata  map[string]interface{} `json:"metadata"`
	Versions  []Version              `json:"versions"`
}

// MarshalJSON provides thread-safe custom JSON serialization.
func (o *Object) MarshalJSON() ([]byte, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	// Create an alias to avoid infinite recursion
	type Alias Object
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(o),
	})
}

// Lock locks the object for exclusive write access.
func (o *Object) Lock() {
	o.mu.Lock()
}

// Unlock unlocks the object.
func (o *Object) Unlock() {
	o.mu.Unlock()
}

// RLock locks the object for read access.
func (o *Object) RLock() {
	o.mu.RLock()
}

// RUnlock unlocks the object's read lock.
func (o *Object) RUnlock() {
	o.mu.RUnlock()
}

// Version represents an immutable snapshot of an Object.
type Version struct {
	ID        string    `json:"id"`
	Hash      string    `json:"hash"`
	Timestamp time.Time `json:"timestamp"`
	Copies    []Copy    `json:"copies"`
}

// Copy is a physical instance of an Object Version on a specific storage provider.
type Copy struct {
	ID         string    `json:"id"`
	ProviderID string    `json:"providerId"`
	Path       string    `json:"path"`
	State      string    `json:"state"` // healthy, corrupted, missing, degraded
	VerifiedAt time.Time `json:"verifiedAt"`
}

// Project is a collection of related Objects with scoped storage policies.
type Project struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Policies []config.Policy        `json:"policies"`
	Metadata map[string]interface{} `json:"metadata"`
}

// Seshat serves as the metadata catalogue (the librarian's ledger).
type Seshat struct {
	mu        sync.RWMutex
	objects   map[string]*Object
	projects  map[string]*Project
	providers map[string]*config.StorageProvider
	eventBus  *event.Bus
	dbStore   *SQLiteStore
}

// NewSeshat creates a new instance of the Seshat catalogue.
func NewSeshat(bus *event.Bus, dbStore *SQLiteStore) *Seshat {
	s := &Seshat{
		objects:   make(map[string]*Object),
		projects:  make(map[string]*Project),
		providers: make(map[string]*config.StorageProvider),
		eventBus:  bus,
		dbStore:   dbStore,
	}

	if dbStore != nil {
		s.loadFromDB()
	}

	return s
}

func (s *Seshat) loadFromDB() {
	log.Println("[Seshat] Restoring metadata state from persistent SQLite database...")

	// 1. Load Providers
	if provs, err := s.dbStore.LoadProviders(); err == nil {
		for _, p := range provs {
			s.providers[p.ID] = p
			log.Printf("[Seshat] Restored Storage Provider: %s", p.ID)
		}
	}

	// 2. Load Projects
	if projs, err := s.dbStore.LoadProjects(); err == nil {
		for _, p := range projs {
			s.projects[p.ID] = p
			log.Printf("[Seshat] Restored Project: %s", p.ID)
		}
	}

	// 3. Load Objects
	if objs, err := s.dbStore.LoadObjects(); err == nil {
		for _, obj := range objs {
			s.objects[obj.ID] = obj
			log.Printf("[Seshat] Restored tracked Object: %s", obj.ID)
		}
	}
}

// AddProvider registers a storage provider.
func (s *Seshat) AddProvider(p *config.StorageProvider) {
	s.mu.Lock()
	s.providers[p.ID] = p
	s.mu.Unlock()

	if s.dbStore != nil {
		if err := s.dbStore.SaveProvider(p); err != nil {
			log.Printf("[Seshat] Error persisting provider %s: %v", p.ID, err)
		}
	}
}

// GetProvider retrieves a storage provider config.
func (s *Seshat) GetProvider(id string) (*config.StorageProvider, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, exists := s.providers[id]
	if !exists {
		return nil, errors.New("provider not found")
	}
	return p, nil
}

// ListProviders returns all registered providers.
func (s *Seshat) ListProviders() []*config.StorageProvider {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*config.StorageProvider, 0, len(s.providers))
	for _, p := range s.providers {
		list = append(list, p)
	}
	return list
}

// AddProject registers a new logical Project.
func (s *Seshat) AddProject(p *Project) {
	s.mu.Lock()
	s.projects[p.ID] = p
	s.mu.Unlock()

	if s.dbStore != nil {
		if err := s.dbStore.SaveProject(p); err != nil {
			log.Printf("[Seshat] Error persisting project %s: %v", p.ID, err)
		}
	}
}

// GetProject returns a Project by ID.
func (s *Seshat) GetProject(id string) (*Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, exists := s.projects[id]
	if !exists {
		return nil, errors.New("project not found")
	}
	return p, nil
}

// PutObject creates or updates an Object in the catalogue.
func (s *Seshat) PutObject(obj *Object) {
	s.mu.Lock()
	s.objects[obj.ID] = obj
	s.mu.Unlock()

	if s.dbStore != nil {
		obj.RLock()
		err := s.dbStore.SaveObject(obj)
		obj.RUnlock()
		if err != nil {
			log.Printf("[Seshat] Error persisting object %s: %v", obj.ID, err)
		}
	}

	// Notify that an object was updated or created
	s.eventBus.Publish(event.Event{
		ID:        obj.ID,
		Type:      event.EventObjectCreated,
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"objectId":  obj.ID,
			"hash":      obj.Hash,
			"projectId": obj.ProjectID,
		},
	})
}

// GetObject retrieves an Object by ID.
func (s *Seshat) GetObject(id string) (*Object, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	obj, exists := s.objects[id]
	if !exists {
		return nil, errors.New("object not found")
	}
	return obj, nil
}

// ListObjects returns all objects in the catalogue.
func (s *Seshat) ListObjects() []*Object {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*Object, 0, len(s.objects))
	for _, obj := range s.objects {
		list = append(list, obj)
	}
	return list
}

// AddCopy adds a new physical instance location (Copy) for a specific version.
func (s *Seshat) AddCopy(objectID string, versionID string, cp Copy) error {
	s.mu.RLock()
	obj, exists := s.objects[objectID]
	s.mu.RUnlock()

	if !exists {
		return errors.New("object not found")
	}

	obj.Lock()
	for i, ver := range obj.Versions {
		if ver.ID == versionID {
			obj.Versions[i].Copies = append(obj.Versions[i].Copies, cp)
			obj.Unlock()

			if s.dbStore != nil {
				obj.RLock()
				err := s.dbStore.SaveObject(obj)
				obj.RUnlock()
				if err != nil {
					log.Printf("[Seshat] Error persisting updated copies for object %s: %v", obj.ID, err)
				}
			}
			return nil
		}
	}
	obj.Unlock()

	return errors.New("version not found")
}

// UpdateCopyState updates the state of a copy on an object with lock safety.
func (s *Seshat) UpdateCopyState(objID string, versionIndex, copyIndex int, newState string) {
	s.mu.RLock()
	obj, exists := s.objects[objID]
	s.mu.RUnlock()

	if exists {
		obj.Lock()
		if versionIndex < len(obj.Versions) && copyIndex < len(obj.Versions[versionIndex].Copies) {
			obj.Versions[versionIndex].Copies[copyIndex].State = newState
			obj.Versions[versionIndex].Copies[copyIndex].VerifiedAt = time.Now()
			versionID := obj.Versions[versionIndex].ID
			copyID := obj.Versions[versionIndex].Copies[copyIndex].ID
			obj.Unlock()

			if s.dbStore != nil {
				s.dbStore.UpdateCopyState(objID, versionID, copyID, newState)
			}
			return
		}
		obj.Unlock()
	}
}

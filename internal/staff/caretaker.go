package staff

import (
	"log"
	"time"

	"mash/internal/event"
	"mash/internal/librarian"
)

// Caretaker manages background maintenance: replica repair, hash checks, and health updates.
type Caretaker struct {
	catalog  *librarian.Seshat
	eventBus *event.Bus
	quit     chan struct{}
}

// NewCaretaker creates a Caretaker instance.
func NewCaretaker(catalog *librarian.Seshat, bus *event.Bus) *Caretaker {
	return &Caretaker{
		catalog:  catalog,
		eventBus: bus,
		quit:     make(chan struct{}),
	}
}

// Start launches background health verification.
func (c *Caretaker) Start(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				c.PerformIntegrityCheck()
			case <-c.quit:
				ticker.Stop()
				return
			}
		}
	}()
}

// Stop stops the caretaker worker loop.
func (c *Caretaker) Stop() {
	close(c.quit)
}

// PerformIntegrityCheck verifies the hash of each Copy in the catalogue.
func (c *Caretaker) PerformIntegrityCheck() {
	log.Println("[Caretaker] Starting scheduled health and integrity check...")
	objects := c.catalog.ListObjects()

	for _, obj := range objects {
		obj.RLock()
		// Copy version/copies info locally under a read-lock so we don't hold the lock during long filesystem reads/hashing.
		type CopyInfo struct {
			id         string
			providerID string
			path       string
			state      string
		}
		type VersionInfo struct {
			index  int
			hash   string
			copies []CopyInfo
		}

		versions := make([]VersionInfo, len(obj.Versions))
		for idx, v := range obj.Versions {
			copies := make([]CopyInfo, len(v.Copies))
			for cIdx, cp := range v.Copies {
				copies[cIdx] = CopyInfo{
					id:         cp.ID,
					providerID: cp.ProviderID,
					path:       cp.Path,
					state:      cp.State,
				}
			}
			versions[idx] = VersionInfo{
				index:  idx,
				hash:   v.Hash,
				copies: copies,
			}
		}
		obj.RUnlock()

		for _, vInfo := range versions {
			for cIdx, cp := range vInfo.copies {
				// Re-verify hash of local copies
				currentHash, err := hashFile(cp.path)
				if err != nil {
					log.Printf("[Caretaker] Copy %s is missing or inaccessible: %v", cp.id, err)
					if cp.state != "missing" {
						c.catalog.UpdateCopyState(obj.ID, vInfo.index, cIdx, "missing")
						c.eventBus.Publish(event.Event{
							ID:        cp.id + "-missing",
							Type:      event.EventProviderOffline,
							Timestamp: time.Now(),
							Payload: map[string]interface{}{
								"objectId":   obj.ID,
								"copyId":     cp.id,
								"providerId": cp.providerID,
							},
						})
					}
					continue
				}

				if currentHash != vInfo.hash {
					log.Printf("[Caretaker] Copy %s failed hash validation! Expected %s, got %s", cp.id, vInfo.hash, currentHash)
					if cp.state != "corrupted" {
						c.catalog.UpdateCopyState(obj.ID, vInfo.index, cIdx, "corrupted")
						c.eventBus.Publish(event.Event{
							ID:        cp.id + "-corrupted",
							Type:      event.EventCopyCorrupted,
							Timestamp: time.Now(),
							Payload: map[string]interface{}{
								"objectId":   obj.ID,
								"copyId":     cp.id,
								"providerId": cp.providerID,
							},
						})
					}
				} else {
					if cp.state != "healthy" {
						c.catalog.UpdateCopyState(obj.ID, vInfo.index, cIdx, "healthy")
					}
				}
			}
		}
	}
}

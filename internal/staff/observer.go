package staff

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"mash/internal/event"
	"mash/internal/librarian"
)

// Observer watches registered Storage Providers and discovers files/objects.
type Observer struct {
	catalog  *librarian.Seshat
	eventBus *event.Bus
	quit     chan struct{}
}

// NewObserver initializes an Observer.
func NewObserver(catalog *librarian.Seshat, bus *event.Bus) *Observer {
	return &Observer{
		catalog:  catalog,
		eventBus: bus,
		quit:     make(chan struct{}),
	}
}

// Start runs the observation loop periodically scanning paths.
func (o *Observer) Start(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				o.ScanProviders()
			case <-o.quit:
				ticker.Stop()
				return
			}
		}
	}()
}

// Stop halts the observation loop.
func (o *Observer) Stop() {
	close(o.quit)
}

// ScanProviders scans all local storage provider paths for file additions/modifications.
func (o *Observer) ScanProviders() {
	providers := o.catalog.ListProviders()
	for _, p := range providers {
		if p.Type != "local" {
			continue
		}

		err := filepath.Walk(p.Path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}

			// Determine relative path to use as a collision-free objID
			relPath, err := filepath.Rel(p.Path, path)
			if err != nil {
				log.Printf("[Observer] Walk error determining rel path for %s: %v", path, err)
				return nil
			}

			// Generate virtual identifier / hash
			hash, err := hashFile(path)
			if err != nil {
				log.Printf("[Observer] Walk error hashing %s: %v", path, err)
				return nil
			}

			// Check if already in catalog using relPath
			objID := relPath
			_, err = o.catalog.GetObject(objID)
			if err != nil {
				// Not tracked yet! Register Object.
				log.Printf("[Observer] Discovered tracked Object: %s at %s", objID, path)

				cp := librarian.Copy{
					ID:         objID + "-" + p.ID + "-copy",
					ProviderID: p.ID,
					Path:       path,
					State:      "healthy",
					VerifiedAt: time.Now(),
				}

				version := librarian.Version{
					ID:        "v1",
					Hash:      hash,
					Timestamp: time.Now(),
					Copies:    []librarian.Copy{cp},
				}

				obj := &librarian.Object{
					ID:       objID,
					Hash:     hash,
					Versions: []librarian.Version{version},
					Metadata: map[string]interface{}{
						"originalPath": path,
					},
				}

				o.catalog.PutObject(obj)
			}
			return nil
		})
		if err != nil {
			log.Printf("[Observer] Walk error scanning provider %s: %v", p.ID, err)
		}
	}
}

func hashFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

package staff

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"mash/internal/event"
	"mash/internal/librarian"
)

// Boatman handles physical object transfers and replication jobs between storage providers.
type Boatman struct {
	catalog  *librarian.Seshat
	eventBus *event.Bus
}

// NewBoatman initializes a new Boatman transfer service.
func NewBoatman(catalog *librarian.Seshat, bus *event.Bus) *Boatman {
	b := &Boatman{
		catalog:  catalog,
		eventBus: bus,
	}

	bus.Subscribe(event.EventTransferPlanCreated, b.HandleTransferPlan)

	return b
}

// HandleTransferPlan executes a transfer or replication action requested by Tuoni.
func (b *Boatman) HandleTransferPlan(e event.Event) {
	objID, ok1 := e.Payload["objectId"].(string)
	verID, ok2 := e.Payload["versionId"].(string)
	srcProvID, ok3 := e.Payload["sourceProviderId"].(string)
	srcPath, ok4 := e.Payload["sourcePath"].(string)
	tgtProvID, ok5 := e.Payload["targetProviderId"].(string)

	if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 {
		log.Println("[Boatman] Missing required payload arguments for transfer job")
		return
	}

	log.Printf("[Boatman] Starting replication job: %s from Provider %s to %s", objID, srcProvID, tgtProvID)

	b.eventBus.Publish(event.Event{
		ID:        objID + "-" + tgtProvID + "-job",
		Type:      event.EventJobStarted,
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"objectId":         objID,
			"targetProviderId": tgtProvID,
		},
	})

	// Retrieve actual source path & target provider paths
	tgtProvider, err := b.catalog.GetProvider(tgtProvID)
	if err != nil {
		b.failJob(objID, tgtProvID, fmt.Errorf("target provider not found in catalog: %v", err))
		return
	}

	// Use objID (which is the relative path from the provider) to build the correct destination filepath structure.
	targetFilePath := filepath.Join(tgtProvider.Path, objID)
	targetDir := filepath.Dir(targetFilePath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		b.failJob(objID, tgtProvID, fmt.Errorf("failed to create directory tree %s: %v", targetDir, err))
		return
	}

	// Perform actual file transfer / copy
	if err := b.copyFile(srcPath, targetFilePath); err != nil {
		b.failJob(objID, tgtProvID, fmt.Errorf("file copy failed: %v", err))
		return
	}

	// Successfully replicated! Add new Copy to the Seshat Catalogue.
	newCopy := librarian.Copy{
		ID:         objID + "-" + tgtProvID + "-copy",
		ProviderID: tgtProvID,
		Path:       targetFilePath,
		State:      "healthy",
		VerifiedAt: time.Now(),
	}

	if err := b.catalog.AddCopy(objID, verID, newCopy); err != nil {
		b.failJob(objID, tgtProvID, fmt.Errorf("failed to register new copy in catalogue: %v", err))
		return
	}

	log.Printf("[Boatman] Successfully replicated Object %s to Provider %s", objID, tgtProvID)

	b.eventBus.Publish(event.Event{
		ID:        objID + "-" + tgtProvID + "-job",
		Type:      event.EventJobCompleted,
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"objectId":         objID,
			"targetProviderId": tgtProvID,
			"copyPath":         targetFilePath,
		},
	})
}

func (b *Boatman) copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func (b *Boatman) failJob(objID, providerID string, err error) {
	log.Printf("[Boatman] Transfer failed: %v", err)
	b.eventBus.Publish(event.Event{
		ID:        objID + "-" + providerID + "-job",
		Type:      event.EventJobFailed,
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"objectId":         objID,
			"targetProviderId": providerID,
			"error":            err.Error(),
		},
	})
}

package staff

import (
	"log"
	"strconv"
	"time"

	"mash/internal/config"
	"mash/internal/event"
	"mash/internal/librarian"
)

// Tuoni is the reasoning decision engine.
type Tuoni struct {
	catalog  *librarian.Seshat
	eventBus *event.Bus
	policies []config.Policy
}

// NewTuoni instantiates a new Tuoni service.
func NewTuoni(catalog *librarian.Seshat, bus *event.Bus, policies []config.Policy) *Tuoni {
	t := &Tuoni{
		catalog:  catalog,
		eventBus: bus,
		policies: policies,
	}

	// Subscribe to event triggers
	bus.Subscribe(event.EventObjectCreated, t.HandleObjectCreated)
	bus.Subscribe(event.EventPolicyChanged, t.HandlePolicyChanged)

	return t
}

// HandleObjectCreated processes metadata and evaluates storage policies.
func (t *Tuoni) HandleObjectCreated(e event.Event) {
	objID, ok := e.Payload["objectId"].(string)
	if !ok {
		return
	}

	log.Printf("[Tuoni] Evaluating policies for newly discovered Object: %s", objID)
	t.EvaluatePoliciesForObject(objID)
}

// HandlePolicyChanged triggers full policy evaluation of the entire catalogue.
func (t *Tuoni) HandlePolicyChanged(e event.Event) {
	log.Println("[Tuoni] Storage policies modified. Re-evaluating entire catalogue...")

	// Reload policies if they are in the event payload
	if rawPolicies, exists := e.Payload["policies"]; exists {
		if updated, ok := rawPolicies.([]config.Policy); ok {
			t.policies = updated
		}
	}

	objects := t.catalog.ListObjects()
	for _, obj := range objects {
		t.EvaluatePoliciesForObject(obj.ID)
	}
}

// EvaluatePoliciesForObject scans applicable policies and issues Transfer Plans.
func (t *Tuoni) EvaluatePoliciesForObject(objID string) {
	obj, err := t.catalog.GetObject(objID)
	if err != nil {
		log.Printf("[Tuoni] Error fetching object %s for evaluation: %v", objID, err)
		return
	}

	obj.RLock()
	defer obj.RUnlock()

	// Simple replication policy logic:
	// Find the replication policies that apply.
	replicationTargetCount := 1
	for _, policy := range t.policies {
		if policy.Type == "replicate" {
			if policy.Target == "object" || (policy.Target == "project" && obj.ProjectID == policy.ID) {
				count, err := strconv.Atoi(policy.Value)
				if err == nil && count > replicationTargetCount {
					replicationTargetCount = count
				}
			}
		}
	}

	// Count current healthy copies
	currentHealthyCount := 0
	var sourceCopy *librarian.Copy
	var latestVersionID string
	var latestVersionCopies []librarian.Copy

	if len(obj.Versions) > 0 {
		latestVersion := obj.Versions[len(obj.Versions)-1]
		latestVersionID = latestVersion.ID
		latestVersionCopies = make([]librarian.Copy, len(latestVersion.Copies))
		copy(latestVersionCopies, latestVersion.Copies)

		for i := range latestVersion.Copies {
			cp := &latestVersion.Copies[i]
			if cp.State == "healthy" {
				currentHealthyCount++
				sourceCopy = cp
			}
		}
	}

	if currentHealthyCount < replicationTargetCount && sourceCopy != nil {
		// Need to replicate! Select an available provider without a copy.
		providers := t.catalog.ListProviders()

		for _, provider := range providers {
			if currentHealthyCount >= replicationTargetCount {
				break
			}

			// Check if provider already has a copy
			hasCopy := false
			for _, cp := range latestVersionCopies {
				if cp.ProviderID == provider.ID {
					hasCopy = true
					break
				}
			}

			if !hasCopy {
				log.Printf("[Tuoni] Creating transfer plan: replicate Object %s to Provider %s", obj.ID, provider.ID)
				t.eventBus.Publish(event.Event{
					ID:        obj.ID + "-" + provider.ID,
					Type:      event.EventTransferPlanCreated,
					Timestamp: time.Now(),
					Payload: map[string]interface{}{
						"objectId":         obj.ID,
						"versionId":        latestVersionID,
						"sourceProviderId": sourceCopy.ProviderID,
						"sourcePath":       sourceCopy.Path,
						"targetProviderId": provider.ID,
					},
				})
				currentHealthyCount++
			}
		}
	}
}

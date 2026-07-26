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
	var hasMigrationPolicy bool
	for _, policy := range t.policies {
		if policy.Type == "replicate" {
			if policy.Target == "object" || (policy.Target == "project" && obj.ProjectID == policy.ID) {
				count, err := strconv.Atoi(policy.Value)
				if err == nil && count > replicationTargetCount {
					replicationTargetCount = count
				}
			}
		} else if policy.Type == "migrate" {
			hasMigrationPolicy = true
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

			// Exclude read-only / index and observe providers from replication targets
			if provider.Capabilities != nil {
				if ctrl, exists := provider.Capabilities["control"]; exists && ctrl == "index_observe" {
					continue
				}
				if ro, exists := provider.Capabilities["read_only"]; exists && ro == "true" {
					continue
				}
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

	// Move files around if there's an active migrate policy
	if hasMigrationPolicy && sourceCopy != nil {
		providers := t.catalog.ListProviders()
		var ssdProviders []*config.StorageProvider
		var hddProviders []*config.StorageProvider

		for _, p := range providers {
			latency := ""
			if p.Capabilities != nil {
				latency = p.Capabilities["latency"]
			}
			isSSD := latency == "low" || (p.Capabilities != nil && p.Capabilities["drive_type"] == "ssd")
			isHDD := latency == "high" || (p.Capabilities != nil && p.Capabilities["drive_type"] == "hdd")

			if isSSD {
				ssdProviders = append(ssdProviders, p)
			} else if isHDD {
				hddProviders = append(hddProviders, p)
			}
		}

		// Check access frequency from metadata
		accessFrequency := "low" // default to low/cold
		if obj.Metadata != nil {
			if freq, exists := obj.Metadata["access_frequency"]; exists {
				if strFreq, ok := freq.(string); ok {
					accessFrequency = strFreq
				}
			} else if count, exists := obj.Metadata["access_count"]; exists {
				var numCount float64
				var hasCount bool
				switch v := count.(type) {
				case int:
					numCount = float64(v)
					hasCount = true
				case float64:
					numCount = v
					hasCount = true
				}
				if hasCount {
					if numCount > 5 {
						accessFrequency = "high"
					} else {
						accessFrequency = "low"
					}
				}
			}
		}

		// Rule 1: High frequency objects should have a copy on an SSD
		if (accessFrequency == "high" || accessFrequency == "hot") && len(ssdProviders) > 0 {
			hasSSDCopy := false
			for _, cp := range latestVersionCopies {
				if cp.State == "healthy" {
					for _, ssd := range ssdProviders {
						if cp.ProviderID == ssd.ID {
							hasSSDCopy = true
							break
						}
					}
				}
			}

			if !hasSSDCopy {
				// Find a target SSD provider that is not read-only
				for _, ssd := range ssdProviders {
					if ssd.Capabilities != nil {
						if ctrl := ssd.Capabilities["control"]; ctrl == "index_observe" {
							continue
						}
						if ro := ssd.Capabilities["read_only"]; ro == "true" {
							continue
						}
					}

					log.Printf("[Tuoni] Hot file tiering: replicate hot Object %s to fast SSD Provider %s", obj.ID, ssd.ID)
					t.eventBus.Publish(event.Event{
						ID:        obj.ID + "-" + ssd.ID + "-tiering",
						Type:      event.EventTransferPlanCreated,
						Timestamp: time.Now(),
						Payload: map[string]interface{}{
							"objectId":         obj.ID,
							"versionId":        latestVersionID,
							"sourceProviderId": sourceCopy.ProviderID,
							"sourcePath":       sourceCopy.Path,
							"targetProviderId": ssd.ID,
						},
					})
					break // target found
				}
			}
		}

		// Rule 2: Low frequency objects should have a copy on an HDD
		if (accessFrequency == "low" || accessFrequency == "cold") && len(hddProviders) > 0 {
			hasHDDCopy := false
			for _, cp := range latestVersionCopies {
				if cp.State == "healthy" {
					for _, hdd := range hddProviders {
						if cp.ProviderID == hdd.ID {
							hasHDDCopy = true
							break
						}
					}
				}
			}

			if !hasHDDCopy {
				// Find a target HDD provider that is not read-only
				for _, hdd := range hddProviders {
					if hdd.Capabilities != nil {
						if ctrl := hdd.Capabilities["control"]; ctrl == "index_observe" {
							continue
						}
						if ro := hdd.Capabilities["read_only"]; ro == "true" {
							continue
						}
					}

					log.Printf("[Tuoni] Cold file tiering: replicate cold Object %s to HDD Provider %s", obj.ID, hdd.ID)
					t.eventBus.Publish(event.Event{
						ID:        obj.ID + "-" + hdd.ID + "-tiering",
						Type:      event.EventTransferPlanCreated,
						Timestamp: time.Now(),
						Payload: map[string]interface{}{
							"objectId":         obj.ID,
							"versionId":        latestVersionID,
							"sourceProviderId": sourceCopy.ProviderID,
							"sourcePath":       sourceCopy.Path,
							"targetProviderId": hdd.ID,
						},
					})
					break // target found
				}
			}
		}
	}
}

package diagnostics

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSessionWritesIncrementallyAndRedactsSecrets(t *testing.T) {
	root := t.TempDir()
	manager := NewManager(root, BuildEnvironment(map[string]any{
		"data_dir":  `C:\Users\Arttu\SAMPO`,
		"api_token": "configuration-secret",
	}))
	ctx := WithCorrelation(context.Background(), "action-test")
	started, err := manager.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	manager.Record(ctx, Event{
		Severity: SeverityInfo, Component: "gateway", Operation: "provider.enroll",
		Phase: "outcome", Outcome: "refused",
		Message: "Bearer message-secret",
		Attributes: map[string]any{
			"path":          `C:\Users\Arttu\Media`,
			"password":      "attribute-secret",
			"endpoint":      "https://alice:private@example.test/bucket?token=query-secret",
			"authorization": "Bearer header-secret",
		},
	})

	beforeStop, err := os.ReadFile(filepath.Join(started.BundlePath, "events.jsonl"))
	if err != nil || !strings.Contains(string(beforeStop), "provider.enroll") {
		t.Fatalf("incremental event log = %q, %v", beforeStop, err)
	}
	if _, err := os.Stat(filepath.Join(started.BundlePath, "summary.txt")); err != nil {
		t.Fatal(err)
	}
	stopped, err := manager.Stop(ctx)
	if err != nil || stopped.Status != "completed" || stopped.Active {
		t.Fatalf("stopped session = %#v, %v", stopped, err)
	}

	var combined strings.Builder
	for _, name := range []string{"events.jsonl", "environment.json", "summary.txt"} {
		content, err := os.ReadFile(filepath.Join(stopped.BundlePath, name))
		if err != nil {
			t.Fatal(err)
		}
		combined.Write(content)
	}
	got := combined.String()
	for _, secret := range []string{"configuration-secret", "message-secret", "attribute-secret", "private", "query-secret", "header-secret"} {
		if strings.Contains(got, secret) {
			t.Fatalf("diagnostic bundle leaked %q: %s", secret, got)
		}
	}
	if !strings.Contains(got, `C:\\Users\\Arttu\\Media`) || !strings.Contains(got, redacted) {
		t.Fatalf("bundle lost necessary path or redaction marker: %s", got)
	}
}

func TestRecordedEventsCarrySessionAndCorrelation(t *testing.T) {
	manager := NewManager(t.TempDir(), BuildEnvironment(nil))
	ctx := WithCorrelation(context.Background(), "action-123")
	info, err := manager.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	manager.Record(ctx, Event{Component: "application", Operation: "provider.scan", Outcome: "succeeded"})
	if _, err := manager.Stop(ctx); err != nil {
		t.Fatal(err)
	}

	file, err := os.Open(filepath.Join(info.BundlePath, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var found bool
	for scanner.Scan() {
		var event RecordedEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatal(err)
		}
		if event.Operation == "provider.scan" {
			found = true
			if event.SessionID != info.SessionID || event.CorrelationID != "action-123" {
				t.Fatalf("uncorrelated event: %#v", event)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("provider.scan event not recorded")
	}
}

func TestCapturePanicCreatesCrashEvidence(t *testing.T) {
	manager := NewManager(t.TempDir(), BuildEnvironment(nil))
	ctx := WithCorrelation(context.Background(), "panic-action")
	info, err := manager.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	manager.CapturePanic(ctx, "gateway", "test.panic", "token=panic-secret", []byte("stack evidence"))
	manager.Interrupt("test interrupted")
	content, err := os.ReadFile(filepath.Join(info.BundlePath, "panic.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "panic-secret") || !strings.Contains(string(content), "stack evidence") {
		t.Fatalf("unsafe or incomplete panic evidence: %s", content)
	}
}

func TestRecorderWriteFailureIsContained(t *testing.T) {
	manager := NewManager(t.TempDir(), BuildEnvironment(nil))
	ctx := WithCorrelation(context.Background(), "failure-action")
	if _, err := manager.Start(ctx); err != nil {
		t.Fatal(err)
	}
	manager.mu.Lock()
	if err := manager.active.log.Close(); err != nil {
		manager.mu.Unlock()
		t.Fatal(err)
	}
	manager.mu.Unlock()

	manager.Record(ctx, Event{Component: "application", Operation: "catalogue.stats", Outcome: "completed"})
	if warning := manager.Status().RecorderWarning; warning == "" {
		t.Fatal("recorder write failure was not contained and reported")
	}
	if _, err := manager.Stop(ctx); err != nil {
		t.Fatalf("recorder failure escaped through session stop: %v", err)
	}
}

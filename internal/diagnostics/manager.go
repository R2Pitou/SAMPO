package diagnostics

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

type Manager struct {
	root        string
	environment Environment
	enabled     atomic.Bool
	mu          sync.Mutex
	active      *sessionRecorder
	last        SessionInfo
}

type sessionRecorder struct {
	id            string
	dir           string
	started       time.Time
	ended         *time.Time
	status        string
	log           *os.File
	events        int
	warnings      int
	errors        int
	lastOperation string
	writeFailure  string
}

func NewManager(root string, environment Environment) *Manager {
	return &Manager{root: root, environment: environment, last: SessionInfo{Status: "inactive"}}
}

func (m *Manager) Enabled() bool { return m != nil && m.enabled.Load() }

func (m *Manager) Start(ctx context.Context) (SessionInfo, error) {
	if m == nil {
		return SessionInfo{}, errorsUnavailable()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active != nil {
		return m.sessionInfoLocked(), ErrAlreadyActive
	}
	started := time.Now().UTC()
	id := started.Format("20060102T150405.000000000Z") + "-" + newID("debug")
	dir := filepath.Join(m.root, id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return SessionInfo{}, fmt.Errorf("create Debug Mode bundle: %w", err)
	}
	logFile, err := os.OpenFile(filepath.Join(dir, "events.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return SessionInfo{}, fmt.Errorf("create Debug Mode event log: %w", err)
	}
	if err := writeJSONFile(filepath.Join(dir, "environment.json"), m.environment); err != nil {
		_ = logFile.Close()
		return SessionInfo{}, fmt.Errorf("write Debug Mode environment: %w", err)
	}
	m.active = &sessionRecorder{id: id, dir: dir, started: started, status: "active", log: logFile}
	m.enabled.Store(true)
	m.writeSummaryLocked()
	ctx = EnsureCorrelation(ctx)
	m.recordLocked(ctx, Event{
		Severity: SeverityInfo, Component: "diagnostics", Operation: "debug.session",
		Phase: "state", Outcome: "started", Message: "Debug Mode session started",
	})
	return m.sessionInfoLocked(), nil
}

func (m *Manager) Stop(ctx context.Context) (SessionInfo, error) {
	if m == nil {
		return SessionInfo{}, ErrNotActive
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active == nil {
		return m.last, ErrNotActive
	}
	ctx = EnsureCorrelation(ctx)
	m.recordLocked(ctx, Event{
		Severity: SeverityInfo, Component: "diagnostics", Operation: "debug.session",
		Phase: "state", Outcome: "completed", Message: "Debug Mode session stopped",
	})
	ended := time.Now().UTC()
	m.active.ended = &ended
	m.active.status = "completed"
	m.writeSummaryLocked()
	_ = m.active.log.Sync()
	_ = m.active.log.Close()
	info := m.sessionInfoLocked()
	m.last = info
	m.active = nil
	m.enabled.Store(false)
	return info, nil
}

func (m *Manager) Interrupt(reason string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active == nil {
		return
	}
	ctx := WithCorrelation(context.Background(), newID("shutdown"))
	m.recordLocked(ctx, Event{
		Severity: SeverityWarn, Component: "diagnostics", Operation: "debug.session",
		Phase: "state", Outcome: "interrupted", Message: reason,
	})
	ended := time.Now().UTC()
	m.active.ended = &ended
	m.active.status = "interrupted"
	m.writeSummaryLocked()
	_ = m.active.log.Sync()
	_ = m.active.log.Close()
	m.last = m.sessionInfoLocked()
	m.active = nil
	m.enabled.Store(false)
}

func (m *Manager) Status() SessionInfo {
	if m == nil {
		return SessionInfo{Status: "inactive"}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active == nil {
		return m.last
	}
	return m.sessionInfoLocked()
}

func (m *Manager) Record(ctx context.Context, event Event) {
	if !m.Enabled() {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active != nil {
		m.recordLocked(ctx, event)
	}
}

func (m *Manager) CapturePanic(ctx context.Context, component, operation string, value any, stack []byte) {
	if !m.Enabled() {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active == nil {
		return
	}
	ctx = EnsureCorrelation(ctx)
	m.recordLocked(ctx, Event{
		Severity: SeverityError, Component: component, Operation: operation,
		Phase: "panic", Outcome: "failed", Message: fmt.Sprint(value),
	})
	content := fmt.Sprintf("SAMPO captured a panic during Debug Mode.\n\nTime: %s\nComponent: %s\nOperation: %s\nCorrelation ID: %s\nPanic: %s\n\n%s",
		time.Now().UTC().Format(time.RFC3339Nano), component, operation, CorrelationID(ctx), redactString(fmt.Sprint(value)), stack)
	if err := writePrivateFile(filepath.Join(m.active.dir, "panic.txt"), []byte(content)); err != nil {
		m.active.writeFailure = err.Error()
	}
}

func (m *Manager) recordLocked(ctx context.Context, event Event) {
	correlation := CorrelationID(ctx)
	if correlation == "" {
		correlation = newID("detached")
	}
	severity := event.Severity
	if severity == "" {
		severity = SeverityInfo
	}
	recorded := RecordedEvent{
		Timestamp: time.Now().UTC(), Severity: severity, Component: event.Component,
		Operation: event.Operation, Phase: event.Phase, Outcome: event.Outcome,
		SessionID: m.active.id, CorrelationID: correlation,
		Message: redactString(event.Message), Attributes: RedactMap(event.Attributes),
	}
	if event.Duration > 0 {
		recorded.DurationMS = float64(event.Duration.Microseconds()) / 1000
	}
	encoded, err := json.Marshal(recorded)
	if err == nil {
		_, err = m.active.log.Write(append(encoded, '\n'))
	}
	if err == nil {
		err = m.active.log.Sync()
	}
	if err != nil {
		m.active.writeFailure = err.Error()
		return
	}
	m.active.events++
	m.active.lastOperation = event.Component + "." + event.Operation
	if severity == SeverityWarn {
		m.active.warnings++
	}
	if severity == SeverityError {
		m.active.errors++
	}
	m.writeSummaryLocked()
}

func (m *Manager) sessionInfoLocked() SessionInfo {
	if m.active == nil {
		return m.last
	}
	started := m.active.started
	return SessionInfo{
		Active: m.active.status == "active", Status: m.active.status, SessionID: m.active.id,
		StartedAt: &started, EndedAt: m.active.ended, BundlePath: m.active.dir,
		RecorderWarning: m.active.writeFailure,
	}
}

func (m *Manager) writeSummaryLocked() {
	if m.active == nil {
		return
	}
	ended := "not yet recorded"
	if m.active.ended != nil {
		ended = m.active.ended.Format(time.RFC3339Nano)
	}
	text := fmt.Sprintf(`SAMPO Debug Mode session

Session ID: %s
Status: %s
Started: %s
Ended: %s
Events: %d
Warnings: %d
Errors: %d
Last operation: %s
Recorder warning: %s

The structured event history is in events.jsonl.
Application and sanitized configuration evidence is in environment.json.
If Status remains active after SAMPO is no longer running, the process was forcibly interrupted.
`, m.active.id, m.active.status, m.active.started.Format(time.RFC3339Nano), ended,
		m.active.events, m.active.warnings, m.active.errors, m.active.lastOperation, m.active.writeFailure)
	if err := writePrivateFile(filepath.Join(m.active.dir, "summary.txt"), []byte(text)); err != nil {
		m.active.writeFailure = err.Error()
	}
}

func writeJSONFile(path string, value any) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writePrivateFile(path, append(encoded, '\n'))
}

func writePrivateFile(path string, content []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(content); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func newID(prefix string) string {
	var buffer [12]byte
	if _, err := rand.Read(buffer[:]); err == nil {
		return prefix + "-" + hex.EncodeToString(buffer[:])
	}
	return fmt.Sprintf("%s-%d", prefix, time.Now().UTC().UnixNano())
}

func errorsUnavailable() error {
	return fmt.Errorf("Debug Mode recorder is unavailable")
}

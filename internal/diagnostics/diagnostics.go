package diagnostics

import (
	"context"
	"errors"
	"runtime"
	"runtime/debug"
	"time"
)

var (
	ErrAlreadyActive = errors.New("Debug Mode is already active")
	ErrNotActive     = errors.New("Debug Mode is not active")
)

type Severity string

const (
	SeverityDebug Severity = "debug"
	SeverityInfo  Severity = "info"
	SeverityWarn  Severity = "warn"
	SeverityError Severity = "error"
)

type Event struct {
	Severity   Severity
	Component  string
	Operation  string
	Phase      string
	Outcome    string
	Message    string
	Duration   time.Duration
	Attributes map[string]any
}

type RecordedEvent struct {
	Timestamp     time.Time      `json:"timestamp"`
	Severity      Severity       `json:"severity"`
	Component     string         `json:"component"`
	Operation     string         `json:"operation"`
	Phase         string         `json:"phase,omitempty"`
	Outcome       string         `json:"outcome,omitempty"`
	SessionID     string         `json:"session_id"`
	CorrelationID string         `json:"correlation_id"`
	DurationMS    float64        `json:"duration_ms,omitempty"`
	Message       string         `json:"message,omitempty"`
	Attributes    map[string]any `json:"attributes,omitempty"`
}

type Environment struct {
	ApplicationVersion string         `json:"application_version"`
	GitCommit          string         `json:"git_commit,omitempty"`
	GitTime            string         `json:"git_time,omitempty"`
	GitModified        bool           `json:"git_modified,omitempty"`
	OperatingSystem    string         `json:"operating_system"`
	Architecture       string         `json:"architecture"`
	GoVersion          string         `json:"go_version"`
	Configuration      map[string]any `json:"configuration"`
}

type SessionInfo struct {
	Active          bool       `json:"active"`
	Status          string     `json:"status"`
	SessionID       string     `json:"sessionId,omitempty"`
	StartedAt       *time.Time `json:"startedAt,omitempty"`
	EndedAt         *time.Time `json:"endedAt,omitempty"`
	BundlePath      string     `json:"bundlePath,omitempty"`
	RecorderWarning string     `json:"recorderWarning,omitempty"`
}

type Sink interface {
	Enabled() bool
	Record(context.Context, Event)
}

type Controller interface {
	Sink
	Start(context.Context) (SessionInfo, error)
	Stop(context.Context) (SessionInfo, error)
	Status() SessionInfo
	CapturePanic(context.Context, string, string, any, []byte)
	Interrupt(string)
}

type NopSink struct{}

func (NopSink) Enabled() bool                 { return false }
func (NopSink) Record(context.Context, Event) {}

type correlationKey struct{}

func WithCorrelation(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationKey{}, id)
}

func CorrelationID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	id, _ := ctx.Value(correlationKey{}).(string)
	return id
}

func EnsureCorrelation(ctx context.Context) context.Context {
	if CorrelationID(ctx) != "" {
		return ctx
	}
	return WithCorrelation(ctx, newID("action"))
}

func BuildEnvironment(configuration map[string]any) Environment {
	environment := Environment{
		ApplicationVersion: "development",
		OperatingSystem:    runtime.GOOS,
		Architecture:       runtime.GOARCH,
		GoVersion:          runtime.Version(),
		Configuration:      RedactMap(configuration),
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			environment.ApplicationVersion = info.Main.Version
		}
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				environment.GitCommit = setting.Value
			case "vcs.time":
				environment.GitTime = setting.Value
			case "vcs.modified":
				environment.GitModified = setting.Value == "true"
			}
		}
	}
	return environment
}

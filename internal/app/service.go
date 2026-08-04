package app

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"sampo/internal/diagnostics"
	"sampo/internal/domain"
	"sampo/internal/observer"
	"sampo/internal/rootidentity"
	"sampo/internal/seshat"
)

type Service struct {
	store   seshat.Catalogue
	scanner observer.Scanner
	roots   rootidentity.Prober
	diag    diagnostics.Sink
	mu      sync.Mutex
	running map[string]bool
}

func New(store seshat.Catalogue, diagnosticSinks ...diagnostics.Sink) *Service {
	return newWithRootProber(store, rootidentity.SystemProber{}, diagnosticSinks...)
}

func newWithRootProber(store seshat.Catalogue, roots rootidentity.Prober, diagnosticSinks ...diagnostics.Sink) *Service {
	var diagnostic diagnostics.Sink
	if len(diagnosticSinks) > 0 {
		diagnostic = diagnosticSinks[0]
	}
	if diagnostic == nil {
		diagnostic = diagnostics.NopSink{}
	}
	return &Service{
		store:   store,
		scanner: observer.Scanner{HashRetries: 2},
		roots:   roots,
		diag:    diagnostic,
		running: make(map[string]bool),
	}
}

func (s *Service) EnrollFilesystem(ctx context.Context, displayName, root string) (domain.Provider, error) {
	ctx, finish := s.operation(ctx, "provider.enroll", map[string]any{"display_name": displayName, "submitted_locator": root})
	var outcomeErr error
	defer func() { finish(outcomeErr, nil) }()
	if root == "" {
		outcomeErr = errors.New("provider root is required")
		return domain.Provider{}, outcomeErr
	}
	probeStarted := time.Now()
	evidence, err := s.roots.Probe(root)
	if err != nil {
		outcomeErr = err
		s.record(ctx, diagnostics.Event{Severity: diagnostics.SeverityWarn, Component: "provider", Operation: "root.probe", Phase: "result", Outcome: "failed", Message: err.Error(), Duration: time.Since(probeStarted), Attributes: map[string]any{"submitted_locator": root}})
		return domain.Provider{}, err
	}
	s.record(ctx, diagnostics.Event{Component: "provider", Operation: "root.probe", Phase: "result", Outcome: "verified", Duration: time.Since(probeStarted), Attributes: map[string]any{"submitted_locator": root, "operational_locator": evidence.OperationalLocator, "identity_confidence": evidence.IdentityConfidence}})
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		displayName = filepath.Base(evidence.OperationalLocator)
		if displayName == "." || displayName == string(filepath.Separator) || displayName == "" {
			displayName = evidence.OperationalLocator
		}
	}
	provider, err := s.store.AddFilesystemProvider(ctx, displayName, evidence)
	outcomeErr = err
	return provider, err
}

func (s *Service) Scan(ctx context.Context, providerID string) (domain.ScanSummary, error) {
	ctx, finish := s.operation(ctx, "provider.scan", map[string]any{"provider_id": providerID})
	var outcomeErr error
	var outcomeAttrs map[string]any
	defer func() { finish(outcomeErr, outcomeAttrs) }()
	s.mu.Lock()
	if s.running[providerID] {
		s.mu.Unlock()
		outcomeErr = errors.New("provider scan is already running")
		return domain.ScanSummary{}, outcomeErr
	}
	s.running[providerID] = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.running, providerID)
		s.mu.Unlock()
	}()

	provider, err := s.store.Provider(ctx, providerID)
	if err != nil {
		outcomeErr = err
		if errors.Is(err, seshat.ErrNotFound) {
			return domain.ScanSummary{}, fmt.Errorf("provider: %w", err)
		}
		return domain.ScanSummary{}, err
	}
	started := time.Now().UTC()
	if err := s.store.BeginScan(ctx, providerID, started); err != nil {
		outcomeErr = err
		return domain.ScanSummary{}, err
	}
	currentRoot, err := s.roots.Probe(provider.SubmittedLocator)
	if err == nil && provider.IdentityConfidence == domain.RootIdentityLegacy {
		err = s.store.EstablishProviderRoot(ctx, providerID, currentRoot)
		if err == nil {
			provider.ProviderRoot = currentRoot
		}
	}
	if err == nil {
		err = rootidentity.Verify(provider.ProviderRoot, currentRoot)
	}
	if err != nil {
		wrapped := fmt.Errorf("verify provider root before scan: %w", err)
		outcomeErr = wrapped
		s.record(ctx, diagnostics.Event{Severity: diagnostics.SeverityWarn, Component: "provider", Operation: "root.verify", Phase: "before_scan", Outcome: "failed", Message: wrapped.Error(), Attributes: map[string]any{"provider_id": providerID}})
		_ = s.store.FailScan(context.WithoutCancel(ctx), providerID, time.Now().UTC(), wrapped)
		return domain.ScanSummary{}, wrapped
	}
	s.record(ctx, diagnostics.Event{Component: "provider", Operation: "root.verify", Phase: "before_scan", Outcome: "verified", Attributes: map[string]any{"provider_id": providerID, "identity_confidence": currentRoot.IdentityConfidence}})
	scanStarted := time.Now()
	result, err := s.scanner.Scan(ctx, currentRoot.OperationalLocator)
	if err != nil {
		outcomeErr = err
		s.record(ctx, diagnostics.Event{Severity: diagnostics.SeverityError, Component: "observer", Operation: "provider.scan", Phase: "result", Outcome: "failed", Message: err.Error(), Duration: time.Since(scanStarted), Attributes: map[string]any{"provider_id": providerID}})
		_ = s.store.FailScan(context.WithoutCancel(ctx), providerID, time.Now().UTC(), err)
		return domain.ScanSummary{}, err
	}
	s.record(ctx, diagnostics.Event{Component: "observer", Operation: "provider.scan", Phase: "result", Outcome: "observed", Duration: time.Since(scanStarted), Attributes: map[string]any{"provider_id": providerID, "observation_count": len(result.Observations), "unstable_count": result.Unstable, "issue_count": len(result.Issues)}})
	afterRoot, err := s.roots.Probe(provider.SubmittedLocator)
	if err == nil {
		err = rootidentity.Verify(provider.ProviderRoot, afterRoot)
	}
	if err != nil {
		wrapped := fmt.Errorf("verify provider root after scan: %w", err)
		outcomeErr = wrapped
		s.record(ctx, diagnostics.Event{Severity: diagnostics.SeverityWarn, Component: "provider", Operation: "root.verify", Phase: "after_scan", Outcome: "failed", Message: wrapped.Error(), Attributes: map[string]any{"provider_id": providerID}})
		_ = s.store.FailScan(context.WithoutCancel(ctx), providerID, time.Now().UTC(), wrapped)
		return domain.ScanSummary{}, wrapped
	}
	s.record(ctx, diagnostics.Event{Component: "provider", Operation: "root.verify", Phase: "after_scan", Outcome: "verified", Attributes: map[string]any{"provider_id": providerID}})
	if err := s.store.ReconcileScan(ctx, providerID, result); err != nil {
		outcomeErr = err
		_ = s.store.FailScan(context.WithoutCancel(ctx), providerID, time.Now().UTC(), err)
		return domain.ScanSummary{}, err
	}
	summary := domain.ScanSummary{
		ProviderID: providerID,
		StartedAt:  result.StartedAt,
		EndedAt:    result.EndedAt,
		Observed:   len(result.Observations),
		Unstable:   result.Unstable,
		Issues:     result.Issues,
	}
	outcomeAttrs = map[string]any{"observed": summary.Observed, "unstable": summary.Unstable, "issues": len(summary.Issues)}
	return summary, nil
}

func (s *Service) Providers(ctx context.Context) ([]domain.Provider, error) {
	ctx, finish := s.operation(ctx, "provider.list", nil)
	providers, err := s.store.Providers(ctx)
	finish(err, map[string]any{"result_count": len(providers)})
	return providers, err
}

func (s *Service) Search(ctx context.Context, query string, limit int) ([]domain.SearchResult, error) {
	query = strings.TrimSpace(query)
	ctx, finish := s.operation(ctx, "catalogue.search", map[string]any{"query_length": len(query), "limit": limit})
	if len(query) > 512 {
		err := errors.New("search query is too long")
		finish(err, nil)
		return nil, err
	}
	results, err := s.store.Search(ctx, query, limit)
	finish(err, map[string]any{"result_count": len(results)})
	return results, err
}

func (s *Service) Stats(ctx context.Context) (domain.CatalogueStats, error) {
	ctx, finish := s.operation(ctx, "catalogue.stats", nil)
	stats, err := s.store.Stats(ctx)
	finish(err, nil)
	return stats, err
}

func (s *Service) operation(ctx context.Context, operation string, attributes map[string]any) (context.Context, func(error, map[string]any)) {
	if !s.diag.Enabled() {
		return ctx, func(error, map[string]any) {}
	}
	ctx = diagnostics.EnsureCorrelation(ctx)
	started := time.Now()
	s.diag.Record(ctx, diagnostics.Event{Component: "application", Operation: operation, Phase: "intent", Outcome: "received", Attributes: attributes})
	return ctx, func(err error, result map[string]any) {
		event := diagnostics.Event{Component: "application", Operation: operation, Phase: "result", Outcome: "completed", Duration: time.Since(started), Attributes: result}
		if err != nil {
			event.Severity = diagnostics.SeverityWarn
			event.Outcome = "failed"
			event.Message = err.Error()
		}
		s.diag.Record(ctx, event)
	}
}

func (s *Service) record(ctx context.Context, event diagnostics.Event) {
	if s.diag.Enabled() {
		s.diag.Record(ctx, event)
	}
}

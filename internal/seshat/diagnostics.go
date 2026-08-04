package seshat

import (
	"context"
	"errors"
	"time"

	"sampo/internal/diagnostics"
	"sampo/internal/domain"
)

// WithDiagnostics narrates Seshat's public semantic boundary without exposing
// persistence details. It deliberately records no SQL, tables, or row IDs.
func WithDiagnostics(inner Catalogue, sink diagnostics.Sink) Catalogue {
	if sink == nil {
		sink = diagnostics.NopSink{}
	}
	return &diagnosticCatalogue{inner: inner, sink: sink}
}

type diagnosticCatalogue struct {
	inner Catalogue
	sink  diagnostics.Sink
}

func (d *diagnosticCatalogue) AddFilesystemProvider(ctx context.Context, name string, root domain.ProviderRoot) (domain.Provider, error) {
	if !d.sink.Enabled() {
		return d.inner.AddFilesystemProvider(ctx, name, root)
	}
	ctx = diagnostics.EnsureCorrelation(ctx)
	started := time.Now()
	d.sink.Record(ctx, diagnostics.Event{Component: "seshat", Operation: "provider.enroll", Phase: "command", Outcome: "received", Attributes: map[string]any{"display_name": name, "submitted_locator": root.SubmittedLocator}})
	provider, err := d.inner.AddFilesystemProvider(ctx, name, root)
	d.outcome(ctx, "provider.enroll", started, err, map[string]any{"provider_id": provider.ID})
	return provider, err
}

func (d *diagnosticCatalogue) EstablishProviderRoot(ctx context.Context, id string, root domain.ProviderRoot) error {
	if !d.sink.Enabled() {
		return d.inner.EstablishProviderRoot(ctx, id, root)
	}
	ctx = diagnostics.EnsureCorrelation(ctx)
	started := time.Now()
	err := d.inner.EstablishProviderRoot(ctx, id, root)
	d.outcome(ctx, "provider.root.establish", started, err, map[string]any{"provider_id": id, "identity_confidence": root.IdentityConfidence})
	return err
}

func (d *diagnosticCatalogue) Provider(ctx context.Context, id string) (domain.Provider, error) {
	if !d.sink.Enabled() {
		return d.inner.Provider(ctx, id)
	}
	ctx = diagnostics.EnsureCorrelation(ctx)
	started := time.Now()
	provider, err := d.inner.Provider(ctx, id)
	d.outcome(ctx, "provider.get", started, err, map[string]any{"provider_id": id})
	return provider, err
}

func (d *diagnosticCatalogue) Providers(ctx context.Context) ([]domain.Provider, error) {
	if !d.sink.Enabled() {
		return d.inner.Providers(ctx)
	}
	ctx = diagnostics.EnsureCorrelation(ctx)
	started := time.Now()
	providers, err := d.inner.Providers(ctx)
	d.outcome(ctx, "provider.list", started, err, map[string]any{"result_count": len(providers)})
	return providers, err
}

func (d *diagnosticCatalogue) BeginScan(ctx context.Context, id string, at time.Time) error {
	if !d.sink.Enabled() {
		return d.inner.BeginScan(ctx, id, at)
	}
	ctx = diagnostics.EnsureCorrelation(ctx)
	started := time.Now()
	err := d.inner.BeginScan(ctx, id, at)
	d.outcome(ctx, "provider.scan.begin", started, err, map[string]any{"provider_id": id})
	return err
}

func (d *diagnosticCatalogue) FailScan(ctx context.Context, id string, at time.Time, cause error) error {
	if !d.sink.Enabled() {
		return d.inner.FailScan(ctx, id, at, cause)
	}
	ctx = diagnostics.EnsureCorrelation(ctx)
	started := time.Now()
	err := d.inner.FailScan(ctx, id, at, cause)
	attributes := map[string]any{"provider_id": id}
	if cause != nil {
		attributes["scan_error"] = cause.Error()
	}
	d.outcome(ctx, "provider.scan.fail", started, err, attributes)
	return err
}

func (d *diagnosticCatalogue) ReconcileScan(ctx context.Context, id string, scan domain.ProviderScan) error {
	if !d.sink.Enabled() {
		return d.inner.ReconcileScan(ctx, id, scan)
	}
	ctx = diagnostics.EnsureCorrelation(ctx)
	started := time.Now()
	err := d.inner.ReconcileScan(ctx, id, scan)
	d.outcome(ctx, "provider.scan.reconcile", started, err, map[string]any{
		"provider_id": id, "observation_count": len(scan.Observations), "unstable_count": scan.Unstable, "issue_count": len(scan.Issues),
	})
	return err
}

func (d *diagnosticCatalogue) Search(ctx context.Context, query string, limit int) ([]domain.SearchResult, error) {
	if !d.sink.Enabled() {
		return d.inner.Search(ctx, query, limit)
	}
	ctx = diagnostics.EnsureCorrelation(ctx)
	started := time.Now()
	results, err := d.inner.Search(ctx, query, limit)
	d.outcome(ctx, "catalogue.search", started, err, map[string]any{"query_length": len(query), "limit": limit, "result_count": len(results)})
	return results, err
}

func (d *diagnosticCatalogue) Stats(ctx context.Context) (domain.CatalogueStats, error) {
	if !d.sink.Enabled() {
		return d.inner.Stats(ctx)
	}
	ctx = diagnostics.EnsureCorrelation(ctx)
	started := time.Now()
	stats, err := d.inner.Stats(ctx)
	d.outcome(ctx, "catalogue.stats", started, err, nil)
	return stats, err
}

func (d *diagnosticCatalogue) outcome(ctx context.Context, operation string, started time.Time, err error, attributes map[string]any) {
	event := diagnostics.Event{Component: "seshat", Operation: operation, Phase: "decision", Outcome: "accepted", Duration: time.Since(started), Attributes: attributes}
	if err != nil {
		event.Severity = diagnostics.SeverityError
		event.Outcome = "failed"
		if errors.Is(err, ErrNotFound) || errors.Is(err, ErrProviderRootDuplicate) || errors.Is(err, ErrProviderRootOverlap) {
			event.Severity = diagnostics.SeverityWarn
			event.Outcome = "refused"
		}
		event.Message = err.Error()
	}
	d.sink.Record(ctx, event)
}

var _ Catalogue = (*diagnosticCatalogue)(nil)

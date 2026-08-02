package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"sampo/internal/catalog"
	"sampo/internal/domain"
	"sampo/internal/observer"
)

type Service struct {
	store   *catalog.Store
	scanner observer.Scanner
	mu      sync.Mutex
	running map[string]bool
}

func New(store *catalog.Store) *Service {
	return &Service{
		store:   store,
		scanner: observer.Scanner{HashRetries: 2},
		running: make(map[string]bool),
	}
}

func (s *Service) EnrollFilesystem(ctx context.Context, displayName, root string) (domain.Provider, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return domain.Provider{}, errors.New("provider root is required")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return domain.Provider{}, fmt.Errorf("resolve provider root: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return domain.Provider{}, fmt.Errorf("resolve provider root: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return domain.Provider{}, fmt.Errorf("inspect provider root: %w", err)
	}
	if !info.IsDir() {
		return domain.Provider{}, errors.New("provider root must be a directory")
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		displayName = filepath.Base(resolved)
		if displayName == "." || displayName == string(filepath.Separator) || displayName == "" {
			displayName = resolved
		}
	}
	return s.store.AddFilesystemProvider(ctx, displayName, filepath.Clean(resolved))
}

func (s *Service) Scan(ctx context.Context, providerID string) (domain.ScanSummary, error) {
	s.mu.Lock()
	if s.running[providerID] {
		s.mu.Unlock()
		return domain.ScanSummary{}, errors.New("provider scan is already running")
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
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ScanSummary{}, errors.New("provider not found")
		}
		return domain.ScanSummary{}, err
	}
	started := time.Now().UTC()
	if err := s.store.BeginScan(ctx, providerID, started); err != nil {
		return domain.ScanSummary{}, err
	}
	result, err := s.scanner.Scan(ctx, provider.RootLocator)
	if err != nil {
		_ = s.store.FailScan(context.WithoutCancel(ctx), providerID, time.Now().UTC(), err)
		return domain.ScanSummary{}, err
	}
	if err := s.store.ReconcileScan(ctx, providerID, result); err != nil {
		_ = s.store.FailScan(context.WithoutCancel(ctx), providerID, time.Now().UTC(), err)
		return domain.ScanSummary{}, err
	}
	return domain.ScanSummary{
		ProviderID: providerID,
		StartedAt:  result.StartedAt,
		EndedAt:    result.EndedAt,
		Observed:   len(result.Observations),
		Unstable:   result.Unstable,
		Issues:     result.Issues,
	}, nil
}

func (s *Service) Providers(ctx context.Context) ([]domain.Provider, error) {
	return s.store.Providers(ctx)
}

func (s *Service) Search(ctx context.Context, query string, limit int) ([]domain.SearchResult, error) {
	query = strings.TrimSpace(query)
	if len(query) > 512 {
		return nil, errors.New("search query is too long")
	}
	return s.store.Search(ctx, query, limit)
}

func (s *Service) Stats(ctx context.Context) (catalog.Stats, error) {
	return s.store.Stats(ctx)
}

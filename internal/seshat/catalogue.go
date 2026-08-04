package seshat

import (
	"context"
	"errors"
	"time"

	"sampo/internal/domain"
)

var (
	ErrNotFound              = errors.New("Seshat record not found")
	ErrProviderRootDuplicate = errors.New("provider root is already enrolled")
	ErrProviderRootOverlap   = errors.New("provider root overlaps an enrolled provider")
)

// Catalogue is Seshat's consumer-facing command and query boundary.
// Implementations own their persistence, transaction, and concurrency details.
type Catalogue interface {
	AddFilesystemProvider(context.Context, string, domain.ProviderRoot) (domain.Provider, error)
	EstablishProviderRoot(context.Context, string, domain.ProviderRoot) error
	Provider(context.Context, string) (domain.Provider, error)
	Providers(context.Context) ([]domain.Provider, error)
	BeginScan(context.Context, string, time.Time) error
	FailScan(context.Context, string, time.Time, error) error
	ReconcileScan(context.Context, string, domain.ProviderScan) error
	Search(context.Context, string, int) ([]domain.SearchResult, error)
	Stats(context.Context) (domain.CatalogueStats, error)
}

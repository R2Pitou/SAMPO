package catalog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"sampo/internal/domain"
	"sampo/internal/seshat"
)

const schemaVersion = 2

type Store struct {
	db *sql.DB
}

var _ seshat.Catalogue = (*Store)(nil)

func Open(ctx context.Context, path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create application data directory: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open catalogue: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	store := &Store{db: db}
	if err := store.configure(ctx); err != nil {
		db.Close()
		return nil, err
	}
	if err := store.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	if err := store.quickCheck(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) configure(ctx context.Context) error {
	pragmas := []string{
		"PRAGMA journal_mode=DELETE",
		"PRAGMA synchronous=FULL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	}
	for _, statement := range pragmas {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("configure catalogue (%s): %w", statement, err)
		}
	}
	return nil
}

func (s *Store) quickCheck(ctx context.Context) error {
	var result string
	if err := s.db.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&result); err != nil {
		return fmt.Errorf("check catalogue integrity: %w", err)
	}
	if result != "ok" {
		return fmt.Errorf("catalogue integrity check failed: %s", result)
	}
	return nil
}

func (s *Store) migrate(ctx context.Context) error {
	var version int
	if err := s.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("read catalogue schema version: %w", err)
	}
	if version > schemaVersion {
		return fmt.Errorf("catalogue schema %d is newer than supported schema %d", version, schemaVersion)
	}
	if version == schemaVersion {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin catalogue migration: %w", err)
	}
	defer tx.Rollback()
	if version < 1 {
		if _, err := tx.ExecContext(ctx, migrationV1); err != nil {
			return fmt.Errorf("apply catalogue migration 1: %w", err)
		}
	}
	if version < 2 {
		if _, err := tx.ExecContext(ctx, migrationV2); err != nil {
			return fmt.Errorf("apply catalogue migration 2: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version=%d", schemaVersion)); err != nil {
		return fmt.Errorf("record catalogue schema version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit catalogue migration: %w", err)
	}
	return nil
}

const migrationV1 = `
CREATE TABLE providers (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL CHECK (kind = 'filesystem'),
    display_name TEXT NOT NULL,
    root_locator TEXT NOT NULL UNIQUE,
    created_at_ns INTEGER NOT NULL,
    last_scan_started_ns INTEGER,
    last_scan_ended_ns INTEGER,
    scan_status TEXT NOT NULL,
    scan_error TEXT NOT NULL DEFAULT ''
);

CREATE TABLE contents (
    id TEXT PRIMARY KEY,
    algorithm TEXT NOT NULL,
    digest_hex TEXT NOT NULL,
    byte_length INTEGER NOT NULL CHECK (byte_length >= 0),
    first_verified_at_ns INTEGER NOT NULL,
    UNIQUE (algorithm, digest_hex, byte_length)
);

CREATE TABLE appearances (
    id TEXT PRIMARY KEY,
    provider_id TEXT NOT NULL REFERENCES providers(id) ON DELETE RESTRICT,
    content_id TEXT NOT NULL REFERENCES contents(id) ON DELETE RESTRICT,
    locator TEXT NOT NULL,
    display_name TEXT NOT NULL,
    native_identity TEXT NOT NULL DEFAULT '',
    byte_length INTEGER NOT NULL CHECK (byte_length >= 0),
    modified_at_ns INTEGER NOT NULL,
    custody TEXT NOT NULL CHECK (custody = 'user-owned'),
    availability TEXT NOT NULL CHECK (availability IN ('available', 'unavailable')),
    continuity TEXT NOT NULL CHECK (continuity IN ('discovered', 'same-locator', 'confirmed-rename', 'probable-rename')),
    observed_at_ns INTEGER NOT NULL
);

CREATE INDEX appearances_content_idx ON appearances(content_id);
CREATE INDEX appearances_native_identity_idx ON appearances(provider_id, native_identity);
CREATE UNIQUE INDEX appearances_available_locator_idx
    ON appearances(provider_id, locator) WHERE availability='available';

CREATE TABLE appearance_events (
    id TEXT PRIMARY KEY,
    appearance_id TEXT NOT NULL REFERENCES appearances(id) ON DELETE RESTRICT,
    kind TEXT NOT NULL,
    old_locator TEXT NOT NULL DEFAULT '',
    new_locator TEXT NOT NULL DEFAULT '',
    observed_at_ns INTEGER NOT NULL
);

CREATE TABLE scan_issues (
    provider_id TEXT NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    locator TEXT NOT NULL,
    code TEXT NOT NULL,
    message TEXT NOT NULL,
    observed_at_ns INTEGER NOT NULL
);
`

const migrationV2 = `
ALTER TABLE providers ADD COLUMN submitted_locator TEXT NOT NULL DEFAULT '';
ALTER TABLE providers ADD COLUMN operational_locator TEXT NOT NULL DEFAULT '';
ALTER TABLE providers ADD COLUMN final_path_evidence TEXT NOT NULL DEFAULT '';
ALTER TABLE providers ADD COLUMN physical_identity TEXT NOT NULL DEFAULT '';
ALTER TABLE providers ADD COLUMN fallback_identity TEXT NOT NULL DEFAULT '';
ALTER TABLE providers ADD COLUMN identity_confidence TEXT NOT NULL DEFAULT 'legacy-unverified';
ALTER TABLE providers ADD COLUMN catalogue_only INTEGER NOT NULL DEFAULT 1 CHECK (catalogue_only IN (0, 1));
UPDATE providers SET submitted_locator=root_locator, operational_locator=root_locator,
    final_path_evidence=root_locator WHERE submitted_locator='';
CREATE UNIQUE INDEX providers_physical_identity_idx
    ON providers(physical_identity)
    WHERE physical_identity != '' AND identity_confidence IN ('strong', 'fallback');
CREATE UNIQUE INDEX providers_fallback_identity_idx
    ON providers(fallback_identity)
    WHERE fallback_identity != '' AND identity_confidence IN ('strong', 'fallback');
`

func (s *Store) AddFilesystemProvider(ctx context.Context, displayName string, root domain.ProviderRoot) (domain.Provider, error) {
	if err := validateProviderRoot(root); err != nil {
		return domain.Provider{}, err
	}
	id, err := domain.NewID()
	if err != nil {
		return domain.Provider{}, err
	}
	now := time.Now().UTC()
	provider := domain.Provider{
		ID: id, Kind: domain.ProviderFilesystem, DisplayName: displayName,
		RootLocator: root.SubmittedLocator, ProviderRoot: root,
		CreatedAt: now, ScanStatus: "never-scanned",
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Provider{}, fmt.Errorf("begin provider enrollment: %w", err)
	}
	defer tx.Rollback()
	if err := rejectRootConflict(ctx, tx, "", root); err != nil {
		return domain.Provider{}, err
	}
	_, err = tx.ExecContext(ctx, `
        INSERT INTO providers(id, kind, display_name, root_locator, submitted_locator,
            operational_locator, final_path_evidence, physical_identity, fallback_identity,
            identity_confidence, catalogue_only, created_at_ns, scan_status)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		provider.ID, provider.Kind, provider.DisplayName, provider.RootLocator,
		root.SubmittedLocator, root.OperationalLocator, root.FinalPathEvidence,
		root.PhysicalIdentity, root.FallbackIdentity, root.IdentityConfidence,
		root.CatalogueOnly, unixNano(now), provider.ScanStatus)
	if err != nil {
		if isSQLiteUniqueConstraint(err) {
			return domain.Provider{}, seshat.ErrProviderRootDuplicate
		}
		return domain.Provider{}, fmt.Errorf("add filesystem provider: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domain.Provider{}, fmt.Errorf("commit provider enrollment: %w", err)
	}
	return provider, nil
}

func (s *Store) EstablishProviderRoot(ctx context.Context, providerID string, root domain.ProviderRoot) error {
	if err := validateProviderRoot(root); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin provider identity establishment: %w", err)
	}
	defer tx.Rollback()
	var confidence string
	if err := tx.QueryRowContext(ctx, `SELECT identity_confidence FROM providers WHERE id=?`, providerID).Scan(&confidence); err != nil {
		return translateSQLiteError(err)
	}
	if confidence != domain.RootIdentityLegacy {
		return errors.New("provider root identity is already established")
	}
	if err := rejectRootConflict(ctx, tx, providerID, root); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE providers SET root_locator=?, submitted_locator=?,
        operational_locator=?, final_path_evidence=?, physical_identity=?, fallback_identity=?,
        identity_confidence=?, catalogue_only=? WHERE id=? AND identity_confidence=?`,
		root.SubmittedLocator, root.SubmittedLocator, root.OperationalLocator,
		root.FinalPathEvidence, root.PhysicalIdentity, root.FallbackIdentity,
		root.IdentityConfidence, root.CatalogueOnly, providerID, domain.RootIdentityLegacy)
	if err != nil {
		if isSQLiteUniqueConstraint(err) {
			return seshat.ErrProviderRootDuplicate
		}
		return fmt.Errorf("establish provider root identity: %w", err)
	}
	if n, _ := result.RowsAffected(); n != 1 {
		return errors.New("provider root identity changed concurrently")
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit provider identity establishment: %w", err)
	}
	return nil
}

func (s *Store) Provider(ctx context.Context, id string) (domain.Provider, error) {
	row := s.db.QueryRowContext(ctx, providerSelect+" WHERE id = ?", id)
	return scanProvider(row)
}

func (s *Store) Providers(ctx context.Context) ([]domain.Provider, error) {
	rows, err := s.db.QueryContext(ctx, providerSelect+" ORDER BY display_name, id")
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}
	defer rows.Close()
	var providers []domain.Provider
	for rows.Next() {
		provider, err := scanProvider(rows)
		if err != nil {
			return nil, err
		}
		providers = append(providers, provider)
	}
	return providers, rows.Err()
}

const providerSelect = `SELECT id, kind, display_name, root_locator,
    submitted_locator, operational_locator, final_path_evidence, physical_identity,
    fallback_identity, identity_confidence, catalogue_only, created_at_ns,
    last_scan_started_ns, last_scan_ended_ns, scan_status, scan_error FROM providers`

type rowScanner interface{ Scan(...any) error }

func scanProvider(row rowScanner) (domain.Provider, error) {
	var p domain.Provider
	var created int64
	var started, ended sql.NullInt64
	if err := row.Scan(&p.ID, &p.Kind, &p.DisplayName, &p.RootLocator,
		&p.SubmittedLocator, &p.OperationalLocator, &p.FinalPathEvidence,
		&p.PhysicalIdentity, &p.FallbackIdentity, &p.IdentityConfidence,
		&p.CatalogueOnly, &created,
		&started, &ended, &p.ScanStatus, &p.ScanError); err != nil {
		return domain.Provider{}, fmt.Errorf("scan provider: %w", translateSQLiteError(err))
	}
	p.RootLocator = p.SubmittedLocator
	p.CreatedAt = fromUnixNano(created)
	if started.Valid {
		v := fromUnixNano(started.Int64)
		p.LastScanStarted = &v
	}
	if ended.Valid {
		v := fromUnixNano(ended.Int64)
		p.LastScanEnded = &v
	}
	return p, nil
}

type providerRootRow struct {
	ID string
	domain.ProviderRoot
}

func validateProviderRoot(root domain.ProviderRoot) error {
	if root.SubmittedLocator == "" || root.OperationalLocator == "" || root.FinalPathEvidence == "" {
		return errors.New("provider root identity evidence is incomplete")
	}
	switch root.IdentityConfidence {
	case domain.RootIdentityStrong, domain.RootIdentityFallback, domain.RootIdentityWeak:
	default:
		return fmt.Errorf("invalid provider root identity confidence %q", root.IdentityConfidence)
	}
	if root.IdentityConfidence != domain.RootIdentityWeak && root.PhysicalIdentity == "" && root.FallbackIdentity == "" {
		return errors.New("provider root physical identity is required")
	}
	if root.IdentityConfidence == domain.RootIdentityWeak && !root.CatalogueOnly {
		return errors.New("weak provider root identity must be catalogue-only")
	}
	return nil
}

func rejectRootConflict(ctx context.Context, tx *sql.Tx, excludedID string, candidate domain.ProviderRoot) error {
	rows, err := tx.QueryContext(ctx, `SELECT id, submitted_locator, operational_locator,
        final_path_evidence, physical_identity, fallback_identity, identity_confidence,
        catalogue_only FROM providers WHERE id != ?`, excludedID)
	if err != nil {
		return fmt.Errorf("load enrolled provider roots: %w", err)
	}
	defer rows.Close()
	var enrolled []providerRootRow
	for rows.Next() {
		var item providerRootRow
		if err := rows.Scan(&item.ID, &item.SubmittedLocator, &item.OperationalLocator,
			&item.FinalPathEvidence, &item.PhysicalIdentity, &item.FallbackIdentity,
			&item.IdentityConfidence, &item.CatalogueOnly); err != nil {
			return fmt.Errorf("scan enrolled provider root: %w", err)
		}
		enrolled = append(enrolled, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, existing := range enrolled {
		if samePhysicalRoot(existing.ProviderRoot, candidate) {
			return fmt.Errorf("%w: provider %s", seshat.ErrProviderRootDuplicate, existing.ID)
		}
		if rootPathsOverlap(existing.FinalPathEvidence, candidate.FinalPathEvidence) {
			return fmt.Errorf("%w: provider %s", seshat.ErrProviderRootOverlap, existing.ID)
		}
	}
	return nil
}

func samePhysicalRoot(a, b domain.ProviderRoot) bool {
	if a.IdentityConfidence == domain.RootIdentityWeak || b.IdentityConfidence == domain.RootIdentityWeak ||
		a.IdentityConfidence == domain.RootIdentityLegacy || b.IdentityConfidence == domain.RootIdentityLegacy {
		return false
	}
	return (a.PhysicalIdentity != "" && a.PhysicalIdentity == b.PhysicalIdentity) ||
		(a.FallbackIdentity != "" && a.FallbackIdentity == b.FallbackIdentity)
}

func rootPathsOverlap(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	windowsStyle := strings.HasPrefix(a, `\\`) || strings.HasPrefix(b, `\\`) ||
		(len(a) >= 2 && a[1] == ':') || (len(b) >= 2 && b[1] == ':')
	separator := "/"
	if windowsStyle {
		a = strings.ReplaceAll(a, "/", `\`)
		b = strings.ReplaceAll(b, "/", `\`)
		separator = `\`
	}
	a = trimRootSeparator(a, separator)
	b = trimRootSeparator(b, separator)
	equal := func(x, y string) bool {
		if windowsStyle {
			return strings.EqualFold(x, y)
		}
		return x == y
	}
	if equal(a, b) {
		return true
	}
	if len(a) < len(b) && equal(b[:len(a)], a) && strings.HasPrefix(b[len(a):], separator) {
		return true
	}
	return len(b) < len(a) && equal(a[:len(b)], b) && strings.HasPrefix(a[len(b):], separator)
}

func trimRootSeparator(path, separator string) string {
	trimmed := strings.TrimRight(path, separator)
	if trimmed == "" {
		return separator
	}
	return trimmed
}

func (s *Store) BeginScan(ctx context.Context, providerID string, started time.Time) error {
	result, err := s.db.ExecContext(ctx, `UPDATE providers
        SET last_scan_started_ns=?, scan_status='scanning', scan_error='' WHERE id=?`, unixNano(started), providerID)
	if err != nil {
		return fmt.Errorf("begin scan: %w", err)
	}
	if n, _ := result.RowsAffected(); n != 1 {
		return seshat.ErrNotFound
	}
	return nil
}

func (s *Store) FailScan(ctx context.Context, providerID string, ended time.Time, scanErr error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin failed-scan update: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE providers
        SET last_scan_ended_ns=?, scan_status='unavailable', scan_error=? WHERE id=?`, unixNano(ended), scanErr.Error(), providerID); err != nil {
		return fmt.Errorf("record failed scan: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE appearances SET availability='unavailable' WHERE provider_id=?`, providerID); err != nil {
		return fmt.Errorf("mark unavailable provider appearances: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit failed-scan update: %w", err)
	}
	return nil
}

func unixNano(t time.Time) int64         { return t.UTC().UnixNano() }
func fromUnixNano(value int64) time.Time { return time.Unix(0, value).UTC() }

func (s *Store) ReconcileScan(ctx context.Context, providerID string, scan domain.ProviderScan) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin reconciliation: %w", err)
	}
	defer tx.Rollback()

	existing, err := loadAppearances(ctx, tx, providerID)
	if err != nil {
		return err
	}
	assignments, err := matchObservations(existing, scan.Observations, scan.EndedAt)
	if err != nil {
		return err
	}

	for _, old := range existing {
		temporary := "__sampo_reconcile__/" + old.ID
		if _, err := tx.ExecContext(ctx, `UPDATE appearances SET locator=?, availability='unavailable' WHERE id=?`, temporary, old.ID); err != nil {
			return fmt.Errorf("prepare appearance reconciliation: %w", err)
		}
	}

	seen := make(map[string]bool, len(assignments))
	for _, assignment := range assignments {
		observation := assignment.observation
		contentID := domain.ContentID(domain.DigestSHA256, observation.DigestHex, observation.ByteLength)
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO contents
            (id, algorithm, digest_hex, byte_length, first_verified_at_ns) VALUES (?, ?, ?, ?, ?)`,
			contentID, domain.DigestSHA256, observation.DigestHex, observation.ByteLength, unixNano(scan.EndedAt)); err != nil {
			return fmt.Errorf("record content: %w", err)
		}

		appearanceID := assignment.appearanceID
		if appearanceID == "" {
			appearanceID, err = domain.NewID()
			if err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO appearances
                (id, provider_id, content_id, locator, display_name, native_identity, byte_length,
                 modified_at_ns, custody, availability, continuity, observed_at_ns)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'available', ?, ?)`,
				appearanceID, providerID, contentID, observation.Locator, observation.DisplayName,
				observation.NativeIdentity, observation.ByteLength, unixNano(observation.ModifiedAt),
				domain.CustodyUserOwned, assignment.continuity, unixNano(scan.EndedAt)); err != nil {
				return fmt.Errorf("record appearance: %w", err)
			}
			if err := insertEvent(ctx, tx, appearanceID, "discovered", "", observation.Locator, scan.EndedAt); err != nil {
				return err
			}
		} else {
			if _, err := tx.ExecContext(ctx, `UPDATE appearances SET
                content_id=?, locator=?, display_name=?, native_identity=?, byte_length=?, modified_at_ns=?,
                custody=?, availability='available', continuity=?, observed_at_ns=? WHERE id=?`,
				contentID, observation.Locator, observation.DisplayName, observation.NativeIdentity,
				observation.ByteLength, unixNano(observation.ModifiedAt), domain.CustodyUserOwned,
				assignment.continuity, unixNano(scan.EndedAt), appearanceID); err != nil {
				return fmt.Errorf("update appearance: %w", err)
			}
			if assignment.oldLocator != "" && assignment.oldLocator != observation.Locator {
				if err := insertEvent(ctx, tx, appearanceID, assignment.continuity, assignment.oldLocator, observation.Locator, scan.EndedAt); err != nil {
					return err
				}
			}
		}
		seen[appearanceID] = true
	}

	for _, old := range existing {
		if seen[old.ID] {
			continue
		}
		if _, err := tx.ExecContext(ctx, `UPDATE appearances SET locator=?, availability='unavailable' WHERE id=?`,
			old.Locator, old.ID); err != nil {
			return fmt.Errorf("mark appearance unavailable: %w", err)
		}
		if old.Availability == "available" {
			if err := insertEvent(ctx, tx, old.ID, "not-observed", old.Locator, "", scan.EndedAt); err != nil {
				return err
			}
		}
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM scan_issues WHERE provider_id=?`, providerID); err != nil {
		return fmt.Errorf("clear scan issues: %w", err)
	}
	for _, issue := range scan.Issues {
		if _, err := tx.ExecContext(ctx, `INSERT INTO scan_issues
            (provider_id, locator, code, message, observed_at_ns) VALUES (?, ?, ?, ?, ?)`,
			providerID, issue.Locator, issue.Code, issue.Message, unixNano(scan.EndedAt)); err != nil {
			return fmt.Errorf("record scan issue: %w", err)
		}
	}
	status := "complete"
	if len(scan.Issues) > 0 {
		status = "complete-with-issues"
	}
	if _, err := tx.ExecContext(ctx, `UPDATE providers SET last_scan_started_ns=?, last_scan_ended_ns=?,
        scan_status=?, scan_error='' WHERE id=?`, unixNano(scan.StartedAt), unixNano(scan.EndedAt), status, providerID); err != nil {
		return fmt.Errorf("complete provider scan: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit reconciliation: %w", err)
	}
	return nil
}

type existingAppearance struct {
	domain.Appearance
	DigestHex string
}

func loadAppearances(ctx context.Context, tx *sql.Tx, providerID string) ([]existingAppearance, error) {
	rows, err := tx.QueryContext(ctx, `SELECT a.id, a.provider_id, a.content_id, a.locator, a.display_name,
        a.native_identity, a.byte_length, a.modified_at_ns, a.custody, a.availability, a.continuity,
        a.observed_at_ns, c.digest_hex
        FROM appearances a JOIN contents c ON c.id=a.content_id WHERE a.provider_id=?`, providerID)
	if err != nil {
		return nil, fmt.Errorf("load appearances: %w", err)
	}
	defer rows.Close()
	var appearances []existingAppearance
	for rows.Next() {
		var item existingAppearance
		var modified, observed int64
		if err := rows.Scan(&item.ID, &item.ProviderID, &item.ContentID, &item.Locator, &item.DisplayName,
			&item.NativeIdentity, &item.ByteLength, &modified, &item.Custody, &item.Availability,
			&item.Continuity, &observed, &item.DigestHex); err != nil {
			return nil, fmt.Errorf("scan appearance: %w", err)
		}
		item.ModifiedAt = fromUnixNano(modified)
		item.ObservedAt = fromUnixNano(observed)
		appearances = append(appearances, item)
	}
	return appearances, rows.Err()
}

type assignment struct {
	observation  domain.FileObservation
	appearanceID string
	oldLocator   string
	continuity   string
}

func matchObservations(existing []existingAppearance, observations []domain.FileObservation, observedAt time.Time) ([]assignment, error) {
	byLocator := make(map[string][]int)
	byIdentity := make(map[string][]int)
	for i, item := range existing {
		byLocator[item.Locator] = append(byLocator[item.Locator], i)
		if item.NativeIdentity != "" {
			byIdentity[item.NativeIdentity] = append(byIdentity[item.NativeIdentity], i)
		}
	}
	usedOld := make(map[int]bool)
	assignments := make([]assignment, len(observations))
	unmatchedNew := make([]int, 0)
	for i, observation := range observations {
		assignments[i] = assignment{observation: observation, continuity: "discovered"}
		if observation.NativeIdentity != "" {
			if oldIndex, ok := uniqueCandidate(existing, byIdentity[observation.NativeIdentity], usedOld, observation); ok {
				old := existing[oldIndex]
				assignments[i].appearanceID = old.ID
				assignments[i].oldLocator = old.Locator
				assignments[i].continuity = "same-locator"
				if old.Locator != observation.Locator {
					assignments[i].continuity = "confirmed-rename"
				}
				usedOld[oldIndex] = true
				continue
			}
		}
		if oldIndex, ok := uniqueCandidate(existing, byLocator[observation.Locator], usedOld, observation); ok {
			old := existing[oldIndex]
			assignments[i].appearanceID = old.ID
			assignments[i].oldLocator = old.Locator
			assignments[i].continuity = "same-locator"
			usedOld[oldIndex] = true
			continue
		}
		unmatchedNew = append(unmatchedNew, i)
	}

	type key struct {
		digest string
		size   int64
	}
	oldGroups := make(map[key][]int)
	newGroups := make(map[key][]int)
	for i, old := range existing {
		if !usedOld[i] && !observedAt.Before(old.ObservedAt) && observedAt.Sub(old.ObservedAt) <= 7*24*time.Hour {
			oldGroups[key{old.DigestHex, old.ByteLength}] = append(oldGroups[key{old.DigestHex, old.ByteLength}], i)
		}
	}
	for _, i := range unmatchedNew {
		observation := observations[i]
		newGroups[key{observation.DigestHex, observation.ByteLength}] = append(newGroups[key{observation.DigestHex, observation.ByteLength}], i)
	}
	for k, newIndexes := range newGroups {
		oldIndexes := oldGroups[k]
		if len(newIndexes) != 1 || len(oldIndexes) != 1 {
			continue
		}
		newIndex, oldIndex := newIndexes[0], oldIndexes[0]
		old := existing[oldIndex]
		assignments[newIndex].appearanceID = old.ID
		assignments[newIndex].oldLocator = old.Locator
		assignments[newIndex].continuity = "probable-rename"
		usedOld[oldIndex] = true
	}
	return assignments, nil
}

func sameContent(old existingAppearance, observation domain.FileObservation) bool {
	return old.DigestHex == observation.DigestHex && old.ByteLength == observation.ByteLength
}

func uniqueCandidate(existing []existingAppearance, candidates []int, used map[int]bool, observation domain.FileObservation) (int, bool) {
	available := make([]int, 0, 1)
	unavailable := make([]int, 0, 1)
	for _, index := range candidates {
		if used[index] || !sameContent(existing[index], observation) {
			continue
		}
		if existing[index].Availability == "available" {
			available = append(available, index)
			continue
		}
		unavailable = append(unavailable, index)
	}
	if len(available) == 1 {
		return available[0], true
	}
	if len(available) > 1 || len(unavailable) != 1 {
		return 0, false
	}
	return unavailable[0], true
}

func insertEvent(ctx context.Context, tx *sql.Tx, appearanceID, kind, oldLocator, newLocator string, at time.Time) error {
	id, err := domain.NewID()
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO appearance_events
        (id, appearance_id, kind, old_locator, new_locator, observed_at_ns) VALUES (?, ?, ?, ?, ?, ?)`,
		id, appearanceID, kind, oldLocator, newLocator, unixNano(at)); err != nil {
		return fmt.Errorf("record appearance event: %w", err)
	}
	return nil
}

func translateSQLiteError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return seshat.ErrNotFound
	}
	return err
}

func isSQLiteUniqueConstraint(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	code := sqliteErr.Code()
	return code == sqlite3.SQLITE_CONSTRAINT_UNIQUE || code == sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY
}

package librarian

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"mash/internal/config"
)

// SQLiteStore manages the SQLite database schema and persistence operations.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore initializes SQLite database, performs migrations, and returns the store.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	store := &SQLiteStore{db: db}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

// Close closes the underlying database.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS providers (
		id TEXT PRIMARY KEY,
		type TEXT,
		path TEXT,
		capabilities TEXT
	);

	CREATE TABLE IF NOT EXISTS projects (
		id TEXT PRIMARY KEY,
		name TEXT,
		policies TEXT,
		metadata TEXT
	);

	CREATE TABLE IF NOT EXISTS objects (
		id TEXT PRIMARY KEY,
		hash TEXT,
		project_id TEXT,
		metadata TEXT
	);

	CREATE TABLE IF NOT EXISTS versions (
		id TEXT PRIMARY KEY,
		object_id TEXT,
		hash TEXT,
		timestamp DATETIME
	);

	CREATE TABLE IF NOT EXISTS copies (
		id TEXT PRIMARY KEY,
		version_id TEXT,
		object_id TEXT,
		provider_id TEXT,
		path TEXT,
		state TEXT,
		verified_at DATETIME
	);
	`
	_, err := s.db.Exec(query)
	return err
}

// SaveProvider persists a storage provider to DB.
func (s *SQLiteStore) SaveProvider(p *config.StorageProvider) error {
	caps, err := json.Marshal(p.Capabilities)
	if err != nil {
		return err
	}

	query := `INSERT OR REPLACE INTO providers (id, type, path, capabilities) VALUES (?, ?, ?, ?)`
	_, err = s.db.Exec(query, p.ID, p.Type, p.Path, string(caps))
	return err
}

// LoadProviders loads all storage providers from DB.
func (s *SQLiteStore) LoadProviders() ([]*config.StorageProvider, error) {
	rows, err := s.db.Query(`SELECT id, type, path, capabilities FROM providers`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []*config.StorageProvider
	for rows.Next() {
		var p config.StorageProvider
		var caps string
		if err := rows.Scan(&p.ID, &p.Type, &p.Path, &caps); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(caps), &p.Capabilities)
		providers = append(providers, &p)
	}
	return providers, nil
}

// SaveProject persists a project to DB.
func (s *SQLiteStore) SaveProject(p *Project) error {
	pols, err := json.Marshal(p.Policies)
	if err != nil {
		return err
	}
	meta, err := json.Marshal(p.Metadata)
	if err != nil {
		return err
	}

	query := `INSERT OR REPLACE INTO projects (id, name, policies, metadata) VALUES (?, ?, ?, ?)`
	_, err = s.db.Exec(query, p.ID, p.Name, string(pols), string(meta))
	return err
}

// LoadProjects loads all projects from DB.
func (s *SQLiteStore) LoadProjects() ([]*Project, error) {
	rows, err := s.db.Query(`SELECT id, name, policies, metadata FROM projects`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		var p Project
		var pols, meta string
		if err := rows.Scan(&p.ID, &p.Name, &pols, &meta); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(pols), &p.Policies)
		_ = json.Unmarshal([]byte(meta), &p.Metadata)
		projects = append(projects, &p)
	}
	return projects, nil
}

// SaveObject persists an Object, its Versions, and Copies transactionally.
func (s *SQLiteStore) SaveObject(obj *Object) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	meta, err := json.Marshal(obj.Metadata)
	if err != nil {
		return err
	}

	// 1. Insert/Replace Object
	_, err = tx.Exec(`INSERT OR REPLACE INTO objects (id, hash, project_id, metadata) VALUES (?, ?, ?, ?)`,
		obj.ID, obj.Hash, obj.ProjectID, string(meta))
	if err != nil {
		return err
	}

	// 2. Insert/Replace Versions & Copies
	for _, ver := range obj.Versions {
		_, err = tx.Exec(`INSERT OR REPLACE INTO versions (id, object_id, hash, timestamp) VALUES (?, ?, ?, ?)`,
			ver.ID, obj.ID, ver.Hash, ver.Timestamp)
		if err != nil {
			return err
		}

		for _, cp := range ver.Copies {
			_, err = tx.Exec(`INSERT OR REPLACE INTO copies (id, version_id, object_id, provider_id, path, state, verified_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
				cp.ID, ver.ID, obj.ID, cp.ProviderID, cp.Path, cp.State, cp.VerifiedAt)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// LoadObjects loads all Objects with nested Versions and Copies from DB.
func (s *SQLiteStore) LoadObjects() ([]*Object, error) {
	rows, err := s.db.Query(`SELECT id, hash, project_id, metadata FROM objects`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var objects []*Object
	for rows.Next() {
		var obj Object
		var meta string
		if err := rows.Scan(&obj.ID, &obj.Hash, &obj.ProjectID, &meta); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(meta), &obj.Metadata)

		// Load Versions
		vRows, err := s.db.Query(`SELECT id, hash, timestamp FROM versions WHERE object_id = ?`, obj.ID)
		if err != nil {
			return nil, err
		}

		for vRows.Next() {
			var ver Version
			if err := vRows.Scan(&ver.ID, &ver.Hash, &ver.Timestamp); err != nil {
				vRows.Close()
				return nil, err
			}

			// Load Copies
			cRows, err := s.db.Query(`SELECT id, provider_id, path, state, verified_at FROM copies WHERE version_id = ?`, ver.ID)
			if err != nil {
				vRows.Close()
				return nil, err
			}

			for cRows.Next() {
				var cp Copy
				if err := cRows.Scan(&cp.ID, &cp.ProviderID, &cp.Path, &cp.State, &cp.VerifiedAt); err != nil {
					cRows.Close()
					vRows.Close()
					return nil, err
				}
				ver.Copies = append(ver.Copies, cp)
			}
			cRows.Close()
			obj.Versions = append(obj.Versions, ver)
		}
		vRows.Close()
		objects = append(objects, &obj)
	}

	return objects, nil
}

// UpdateCopyState persists the copy state change to SQLite.
func (s *SQLiteStore) UpdateCopyState(objID string, versionID string, copyID string, newState string) {
	_, err := s.db.Exec(`UPDATE copies SET state = ?, verified_at = ? WHERE id = ? AND version_id = ? AND object_id = ?`,
		newState, time.Now(), copyID, versionID, objID)
	if err != nil {
		log.Printf("[SQLiteStore] Error updating copy state in DB: %v", err)
	}
}

// Package store holds the emulator's state: in-memory cluster/job state,
// and a SQLite-backed metastore for Unity Catalog objects.
package store

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/marcboeker/go-duckdb/v2"
	_ "modernc.org/sqlite"
)

type ClusterState struct {
	ClusterID   string `json:"cluster_id"`
	ClusterName string `json:"cluster_name"`
	State       string `json:"state"`
}

type RunState struct {
	RunID          int64  `json:"run_id"`
	LifeCycleState string `json:"life_cycle_state"`
	ResultState    string `json:"result_state"`
}

type JobDef struct {
	JobID      int64  `json:"job_id"`
	Name       string `json:"name"`
	ScriptPath string `json:"-"`
}

// Store aggregates all emulator state.
type Store struct {
	db          *sql.DB // SQLite metastore: catalogs/schemas/tables/acl/tags/audit/lineage
	warehouseDB *sql.DB // DuckDB: actual query execution for the SQL warehouse emulation

	mu         sync.RWMutex
	clusters   map[string]*ClusterState
	runs       map[int64]*RunState
	jobs       map[int64]*JobDef
	statements map[string]*StatementResult
	nextRun    int64
	nextJob    int64
}

// New opens (or creates) the SQLite metastore at metaPath and the DuckDB
// warehouse at warehousePath, and runs metastore migrations.
func New(metaPath, warehousePath string) (*Store, error) {
	db, err := sql.Open("sqlite", metaPath)
	if err != nil {
		return nil, fmt.Errorf("open metastore: %w", err)
	}
	db.SetMaxOpenConns(1) // modernc.org/sqlite: keep writes serialized

	warehouseDB, err := sql.Open("duckdb", warehousePath)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("open warehouse: %w", err)
	}

	s := &Store{
		db:          db,
		warehouseDB: warehouseDB,
		clusters:    make(map[string]*ClusterState),
		runs:        make(map[int64]*RunState),
		jobs:        make(map[int64]*JobDef),
		statements:  make(map[string]*StatementResult),
	}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	warehouseErr := s.warehouseDB.Close()
	if err := s.db.Close(); err != nil {
		return err
	}
	return warehouseErr
}

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS catalogs (
			name TEXT PRIMARY KEY,
			comment TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS schemas (
			catalog_name TEXT NOT NULL,
			name TEXT NOT NULL,
			comment TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (catalog_name, name)
		)`,
		`CREATE TABLE IF NOT EXISTS tables (
			catalog_name TEXT NOT NULL,
			schema_name TEXT NOT NULL,
			name TEXT NOT NULL,
			table_type TEXT NOT NULL DEFAULT 'MANAGED',
			data_source_format TEXT NOT NULL DEFAULT 'DELTA',
			storage_location TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (catalog_name, schema_name, name)
		)`,
		// Governance (see docs/API.md — Unity Catalog governance)
		`CREATE TABLE IF NOT EXISTS acl (
			principal TEXT NOT NULL,
			securable TEXT NOT NULL,
			privilege TEXT NOT NULL,
			PRIMARY KEY (principal, securable, privilege)
		)`,
		`CREATE TABLE IF NOT EXISTS tags (
			securable TEXT NOT NULL,
			tag TEXT NOT NULL,
			value TEXT,
			PRIMARY KEY (securable, tag)
		)`,
		`CREATE TABLE IF NOT EXISTS audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			principal TEXT,
			action TEXT,
			securable TEXT,
			result TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS lineage (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			job_id TEXT,
			run_id INTEGER,
			input_securable TEXT,
			output_securable TEXT,
			ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

// --- Clusters (in-memory, per spec: never a real Spark cluster) ---

func (s *Store) CreateCluster(name string) *ClusterState {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := &ClusterState{
		ClusterID:   fmt.Sprintf("local-cluster-%d", time.Now().UnixNano()),
		ClusterName: name,
		State:       "RUNNING",
	}
	s.clusters[c.ClusterID] = c
	return c
}

func (s *Store) GetCluster(id string) (*ClusterState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.clusters[id]
	return c, ok
}

func (s *Store) DeleteCluster(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.clusters[id]; !ok {
		return false
	}
	s.clusters[id].State = "TERMINATED"
	return true
}

// --- Jobs / runs (in-memory) ---

func (s *Store) CreateJob(name, scriptPath string) *JobDef {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextJob++
	j := &JobDef{JobID: s.nextJob, Name: name, ScriptPath: scriptPath}
	s.jobs[j.JobID] = j
	return j
}

func (s *Store) GetJob(jobID int64) (*JobDef, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[jobID]
	return j, ok
}

func (s *Store) NewRun() *RunState {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextRun++
	r := &RunState{
		RunID:          s.nextRun,
		LifeCycleState: "RUNNING",
	}
	s.runs[r.RunID] = r
	return r
}

func (s *Store) SetRunResult(runID int64, lifeCycle, result string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.runs[runID]; ok {
		r.LifeCycleState = lifeCycle
		r.ResultState = result
	}
}

func (s *Store) GetRun(runID int64) (*RunState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.runs[runID]
	return r, ok
}

// --- Unity Catalog (SQLite-backed registry) ---

func (s *Store) CreateCatalog(name, comment string) error {
	_, err := s.db.Exec(`INSERT INTO catalogs (name, comment) VALUES (?, ?)`, name, comment)
	return err
}

func (s *Store) CreateSchema(catalog, name, comment string) error {
	_, err := s.db.Exec(`INSERT INTO schemas (catalog_name, name, comment) VALUES (?, ?, ?)`, catalog, name, comment)
	return err
}

func (s *Store) CreateTable(catalog, schema, name, format, location string) error {
	_, err := s.db.Exec(
		`INSERT INTO tables (catalog_name, schema_name, name, data_source_format, storage_location) VALUES (?, ?, ?, ?, ?)`,
		catalog, schema, name, format, location,
	)
	return err
}

type TableInfo struct {
	Catalog  string `json:"catalog_name"`
	Schema   string `json:"schema_name"`
	Name     string `json:"name"`
	Format   string `json:"data_source_format"`
	Location string `json:"storage_location"`
}

func (s *Store) ListTables(catalog, schema string) ([]TableInfo, error) {
	rows, err := s.db.Query(
		`SELECT catalog_name, schema_name, name, data_source_format, storage_location
		 FROM tables WHERE catalog_name = ? AND schema_name = ?`,
		catalog, schema,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []TableInfo
	for rows.Next() {
		var t TableInfo
		if err := rows.Scan(&t.Catalog, &t.Schema, &t.Name, &t.Format, &t.Location); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

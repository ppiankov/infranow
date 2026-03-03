package history

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// SQLite pragmas and operational constants
const (
	busyTimeoutMS = 5000
	dbFileMode    = 0o700
)

// SQLiteStore implements Store using a local SQLite database
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens or creates a SQLite history database at the given path.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	// Create parent directory
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, dbFileMode); err != nil {
		return nil, fmt.Errorf("create history directory: %w", err)
	}

	dsn := fmt.Sprintf("file:%s?_busy_timeout=%d", dbPath, busyTimeoutMS)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open history database: %w", err)
	}

	// Enable WAL mode for concurrent read/write
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf("set WAL mode: %w (also failed to close: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	// Create schema
	if err := createSchema(db); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf("create schema: %w (also failed to close: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("create schema: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

func createSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS problem_history (
		fingerprint      TEXT PRIMARY KEY,
		type             TEXT NOT NULL,
		entity           TEXT NOT NULL,
		namespace        TEXT NOT NULL DEFAULT '',
		severity         TEXT NOT NULL,
		first_seen       INTEGER NOT NULL,
		last_seen        INTEGER NOT NULL,
		occurrence_count INTEGER NOT NULL DEFAULT 1
	);
	CREATE INDEX IF NOT EXISTS idx_history_last_seen ON problem_history(last_seen);
	`
	_, err := db.Exec(schema)
	return err
}

// Upsert inserts or updates history records in a single transaction.
func (s *SQLiteStore) Upsert(ctx context.Context, records []Record) error {
	if len(records) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin upsert transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // No-op after commit

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO problem_history (fingerprint, type, entity, namespace, severity, first_seen, last_seen, occurrence_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1)
		ON CONFLICT(fingerprint) DO UPDATE SET
			severity = excluded.severity,
			last_seen = excluded.last_seen,
			occurrence_count = occurrence_count + 1
	`)
	if err != nil {
		return fmt.Errorf("prepare upsert: %w", err)
	}
	defer func() {
		_ = stmt.Close() // Best-effort
	}()

	for i := range records {
		r := &records[i]
		_, err := stmt.ExecContext(ctx,
			r.Fingerprint,
			r.Type,
			r.Entity,
			r.Namespace,
			r.Severity,
			r.FirstSeen.Unix(),
			r.LastSeen.Unix(),
		)
		if err != nil {
			return fmt.Errorf("upsert record %s: %w", r.Fingerprint, err)
		}
	}

	return tx.Commit()
}

// Lookup retrieves history records for the given fingerprints.
func (s *SQLiteStore) Lookup(ctx context.Context, fingerprints []string) (map[string]*Record, error) {
	if len(fingerprints) == 0 {
		return make(map[string]*Record), nil
	}

	placeholders := make([]string, len(fingerprints))
	args := make([]interface{}, len(fingerprints))
	for i, fp := range fingerprints {
		placeholders[i] = "?"
		args[i] = fp
	}

	query := fmt.Sprintf(
		"SELECT fingerprint, type, entity, namespace, severity, first_seen, last_seen, occurrence_count FROM problem_history WHERE fingerprint IN (%s)",
		strings.Join(placeholders, ","),
	)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("lookup query: %w", err)
	}
	defer func() {
		_ = rows.Close() // Best-effort
	}()

	result := make(map[string]*Record)
	for rows.Next() {
		var r Record
		var firstSeen, lastSeen int64
		if err := rows.Scan(&r.Fingerprint, &r.Type, &r.Entity, &r.Namespace, &r.Severity, &firstSeen, &lastSeen, &r.OccurrenceCount); err != nil {
			return nil, fmt.Errorf("scan record: %w", err)
		}
		r.FirstSeen = time.Unix(firstSeen, 0)
		r.LastSeen = time.Unix(lastSeen, 0)
		result[r.Fingerprint] = &r
	}

	return result, rows.Err()
}

// List returns history records matching the given options, ordered by last_seen descending.
func (s *SQLiteStore) List(ctx context.Context, opts ListOpts) ([]Record, error) {
	var conditions []string
	var args []interface{}

	if !opts.Since.IsZero() {
		conditions = append(conditions, "last_seen >= ?")
		args = append(args, opts.Since.Unix())
	}
	if opts.MinSeverity != "" {
		conditions = append(conditions, "severity = ?")
		args = append(args, opts.MinSeverity)
	}

	query := "SELECT fingerprint, type, entity, namespace, severity, first_seen, last_seen, occurrence_count FROM problem_history"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY last_seen DESC"

	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}
	query += fmt.Sprintf(" LIMIT %d", limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list query: %w", err)
	}
	defer func() {
		_ = rows.Close() // Best-effort
	}()

	var records []Record
	for rows.Next() {
		var r Record
		var firstSeen, lastSeen int64
		if err := rows.Scan(&r.Fingerprint, &r.Type, &r.Entity, &r.Namespace, &r.Severity, &firstSeen, &lastSeen, &r.OccurrenceCount); err != nil {
			return nil, fmt.Errorf("scan record: %w", err)
		}
		r.FirstSeen = time.Unix(firstSeen, 0)
		r.LastSeen = time.Unix(lastSeen, 0)
		records = append(records, r)
	}

	return records, rows.Err()
}

// Prune deletes history entries older than the given duration. Returns the number of deleted rows.
func (s *SQLiteStore) Prune(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan).Unix()
	result, err := s.db.ExecContext(ctx, "DELETE FROM problem_history WHERE last_seen < ?", cutoff)
	if err != nil {
		return 0, fmt.Errorf("prune history: %w", err)
	}
	return result.RowsAffected()
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

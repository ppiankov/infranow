package history

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DefaultRetention is the auto-prune threshold for old history entries
const DefaultRetention = 90 * 24 * time.Hour

// Store defines the interface for problem history persistence
type Store interface {
	Upsert(ctx context.Context, records []Record) error
	Lookup(ctx context.Context, fingerprints []string) (map[string]*Record, error)
	List(ctx context.Context, opts ListOpts) ([]Record, error)
	Prune(ctx context.Context, olderThan time.Duration) (int64, error)
	Close() error
}

// Record represents a persisted history row
type Record struct {
	Fingerprint     string
	Type            string
	Entity          string
	Namespace       string
	Severity        string
	FirstSeen       time.Time
	LastSeen        time.Time
	OccurrenceCount int64
}

// ListOpts controls filtering and pagination for List queries
type ListOpts struct {
	Since       time.Time
	MinSeverity string
	Limit       int
}

// Fingerprint generates a stable identifier for a problem across sessions.
// It hashes length-prefixed fields to prevent collisions when field values overlap.
func Fingerprint(problemType, entity, namespace string) string {
	h := sha256.New()
	fmt.Fprintf(h, "%d:%s\n%d:%s\n%d:%s", len(problemType), problemType, len(entity), entity, len(namespace), namespace) // Best-effort: hash.Write never errors
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// DefaultDBPath returns the default history database path following XDG conventions.
func DefaultDBPath() (string, error) {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "infranow", "history.db"), nil
}

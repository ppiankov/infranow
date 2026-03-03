package history

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("close store: %v", err)
		}
	})
	return store
}

func TestFingerprint_Stable(t *testing.T) {
	fp1 := Fingerprint("oom_kill", "production/payment-api", "production")
	fp2 := Fingerprint("oom_kill", "production/payment-api", "production")
	if fp1 != fp2 {
		t.Errorf("fingerprint not stable: %q != %q", fp1, fp2)
	}
	if len(fp1) != 16 {
		t.Errorf("fingerprint length = %d, want 16", len(fp1))
	}
}

func TestFingerprint_CollisionResistance(t *testing.T) {
	tests := []struct {
		name      string
		a, b      [3]string // type, entity, namespace
		wantEqual bool
	}{
		{
			name: "different type",
			a:    [3]string{"oom_kill", "pod-1", "ns"},
			b:    [3]string{"crashloop", "pod-1", "ns"},
		},
		{
			name: "different entity",
			a:    [3]string{"oom_kill", "pod-1", "ns"},
			b:    [3]string{"oom_kill", "pod-2", "ns"},
		},
		{
			name: "different namespace",
			a:    [3]string{"oom_kill", "pod-1", "ns1"},
			b:    [3]string{"oom_kill", "pod-1", "ns2"},
		},
		{
			name: "separator collision",
			a:    [3]string{"a", "b\x00c", "d"},
			b:    [3]string{"a\x00b", "c", "d"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fpA := Fingerprint(tt.a[0], tt.a[1], tt.a[2])
			fpB := Fingerprint(tt.b[0], tt.b[1], tt.b[2])
			if fpA == fpB && !tt.wantEqual {
				t.Errorf("fingerprint collision: %q == %q", fpA, fpB)
			}
		})
	}
}

func TestUpsert_NewRecord(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	records := []Record{
		{
			Fingerprint: "abc123",
			Type:        "oom_kill",
			Entity:      "production/payment-api",
			Namespace:   "production",
			Severity:    "FATAL",
			FirstSeen:   now,
			LastSeen:    now,
		},
	}

	if err := store.Upsert(ctx, records); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	result, err := store.Lookup(ctx, []string{"abc123"})
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}

	rec, ok := result["abc123"]
	if !ok {
		t.Fatal("record not found after upsert")
	}
	if rec.Type != "oom_kill" {
		t.Errorf("Type = %q, want %q", rec.Type, "oom_kill")
	}
	if rec.Entity != "production/payment-api" {
		t.Errorf("Entity = %q, want %q", rec.Entity, "production/payment-api")
	}
	if rec.OccurrenceCount != 1 {
		t.Errorf("OccurrenceCount = %d, want 1", rec.OccurrenceCount)
	}
	if !rec.FirstSeen.Equal(now) {
		t.Errorf("FirstSeen = %v, want %v", rec.FirstSeen, now)
	}
}

func TestUpsert_IncrementCount(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	later := now.Add(5 * time.Minute)

	// First upsert
	if err := store.Upsert(ctx, []Record{
		{Fingerprint: "abc123", Type: "oom_kill", Entity: "pod-1", Severity: "FATAL", FirstSeen: now, LastSeen: now},
	}); err != nil {
		t.Fatalf("first Upsert: %v", err)
	}

	// Second upsert — should increment count, update last_seen, preserve first_seen
	if err := store.Upsert(ctx, []Record{
		{Fingerprint: "abc123", Type: "oom_kill", Entity: "pod-1", Severity: "CRITICAL", FirstSeen: later, LastSeen: later},
	}); err != nil {
		t.Fatalf("second Upsert: %v", err)
	}

	result, err := store.Lookup(ctx, []string{"abc123"})
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}

	rec := result["abc123"]
	if rec.OccurrenceCount != 2 {
		t.Errorf("OccurrenceCount = %d, want 2", rec.OccurrenceCount)
	}
	if !rec.FirstSeen.Equal(now) {
		t.Errorf("FirstSeen changed: got %v, want %v (should be preserved)", rec.FirstSeen, now)
	}
	if !rec.LastSeen.Equal(later) {
		t.Errorf("LastSeen = %v, want %v", rec.LastSeen, later)
	}
	if rec.Severity != "CRITICAL" {
		t.Errorf("Severity = %q, want %q (should be updated)", rec.Severity, "CRITICAL")
	}
}

func TestUpsert_Empty(t *testing.T) {
	store := newTestStore(t)
	if err := store.Upsert(context.Background(), nil); err != nil {
		t.Fatalf("Upsert(nil): %v", err)
	}
}

func TestLookup_Missing(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	result, err := store.Lookup(ctx, []string{"nonexistent"})
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d records", len(result))
	}
}

func TestLookup_Empty(t *testing.T) {
	store := newTestStore(t)
	result, err := store.Lookup(context.Background(), nil)
	if err != nil {
		t.Fatalf("Lookup(nil): %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d", len(result))
	}
}

func TestLookup_Batch(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	records := []Record{
		{Fingerprint: "fp1", Type: "oom_kill", Entity: "pod-1", Severity: "FATAL", FirstSeen: now, LastSeen: now},
		{Fingerprint: "fp2", Type: "crashloop", Entity: "pod-2", Severity: "CRITICAL", FirstSeen: now, LastSeen: now},
		{Fingerprint: "fp3", Type: "disk_full", Entity: "node-1", Severity: "WARNING", FirstSeen: now, LastSeen: now},
	}
	if err := store.Upsert(ctx, records); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	result, err := store.Lookup(ctx, []string{"fp1", "fp3", "fp_missing"})
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 results, got %d", len(result))
	}
	if _, ok := result["fp1"]; !ok {
		t.Error("fp1 not found")
	}
	if _, ok := result["fp3"]; !ok {
		t.Error("fp3 not found")
	}
}

func TestList_All(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	records := []Record{
		{Fingerprint: "fp1", Type: "oom_kill", Entity: "pod-1", Severity: "FATAL", FirstSeen: now, LastSeen: now},
		{Fingerprint: "fp2", Type: "crashloop", Entity: "pod-2", Severity: "CRITICAL", FirstSeen: now.Add(-time.Hour), LastSeen: now.Add(-time.Hour)},
	}
	if err := store.Upsert(ctx, records); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	list, err := store.List(ctx, ListOpts{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 records, got %d", len(list))
	}
	// Should be ordered by last_seen DESC
	if list[0].Fingerprint != "fp1" {
		t.Errorf("first record should be fp1 (most recent), got %s", list[0].Fingerprint)
	}
}

func TestList_WithSinceFilter(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	records := []Record{
		{Fingerprint: "fp1", Type: "oom_kill", Entity: "pod-1", Severity: "FATAL", FirstSeen: now, LastSeen: now},
		{Fingerprint: "fp2", Type: "crashloop", Entity: "pod-2", Severity: "CRITICAL", FirstSeen: now.Add(-48 * time.Hour), LastSeen: now.Add(-48 * time.Hour)},
	}
	if err := store.Upsert(ctx, records); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	list, err := store.List(ctx, ListOpts{Since: now.Add(-24 * time.Hour)})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 record, got %d", len(list))
	}
}

func TestList_WithLimit(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	records := []Record{
		{Fingerprint: "fp1", Type: "a", Entity: "e1", Severity: "FATAL", FirstSeen: now, LastSeen: now},
		{Fingerprint: "fp2", Type: "b", Entity: "e2", Severity: "WARNING", FirstSeen: now, LastSeen: now},
		{Fingerprint: "fp3", Type: "c", Entity: "e3", Severity: "CRITICAL", FirstSeen: now, LastSeen: now},
	}
	if err := store.Upsert(ctx, records); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	list, err := store.List(ctx, ListOpts{Limit: 2})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 records, got %d", len(list))
	}
}

func TestList_WithSeverityFilter(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	records := []Record{
		{Fingerprint: "fp1", Type: "a", Entity: "e1", Severity: "FATAL", FirstSeen: now, LastSeen: now},
		{Fingerprint: "fp2", Type: "b", Entity: "e2", Severity: "WARNING", FirstSeen: now, LastSeen: now},
	}
	if err := store.Upsert(ctx, records); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	list, err := store.List(ctx, ListOpts{MinSeverity: "FATAL"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 record, got %d", len(list))
	}
	if list[0].Severity != "FATAL" {
		t.Errorf("Severity = %q, want FATAL", list[0].Severity)
	}
}

func TestPrune(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	records := []Record{
		{Fingerprint: "recent", Type: "a", Entity: "e1", Severity: "FATAL", FirstSeen: now, LastSeen: now},
		{Fingerprint: "old", Type: "b", Entity: "e2", Severity: "WARNING", FirstSeen: now.Add(-100 * 24 * time.Hour), LastSeen: now.Add(-100 * 24 * time.Hour)},
	}
	if err := store.Upsert(ctx, records); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	deleted, err := store.Prune(ctx, 90*24*time.Hour)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}

	// Verify only recent remains
	list, err := store.List(ctx, ListOpts{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 record, got %d", len(list))
	}
	if list[0].Fingerprint != "recent" {
		t.Errorf("remaining record = %q, want recent", list[0].Fingerprint)
	}
}

func TestDefaultDBPath(t *testing.T) {
	p, err := DefaultDBPath()
	if err != nil {
		t.Fatalf("DefaultDBPath: %v", err)
	}
	if p == "" {
		t.Error("DefaultDBPath returned empty string")
	}
	if filepath.Base(p) != "history.db" {
		t.Errorf("expected history.db, got %s", filepath.Base(p))
	}
}

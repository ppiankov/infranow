package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/ppiankov/infranow/internal/history"
	"github.com/ppiankov/infranow/internal/util"
)

// Duration flag parsing constants
const (
	defaultPruneAge = "90d"
	defaultListAge  = "7d"
)

var (
	historyListSince    string
	historyListSeverity string
	historyListLimit    int
	historyListOutput   string
	historyPruneAge     string
	historyPruneDryRun  bool
)

// NewHistoryCommand creates the history subcommand
func NewHistoryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history",
		Short: "Manage problem history database",
		Long:  `View and manage the local SQLite database that tracks problem recurrence across sessions.`,
	}

	cmd.AddCommand(newHistoryListCommand())
	cmd.AddCommand(newHistoryPruneCommand())
	return cmd
}

func newHistoryListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List historical problems",
		RunE:  runHistoryList,
	}
	cmd.Flags().StringVar(&historyListSince, "since", defaultListAge, "Show problems seen since (e.g. 7d, 24h)")
	cmd.Flags().StringVar(&historyListSeverity, "min-severity", "", "Filter by severity (WARNING, CRITICAL, FATAL)")
	cmd.Flags().IntVar(&historyListLimit, "limit", 100, "Maximum number of records")
	cmd.Flags().StringVar(&historyListOutput, "output", "text", "Output format (text, json)")
	return cmd
}

func newHistoryPruneCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove old history entries",
		RunE:  runHistoryPrune,
	}
	cmd.Flags().StringVar(&historyPruneAge, "older-than", defaultPruneAge, "Remove entries older than (e.g. 90d, 720h)")
	cmd.Flags().BoolVar(&historyPruneDryRun, "dry-run", false, "Show count without deleting")
	return cmd
}

func openHistoryStore() (*history.SQLiteStore, error) {
	dbPath := historyDBPath
	if dbPath == "" {
		dbPath = os.Getenv("INFRANOW_HISTORY_DB")
	}
	if dbPath == "" {
		var err error
		dbPath, err = history.DefaultDBPath()
		if err != nil {
			return nil, fmt.Errorf("cannot determine history DB path: %w", err)
		}
	}
	return history.NewSQLiteStore(dbPath)
}

func runHistoryList(cmd *cobra.Command, args []string) error {
	store, err := openHistoryStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		util.Exit(util.ExitRuntimeError)
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "[infranow] warning: failed to close history store: %v\n", closeErr)
		}
	}()

	sinceDuration, err := parseDuration(historyListSince)
	if err != nil {
		return fmt.Errorf("invalid --since: %w", err)
	}

	opts := history.ListOpts{
		Since:       time.Now().Add(-sinceDuration),
		MinSeverity: historyListSeverity,
		Limit:       historyListLimit,
	}

	records, err := store.List(context.Background(), opts)
	if err != nil {
		return fmt.Errorf("list history: %w", err)
	}

	if historyListOutput == "json" {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(records)
	}

	// Text output
	if len(records) == 0 {
		fmt.Println("No history records found.")
		return nil
	}

	fmt.Printf("%-10s %-20s %-30s %-10s %6s  %-20s\n", "SEVERITY", "TYPE", "ENTITY", "COUNT", "FP", "LAST SEEN")
	fmt.Printf("%-10s %-20s %-30s %-10s %6s  %-20s\n", "--------", "----", "------", "-----", "--", "---------")
	for i := range records {
		r := &records[i]
		fp := r.Fingerprint
		if len(fp) > 6 {
			fp = fp[:6]
		}
		fmt.Printf("%-10s %-20s %-30s %-10d %6s  %s\n",
			r.Severity,
			truncateStr(r.Type, 20),
			truncateStr(r.Entity, 30),
			r.OccurrenceCount,
			fp,
			r.LastSeen.Format("2006-01-02 15:04"),
		)
	}
	fmt.Printf("\n%d records\n", len(records))

	return nil
}

func runHistoryPrune(cmd *cobra.Command, args []string) error {
	store, err := openHistoryStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		util.Exit(util.ExitRuntimeError)
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "[infranow] warning: failed to close history store: %v\n", closeErr)
		}
	}()

	age, err := parseDuration(historyPruneAge)
	if err != nil {
		return fmt.Errorf("invalid --older-than: %w", err)
	}

	if historyPruneDryRun {
		records, listErr := store.List(context.Background(), history.ListOpts{Limit: 0})
		if listErr != nil {
			return fmt.Errorf("list history: %w", listErr)
		}
		cutoff := time.Now().Add(-age)
		count := 0
		for i := range records {
			if records[i].LastSeen.Before(cutoff) {
				count++
			}
		}
		fmt.Printf("Would prune %d records older than %s\n", count, historyPruneAge)
		return nil
	}

	deleted, err := store.Prune(context.Background(), age)
	if err != nil {
		return fmt.Errorf("prune history: %w", err)
	}

	fmt.Printf("Pruned %d records older than %s\n", deleted, historyPruneAge)
	return nil
}

// parseDuration parses a duration string with day support (e.g. "7d", "24h", "90d")
func parseDuration(s string) (time.Duration, error) {
	if len(s) > 1 && s[len(s)-1] == 'd' {
		var days int
		if _, err := fmt.Sscanf(s, "%dd", &days); err == nil {
			return time.Duration(days) * 24 * time.Hour, nil
		}
	}
	return time.ParseDuration(s)
}

// truncateStr truncates a string to maxLen, appending ".." if truncated
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 2 {
		return s[:maxLen]
	}
	return s[:maxLen-2] + ".."
}

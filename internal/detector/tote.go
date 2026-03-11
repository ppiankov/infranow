package detector

import (
	"context"
	"fmt"
	"time"

	"github.com/ppiankov/infranow/internal/metrics"
	"github.com/ppiankov/infranow/internal/models"
)

const (
	toteCheckInterval    = 30 * time.Second
	blastRadiusSalvage   = 5
	blastRadiusPush      = 3
	blastRadiusDetection = 3
)

// ToteSalvageFailureDetector detects failing tote image salvage operations
type ToteSalvageFailureDetector struct {
	interval time.Duration
}

func NewToteSalvageFailureDetector() *ToteSalvageFailureDetector {
	return &ToteSalvageFailureDetector{interval: toteCheckInterval}
}

func (d *ToteSalvageFailureDetector) Name() string            { return "tote_salvage_failure" }
func (d *ToteSalvageFailureDetector) EntityTypes() []string   { return []string{"tote_salvage"} }
func (d *ToteSalvageFailureDetector) Interval() time.Duration { return d.interval }

func (d *ToteSalvageFailureDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := `increase(tote_salvage_failures_total[5m]) > 0`
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("tote salvage failure query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		failures := float64(sample.Value)
		problems = append(problems, &models.Problem{
			ID:          "tote/salvage_failure",
			Entity:      "tote/salvage",
			EntityType:  "tote_salvage",
			Type:        "tote_salvage_failure",
			Severity:    models.SeverityCritical,
			Title:       fmt.Sprintf("Image salvage failing (%.0f failures in 5m)", failures),
			Message:     fmt.Sprintf("tote: %.0f image salvage operations failed in the last 5 minutes", failures),
			Labels:      map[string]string{},
			Metrics:     map[string]float64{"failures_5m": failures},
			Hint:        "Check tote controller logs and agent connectivity",
			RunbookURL:  models.RunbookBaseURL + "tote_salvage_failure.md",
			BlastRadius: blastRadiusSalvage,
		})
	}
	return problems, nil
}

// TotePushFailureDetector detects failing backup registry push operations
type TotePushFailureDetector struct {
	interval time.Duration
}

func NewTotePushFailureDetector() *TotePushFailureDetector {
	return &TotePushFailureDetector{interval: toteCheckInterval}
}

func (d *TotePushFailureDetector) Name() string            { return "tote_push_failure" }
func (d *TotePushFailureDetector) EntityTypes() []string   { return []string{"tote_push"} }
func (d *TotePushFailureDetector) Interval() time.Duration { return d.interval }

func (d *TotePushFailureDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	query := `increase(tote_push_failures_total[10m]) > 0`
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("tote push failure query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		failures := float64(sample.Value)
		problems = append(problems, &models.Problem{
			ID:          "tote/push_failure",
			Entity:      "tote/push",
			EntityType:  "tote_push",
			Type:        "tote_push_failure",
			Severity:    models.SeverityWarning,
			Title:       fmt.Sprintf("Backup registry push failing (%.0f failures in 10m)", failures),
			Message:     fmt.Sprintf("tote: %.0f backup registry push operations failed in the last 10 minutes", failures),
			Labels:      map[string]string{},
			Metrics:     map[string]float64{"failures_10m": failures},
			Hint:        "Check backup registry connectivity and credentials",
			RunbookURL:  models.RunbookBaseURL + "tote_push_failure.md",
			BlastRadius: blastRadiusPush,
		})
	}
	return problems, nil
}

// ToteHighFailureRateDetector detects when most image pull failures cannot be salvaged
type ToteHighFailureRateDetector struct {
	interval time.Duration
}

func NewToteHighFailureRateDetector() *ToteHighFailureRateDetector {
	return &ToteHighFailureRateDetector{interval: toteCheckInterval}
}

func (d *ToteHighFailureRateDetector) Name() string            { return "tote_high_failure_rate" }
func (d *ToteHighFailureRateDetector) EntityTypes() []string   { return []string{"tote_detection"} }
func (d *ToteHighFailureRateDetector) Interval() time.Duration { return d.interval }

func (d *ToteHighFailureRateDetector) Detect(ctx context.Context, provider metrics.MetricsProvider, _ time.Duration) ([]*models.Problem, error) {
	// Only fire when there are detected failures AND most are not actionable (tag-based, not digest)
	query := `increase(tote_not_actionable_total[10m]) > increase(tote_salvageable_images_total[10m]) and increase(tote_detected_failures_total[10m]) > 0`
	result, err := provider.QueryInstant(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("tote high failure rate query failed: %w", err)
	}

	problems := make([]*models.Problem, 0, len(result))
	for _, sample := range result {
		notActionable := float64(sample.Value)
		problems = append(problems, &models.Problem{
			ID:          "tote/high_failure_rate",
			Entity:      "tote/detection",
			EntityType:  "tote_detection",
			Type:        "tote_high_failure_rate",
			Severity:    models.SeverityWarning,
			Title:       fmt.Sprintf("Most image failures not salvageable (%.0f tag-based in 10m)", notActionable),
			Message:     "tote: more image pull failures use tags than digests — tote cannot salvage tag-based references",
			Labels:      map[string]string{},
			Metrics:     map[string]float64{"not_actionable_10m": notActionable},
			Hint:        "Switch container images from tags to digests for salvage eligibility",
			RunbookURL:  models.RunbookBaseURL + "tote_high_failure_rate.md",
			BlastRadius: blastRadiusDetection,
		})
	}
	return problems, nil
}

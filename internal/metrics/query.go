package metrics

import (
	"fmt"
	"strings"
)

// QueryBuilder helps construct PromQL queries
type QueryBuilder struct {
	metric  string
	filters []string
	funcs   []string
}

// NewQuery creates a new query builder for the given metric
func NewQuery(metric string) *QueryBuilder {
	return &QueryBuilder{
		metric:  metric,
		filters: make([]string, 0),
		funcs:   make([]string, 0),
	}
}

// WithLabel adds a label filter
func (q *QueryBuilder) WithLabel(key, value string) *QueryBuilder {
	q.filters = append(q.filters, fmt.Sprintf(`%s="%s"`, key, value))
	return q
}

// WithLabelRegex adds a regex label filter
func (q *QueryBuilder) WithLabelRegex(key, pattern string) *QueryBuilder {
	q.filters = append(q.filters, fmt.Sprintf(`%s=~"%s"`, key, pattern))
	return q
}

// WithLabelNotEqual adds a not-equal label filter
func (q *QueryBuilder) WithLabelNotEqual(key, value string) *QueryBuilder {
	q.filters = append(q.filters, fmt.Sprintf(`%s!="%s"`, key, value))
	return q
}

// Rate adds rate function with window
func (q *QueryBuilder) Rate(window string) *QueryBuilder {
	q.funcs = append(q.funcs, fmt.Sprintf("rate(%s[%s])", "%s", window))
	return q
}

// Increase adds increase function with window
func (q *QueryBuilder) Increase(window string) *QueryBuilder {
	q.funcs = append(q.funcs, fmt.Sprintf("increase(%s[%s])", "%s", window))
	return q
}

// Build constructs the final PromQL query
func (q *QueryBuilder) Build() string {
	var query string
	if len(q.filters) > 0 {
		query = fmt.Sprintf("%s{%s}", q.metric, strings.Join(q.filters, ","))
	} else {
		query = q.metric
	}

	// Apply functions in order
	for _, fn := range q.funcs {
		query = fmt.Sprintf(fn, query)
	}

	return query
}

// Common query patterns

// GreaterThan adds a comparison operator
func GreaterThan(query string, threshold float64) string {
	return fmt.Sprintf("%s > %f", query, threshold)
}

// LessThan adds a comparison operator
func LessThan(query string, threshold float64) string {
	return fmt.Sprintf("%s < %f", query, threshold)
}

// Equals adds an equality operator
func Equals(query string, value float64) string {
	return fmt.Sprintf("%s == %f", query, value)
}

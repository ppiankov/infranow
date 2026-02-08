package monitor

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ppiankov/infranow/internal/models"
	"github.com/ppiankov/infranow/internal/util"
)

// SortMode determines how problems are sorted
type SortMode int

const (
	SortBySeverity SortMode = iota
	SortByRecency
	SortByCount
)

func (s SortMode) String() string {
	switch s {
	case SortBySeverity:
		return "severity"
	case SortByRecency:
		return "recency"
	case SortByCount:
		return "count"
	default:
		return "unknown"
	}
}

// Model is the Bubbletea model for the TUI
type Model struct {
	watcher         *Watcher
	prometheusURL   string
	refreshInterval time.Duration
	portForward     *util.PortForward

	problems       []*models.Problem
	sortMode       SortMode
	paused         bool
	viewport       viewport.Model
	searchMode     bool
	searchQuery    string
	filteredCount  int

	width  int
	height int
	ready  bool
}

type tickMsg time.Time

type updateMsg struct {
	problems []*models.Problem
}

// NewModel creates a new TUI model
func NewModel(watcher *Watcher, prometheusURL string, refreshInterval time.Duration, portForward *util.PortForward) Model {
	return Model{
		watcher:         watcher,
		prometheusURL:   prometheusURL,
		refreshInterval: refreshInterval,
		portForward:     portForward,
		problems:        []*models.Problem{},
		sortMode:        SortBySeverity,
		paused:          false,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(m.refreshInterval),
		waitForUpdate(m.watcher),
	)
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle search mode input
		if m.searchMode {
			switch msg.String() {
			case "esc", "ctrl+c":
				m.searchMode = false
				m.searchQuery = ""
				m.updateProblems()
			case "enter":
				m.searchMode = false
				m.updateProblems()
			case "backspace":
				if len(m.searchQuery) > 0 {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
					m.updateProblems()
				}
			default:
				// Add character to search
				m.searchQuery += msg.String()
				m.updateProblems()
			}
			return m, nil
		}

		// Normal mode key handling
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "p", " ":
			m.paused = !m.paused

		case "s":
			m.sortMode = (m.sortMode + 1) % 3
			m.updateProblems()

		case "/":
			m.searchMode = true
			m.searchQuery = ""

		case "esc":
			// Clear search
			m.searchQuery = ""
			m.updateProblems()

		case "r":
			// Restart port-forward if active
			if m.portForward != nil {
				go func() {
					_ = m.portForward.Restart() // Best-effort restart, status shown in UI
				}()
			}

		case "up", "k":
			m.viewport.ScrollUp(1)

		case "down", "j":
			m.viewport.ScrollDown(1)

		case "g", "home":
			m.viewport.GotoTop()

		case "G", "end":
			m.viewport.GotoBottom()

		case "pgup":
			m.viewport.PageUp()

		case "pgdown":
			m.viewport.PageDown()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-6) // Reserve space for header/footer
			m.viewport.YPosition = 3
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 6
		}

		m.updateViewport()

	case tickMsg:
		if !m.paused {
			m.updateProblems()
		}
		return m, tickCmd(m.refreshInterval)

	case updateMsg:
		m.problems = msg.problems
		m.updateViewport()
		return m, waitForUpdate(m.watcher)
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the TUI
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	var b strings.Builder

	// Header
	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	// Content
	if len(m.problems) == 0 {
		b.WriteString(m.renderEmptyState())
	} else {
		b.WriteString(m.viewport.View())
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(m.renderFooter())

	return b.String()
}

func (m *Model) updateProblems() {
	// Get problems based on sort mode
	var allProblems []*models.Problem
	switch m.sortMode {
	case SortBySeverity:
		allProblems = m.watcher.GetProblems()
	case SortByRecency:
		allProblems = m.watcher.GetProblemsByRecency()
	case SortByCount:
		allProblems = m.watcher.GetProblemsByCount()
	}

	// Apply search filter if active
	if m.searchQuery != "" {
		filtered := make([]*models.Problem, 0)
		query := strings.ToLower(m.searchQuery)
		for _, p := range allProblems {
			// Search in entity, title, message, type
			if strings.Contains(strings.ToLower(p.Entity), query) ||
				strings.Contains(strings.ToLower(p.Title), query) ||
				strings.Contains(strings.ToLower(p.Message), query) ||
				strings.Contains(strings.ToLower(p.Type), query) ||
				strings.Contains(strings.ToLower(string(p.Severity)), query) {
				filtered = append(filtered, p)
			}
		}
		m.problems = filtered
		m.filteredCount = len(allProblems) - len(filtered)
	} else {
		m.problems = allProblems
		m.filteredCount = 0
	}

	m.updateViewport()
}

func (m *Model) updateViewport() {
	m.viewport.SetContent(m.renderProblems())
}

func (m Model) renderHeader() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12"))

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("10"))

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("9")).
		Bold(true)

	warningStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("11")).
		Bold(true)

	// Check Prometheus health with watchdog stats
	stats := m.watcher.GetPrometheusStats()
	var status string

	if !stats.Healthy {
		timeSince := time.Since(stats.LastCheck)
		status = errorStyle.Render(fmt.Sprintf("⚠  Prometheus DOWN (%s ago)", formatDuration(timeSince)))
	} else if stats.ErrorRate > 0.5 && stats.QueryCount > 10 {
		// High error rate - Prometheus might be struggling
		status = warningStyle.Render(fmt.Sprintf("⚠  Prometheus UNSTABLE (%.0f%% errors)", stats.ErrorRate*100))
	} else if !stats.LastSuccessfulQuery.IsZero() && time.Since(stats.LastSuccessfulQuery) > 2*time.Minute {
		// Haven't had successful query in a while
		status = warningStyle.Render(fmt.Sprintf("⚠  No data (%s ago)", formatDuration(time.Since(stats.LastSuccessfulQuery))))
	} else if m.paused {
		status = statusStyle.Render("⏸  Paused")
	} else {
		// Show healthy with query stats
		status = statusStyle.Render(fmt.Sprintf("●  Running (Q:%d E:%d)", stats.QueryCount, stats.ErrorCount))
	}

	title := titleStyle.Render("infranow - Infrastructure Monitor")
	sortInfo := fmt.Sprintf("Sort: %s", m.sortMode)

	line1 := lipgloss.JoinHorizontal(lipgloss.Left,
		title,
		strings.Repeat(" ", m.width-lipgloss.Width(title)-lipgloss.Width(sortInfo)),
		sortInfo,
	)

	// Line 2: Prometheus URL + Port-forward status
	promInfo := fmt.Sprintf("Prometheus: %s", m.prometheusURL)

	var pfStatus string
	if m.portForward != nil {
		pfStatusStr := m.portForward.GetStatusString()
		pfStyle := statusStyle

		switch m.portForward.GetStatus() {
		case util.StatusRunning:
			pfStyle = statusStyle.Foreground(lipgloss.Color("10")) // Green
			pfStatus = pfStyle.Render(fmt.Sprintf(" [PF: %s]", pfStatusStr))
		case util.StatusStarting:
			pfStyle = statusStyle.Foreground(lipgloss.Color("11")) // Yellow
			pfStatus = pfStyle.Render(" [PF: starting...]")
		case util.StatusFailed:
			pfStyle = errorStyle
			pfStatus = pfStyle.Render(fmt.Sprintf(" [PF: %s]", pfStatusStr))
		default:
			pfStatus = statusStyle.Render(" [PF: stopped]")
		}
	}

	line2 := lipgloss.JoinHorizontal(lipgloss.Left,
		promInfo,
		pfStatus,
		strings.Repeat(" ", 5),
		fmt.Sprintf("Refresh: %s", m.refreshInterval),
	)

	summary := m.watcher.GetSummary()
	problemCount := fmt.Sprintf("Problems: %d", len(m.problems))
	if m.filteredCount > 0 {
		problemCount = fmt.Sprintf("Problems: %d (%d filtered)", len(m.problems), m.filteredCount)
	}

	line3 := lipgloss.JoinHorizontal(lipgloss.Left,
		status,
		strings.Repeat(" ", 20),
		problemCount,
		strings.Repeat(" ", 5),
		fmt.Sprintf("Fatal: %d", summary[models.SeverityFatal]),
		strings.Repeat(" ", 3),
		fmt.Sprintf("Critical: %d", summary[models.SeverityCritical]),
		strings.Repeat(" ", 3),
		fmt.Sprintf("Warning: %d", summary[models.SeverityWarning]),
	)

	border := strings.Repeat("─", m.width)

	return strings.Join([]string{line1, line2, line3, border}, "\n")
}

func (m Model) renderEmptyState() string {
	emptyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("10")).
		Bold(true)

	padding := (m.height - 8) / 2
	var b strings.Builder

	for i := 0; i < padding; i++ {
		b.WriteString("\n")
	}

	centerText := "✓ No problems detected"
	leftPadding := (m.width - len(centerText)) / 2

	b.WriteString(strings.Repeat(" ", leftPadding))
	b.WriteString(emptyStyle.Render(centerText))

	return b.String()
}

func (m Model) renderProblems() string {
	var b strings.Builder

	for i, p := range m.problems {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(m.renderProblem(i+1, p))
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderProblem(index int, p *models.Problem) string {
	var icon, iconColor string
	switch p.Severity {
	case models.SeverityFatal:
		icon = "🔴"
		iconColor = "9"
	case models.SeverityCritical:
		icon = "🟠"
		iconColor = "214"
	case models.SeverityWarning:
		icon = "🟡"
		iconColor = "11"
	}

	indexStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	severityStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(iconColor)).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Italic(true)

	timeSince := time.Since(p.FirstSeen).Round(time.Second)
	timeStr := formatDuration(timeSince)

	var b strings.Builder

	// Line 1: [index] severity: title
	b.WriteString(indexStyle.Render(fmt.Sprintf("[%d/%d]", index, len(m.problems))))
	b.WriteString("\n")
	b.WriteString(severityStyle.Render(fmt.Sprintf("%s %s: %s", icon, p.Severity, p.Title)))
	b.WriteString("\n")

	// Line 2: Entity
	b.WriteString(labelStyle.Render("Entity: "))
	b.WriteString(p.Entity)
	b.WriteString("\n")

	// Line 3: Metadata
	b.WriteString(labelStyle.Render(fmt.Sprintf("First seen: %s | Count: %d", timeStr, p.Count)))
	b.WriteString("\n")

	// Line 4: Hint
	b.WriteString(labelStyle.Render("Hint: "))
	b.WriteString(hintStyle.Render(p.Hint))

	return b.String()
}

func (m Model) renderFooter() string {
	border := strings.Repeat("─", m.width)
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	searchStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("11")).
		Bold(true)

	var help string
	if m.searchMode {
		help = searchStyle.Render(fmt.Sprintf("Search: %s_", m.searchQuery)) + helpStyle.Render("  (enter: apply  esc: cancel)")
	} else if m.searchQuery != "" {
		help = helpStyle.Render(fmt.Sprintf("Filter: %s  ", m.searchQuery)) + searchStyle.Render("(esc: clear)") + helpStyle.Render("  s: sort  p: pause  /: search  q: quit")
	} else {
		baseHelp := "s: sort  p: pause  /: search  ↑↓/jk: scroll  g/G: top/bottom"
		if m.portForward != nil {
			baseHelp += "  r: restart-pf"
		}
		baseHelp += "  q: quit"
		help = helpStyle.Render(baseHelp)
	}

	return border + "\n" + help
}

func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func waitForUpdate(watcher *Watcher) tea.Cmd {
	return func() tea.Msg {
		<-watcher.UpdateChan()
		return updateMsg{
			problems: watcher.GetProblems(),
		}
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}

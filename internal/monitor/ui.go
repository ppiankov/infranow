package monitor

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
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

// Layout constants for terminal space allocation
const (
	headerLines    = 4 // title, prom info, status, separator
	footerLines    = 2 // separator + help text
	detailLines    = 7 // detail panel height
	detailMinLines = 3 // compact detail for small terminals
	separatorLines = 1 // between table and detail
	minTableHeight = 3
	smallTerminal  = 20 // below this, use compact detail

	// Column widths
	numColWidth      = 3
	sevColWidth      = 7
	ageColWidth      = 8
	entityColMin     = 15
	titleColMin      = 10
	entityColDefault = 30
	colPadding       = 10 // total padding between columns

	// promStaleThreshold triggers a warning if no successful query in this duration
	promStaleThreshold = 2 * time.Minute
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

	problems      []*models.Problem
	sortMode      SortMode
	paused        bool
	tbl           table.Model
	searchMode    bool
	searchQuery   string
	filteredCount int
	statusMsg     string

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
	cols := computeColumns(80)
	t := table.New(
		table.WithColumns(cols),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(minTableHeight),
		table.WithKeyMap(infranowTableKeyMap()),
		table.WithStyles(infranowTableStyles()),
	)

	return Model{
		watcher:         watcher,
		prometheusURL:   prometheusURL,
		refreshInterval: refreshInterval,
		portForward:     portForward,
		problems:        []*models.Problem{},
		sortMode:        SortBySeverity,
		tbl:             t,
	}
}

func infranowTableKeyMap() table.KeyMap {
	return table.KeyMap{
		LineUp:       key.NewBinding(key.WithKeys("up", "k")),
		LineDown:     key.NewBinding(key.WithKeys("down", "j")),
		PageUp:       key.NewBinding(key.WithKeys("pgup")),
		PageDown:     key.NewBinding(key.WithKeys("pgdown")),
		GotoTop:      key.NewBinding(key.WithKeys("home", "g")),
		GotoBottom:   key.NewBinding(key.WithKeys("end", "G")),
		HalfPageUp:   key.NewBinding(key.WithKeys("ctrl+u")),
		HalfPageDown: key.NewBinding(key.WithKeys("ctrl+d")),
	}
}

func infranowTableStyles() table.Styles {
	return table.Styles{
		Header:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).Padding(0, 1),
		Cell:     lipgloss.NewStyle().Padding(0, 1),
		Selected: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("57")),
	}
}

func computeColumns(width int) []table.Column {
	entityWidth := entityColDefault
	titleWidth := width - numColWidth - sevColWidth - entityWidth - ageColWidth - colPadding
	if titleWidth < titleColMin {
		entityWidth = entityColMin
		titleWidth = width - numColWidth - sevColWidth - entityWidth - ageColWidth - colPadding
	}
	if titleWidth < titleColMin {
		titleWidth = titleColMin
	}
	return []table.Column{
		{Title: "#", Width: numColWidth},
		{Title: "SEV", Width: sevColWidth},
		{Title: "ENTITY", Width: entityWidth},
		{Title: "TITLE", Width: titleWidth},
		{Title: "AGE", Width: ageColWidth},
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
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.searchMode {
			return m.handleSearchKey(msg)
		}
		return m.handleNormalKey(msg)

	case tea.WindowSizeMsg:
		return m.handleResize(msg)

	case tickMsg:
		m.statusMsg = ""
		if !m.paused {
			m.updateProblems()
		}
		return m, tickCmd(m.refreshInterval)

	case updateMsg:
		m.problems = msg.problems
		m.rebuildTableRows()
		return m, waitForUpdate(m.watcher)
	}

	var cmd tea.Cmd
	m.tbl, cmd = m.tbl.Update(msg)
	return m, cmd
}

func (m Model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		m.searchQuery = ""
		m.updateProblems()
	case "r":
		if m.portForward != nil {
			go func() {
				_ = m.portForward.Restart() // Best-effort restart, status shown in UI
			}()
		}
	case "?":
		m.statusMsg = m.openSelectedRunbook()
	case "c":
		m.statusMsg = m.copySelectedProblem()
	case "y":
		m.statusMsg = m.yankSelectedEntity()
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		m.jumpToRow(int(msg.String()[0] - '0'))
	default:
		// Delegate navigation keys to table
		var cmd tea.Cmd
		m.tbl, cmd = m.tbl.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.searchMode = false
		m.searchQuery = ""
		m.updateProblems()
	case "enter":
		m.searchMode = false
		m.updateProblems()
	case "backspace":
		if m.searchQuery != "" {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			m.updateProblems()
		}
	default:
		m.searchQuery += msg.String()
		m.updateProblems()
	}
	return m, nil
}

func (m Model) handleResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	cols := computeColumns(msg.Width)
	m.tbl.SetColumns(cols)
	m.tbl.SetWidth(msg.Width)

	detailHeight := detailLines
	if msg.Height < smallTerminal {
		detailHeight = detailMinLines
	}
	tableHeight := msg.Height - headerLines - footerLines - separatorLines - detailHeight
	if tableHeight < minTableHeight {
		tableHeight = minTableHeight
	}
	m.tbl.SetHeight(tableHeight)

	m.ready = true
	m.rebuildTableRows()
	return m, nil
}

// View renders the TUI
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	var b strings.Builder

	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	if len(m.problems) == 0 {
		b.WriteString(m.renderEmptyState())
	} else {
		b.WriteString(m.tbl.View())
		b.WriteString("\n")
		b.WriteString(strings.Repeat("─", m.width))
		b.WriteString("\n")
		b.WriteString(m.renderDetailPanel())
	}

	b.WriteString("\n")
	b.WriteString(m.renderFooter())

	return b.String()
}

// selectedProblem returns the problem at the table cursor, or nil.
func (m Model) selectedProblem() *models.Problem {
	idx := m.tbl.Cursor()
	if idx < 0 || idx >= len(m.problems) {
		return nil
	}
	return m.problems[idx]
}

func (m *Model) updateProblems() {
	var allProblems []*models.Problem
	switch m.sortMode {
	case SortBySeverity:
		allProblems = m.watcher.GetProblems()
	case SortByRecency:
		allProblems = m.watcher.GetProblemsByRecency()
	case SortByCount:
		allProblems = m.watcher.GetProblemsByCount()
	}

	m.watcher.AnnotateHistory(allProblems)

	if m.searchQuery != "" {
		filtered := make([]*models.Problem, 0)
		query := strings.ToLower(m.searchQuery)
		for _, p := range allProblems {
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

	m.rebuildTableRows()
}

func (m *Model) rebuildTableRows() {
	rows := make([]table.Row, len(m.problems))
	now := time.Now()
	cols := m.tbl.Columns()

	// Determine entity and title widths from current columns
	entityWidth := entityColDefault
	titleWidth := titleColMin
	if len(cols) >= 5 {
		entityWidth = cols[2].Width
		titleWidth = cols[3].Width
	}

	for i, p := range m.problems {
		rows[i] = table.Row{
			fmt.Sprintf("%d", i+1),
			shortSeverity(p.Severity),
			truncate(p.Entity, entityWidth),
			truncate(p.Title, titleWidth),
			humanAge(now.Sub(p.FirstSeen)),
		}
	}
	m.tbl.SetRows(rows)
}

func (m *Model) jumpToRow(n int) {
	if n < 1 || n > len(m.problems) {
		return
	}
	m.tbl.SetCursor(n - 1)
}

func (m *Model) copySelectedProblem() string {
	p := m.selectedProblem()
	if p == nil {
		return "No problem selected"
	}
	return copyToClipboard(formatProblemDetail(p))
}

func (m *Model) yankSelectedEntity() string {
	p := m.selectedProblem()
	if p == nil {
		return "No problem selected"
	}
	return copyToClipboard(p.Entity)
}

func (m *Model) openSelectedRunbook() string {
	p := m.selectedProblem()
	if p == nil {
		return "No problems to show runbook for"
	}
	if p.RunbookURL == "" {
		return "No runbook available"
	}

	var cmd string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	default:
		cmd = "xdg-open"
	}
	if err := exec.Command(cmd, p.RunbookURL).Start(); err != nil { //nolint:gosec // URL is from internal RunbookBaseURL constant
		return "Failed to open runbook"
	}
	return "Opening runbook..."
}

func formatProblemDetail(p *models.Problem) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %s\n", p.Severity, p.Title)
	fmt.Fprintf(&b, "Entity: %s\n", p.Entity)
	if p.Message != "" {
		fmt.Fprintf(&b, "Message: %s\n", p.Message)
	}
	fmt.Fprintf(&b, "First seen: %s | Count: %d\n", humanAge(time.Since(p.FirstSeen)), p.Count)
	if p.Hint != "" {
		fmt.Fprintf(&b, "Hint: %s\n", p.Hint)
	}
	if p.RunbookURL != "" {
		fmt.Fprintf(&b, "Runbook: %s\n", p.RunbookURL)
	}
	return b.String()
}

func (m Model) renderDetailPanel() string {
	p := m.selectedProblem()
	if p == nil {
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
		return dimStyle.Render("  No problem selected")
	}

	var iconColor string
	switch p.Severity {
	case models.SeverityFatal:
		iconColor = "9"
	case models.SeverityCritical:
		iconColor = "214"
	case models.SeverityWarning:
		iconColor = "11"
	}

	sevStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(iconColor)).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Italic(true)

	var b strings.Builder

	b.WriteString(sevStyle.Render(fmt.Sprintf("  %s: %s", p.Severity, p.Title)))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("  Entity: "))
	b.WriteString(p.Entity)
	b.WriteString("\n")
	b.WriteString(labelStyle.Render(fmt.Sprintf("  Type: %s | Count: %d | Blast: %d", p.Type, p.Count, p.BlastRadius)))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render(fmt.Sprintf("  First: %s | Last: %s",
		humanAge(time.Since(p.FirstSeen)), humanAge(time.Since(p.LastSeen)))))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("  Hint: "))
	b.WriteString(hintStyle.Render(p.Hint))

	if p.History != nil {
		b.WriteString("\n")
		if p.History.TotalOccurrences > 1 {
			b.WriteString(labelStyle.Render(fmt.Sprintf("  History: recurring (%s) | %d occurrences",
				p.History.RecurringSince, p.History.TotalOccurrences)))
		} else {
			b.WriteString(labelStyle.Render("  History: new (first seen)"))
		}
	}

	if m.height >= smallTerminal && p.RunbookURL != "" {
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("  Runbook: "))
		b.WriteString(p.RunbookURL)
	}

	return b.String()
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

	stats := m.watcher.GetPrometheusStats()
	var status string

	if !stats.Healthy {
		timeSince := time.Since(stats.LastCheck)
		status = errorStyle.Render(fmt.Sprintf("⚠  Prometheus DOWN (%s ago)", formatDuration(timeSince)))
	} else if stats.ErrorRate > 0.5 && stats.QueryCount > 10 {
		status = warningStyle.Render(fmt.Sprintf("⚠  Prometheus UNSTABLE (%.0f%% errors)", stats.ErrorRate*100))
	} else if !stats.LastSuccessfulQuery.IsZero() && time.Since(stats.LastSuccessfulQuery) > promStaleThreshold {
		status = warningStyle.Render(fmt.Sprintf("⚠  No data (%s ago)", formatDuration(time.Since(stats.LastSuccessfulQuery))))
	} else if m.paused {
		status = statusStyle.Render("⏸  Paused")
	} else {
		status = statusStyle.Render(fmt.Sprintf("●  Running (Q:%d E:%d)", stats.QueryCount, stats.ErrorCount))
	}

	title := titleStyle.Render("infranow - Infrastructure Monitor")
	sortInfo := fmt.Sprintf("Sort: %s", m.sortMode)

	line1 := lipgloss.JoinHorizontal(lipgloss.Left,
		title,
		strings.Repeat(" ", m.width-lipgloss.Width(title)-lipgloss.Width(sortInfo)),
		sortInfo,
	)

	promInfo := fmt.Sprintf("Prometheus: %s", sanitizeURL(m.prometheusURL))

	var pfStatus string
	if m.portForward != nil {
		pfStatusStr := m.portForward.GetStatusString()
		pfStyle := statusStyle

		switch m.portForward.GetStatus() {
		case util.StatusRunning:
			pfStyle = statusStyle.Foreground(lipgloss.Color("10"))
			pfStatus = pfStyle.Render(fmt.Sprintf(" [PF: %s]", pfStatusStr))
		case util.StatusStarting:
			pfStyle = statusStyle.Foreground(lipgloss.Color("11"))
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
		baseHelp := "s: sort  p: pause  /: search  ?: runbook  c: copy  y: yank  1-9: jump  jk: nav"
		if m.portForward != nil {
			baseHelp += "  r: pf"
		}
		baseHelp += "  q: quit"
		help = helpStyle.Render(baseHelp)
	}

	footer := border + "\n" + help
	if m.statusMsg != "" {
		footer += "\n" + helpStyle.Render(m.statusMsg)
	}
	return footer
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

// sanitizeURL redacts userinfo (credentials) from a URL for safe display
func sanitizeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "[invalid URL]"
	}
	if u.User != nil {
		u.User = url.UserPassword("REDACTED", "REDACTED")
	}
	return u.String()
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

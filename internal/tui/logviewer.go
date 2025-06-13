package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/d-kuro/gwq/internal/claude"
)

// Simplified color palette - minimal and readable
var (
	primaryColor = lipgloss.Color("#0EA5E9") // Blue
	successColor = lipgloss.Color("#22C55E") // Green
	errorColor   = lipgloss.Color("#EF4444") // Red
	warningColor = lipgloss.Color("#F59E0B") // Orange
	mutedColor   = lipgloss.Color("#64748B") // Gray
)

// Simplified styles - focus on readability
var (
	// Header styles - clean and minimal
	headerStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Padding(1, 0).
			MarginBottom(1)

	infoStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Padding(0, 1).
			MarginBottom(1)

	// Status styles - only colors that convey meaning
	statusRunningStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true)

	statusCompletedStyle = lipgloss.NewStyle().
				Foreground(successColor).
				Bold(true)

	statusFailedStyle = lipgloss.NewStyle().
				Foreground(errorColor).
				Bold(true)

	statusAbortedStyle = lipgloss.NewStyle().
				Foreground(warningColor).
				Bold(true)

	// Content styles - minimal borders, focus on content
	sectionTitleStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true).
				Underline(true).
				MarginTop(1).
				MarginBottom(1)

	sectionContentStyle = lipgloss.NewStyle().
				Padding(0, 2).
				MarginBottom(1)

	// Footer styles - unobtrusive
	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

	scrollInfoStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Bold(true)

	footerStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(mutedColor).
			Padding(1, 0).
			MarginTop(1)
)

// LogSection represents a structured section of the log
type LogSection struct {
	Title   string
	Content string
}

// LogViewerModel represents the TUI model for log viewing
type LogViewerModel struct {
	metadata     *claude.ExecutionMetadata
	rawContent   string
	sections     []LogSection
	scrollY      int
	maxScrollY   int
	width        int
	height       int
	contentArea  int
	renderedView string
}

// NewLogViewerModel creates a new log viewer model
func NewLogViewerModel(metadata *claude.ExecutionMetadata, logContent string) LogViewerModel {
	model := LogViewerModel{
		metadata:   metadata,
		rawContent: logContent,
		scrollY:    0,
	}
	model.sections = parseLogContent(logContent)
	return model
}

// Init initializes the model
func (m LogViewerModel) Init() tea.Cmd {
	return nil
}

// Update handles input and updates the model
func (m LogViewerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.contentArea = m.height - 8 // Account for header and footer
		m.renderSections()
		m.updateMaxScroll()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit

		case "up", "k":
			if m.scrollY > 0 {
				m.scrollY--
			}

		case "down", "j":
			if m.scrollY < m.maxScrollY {
				m.scrollY++
			}

		case "pgup":
			m.scrollY -= m.contentArea
			if m.scrollY < 0 {
				m.scrollY = 0
			}

		case "pgdown":
			m.scrollY += m.contentArea
			if m.scrollY > m.maxScrollY {
				m.scrollY = m.maxScrollY
			}

		case "home":
			m.scrollY = 0

		case "end":
			m.scrollY = m.maxScrollY
		}
	}

	return m, nil
}

// View renders the TUI
func (m LogViewerModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Ensure we have rendered content
	if m.renderedView == "" {
		m.renderSections()
	}

	var sections []string

	// Header with execution info
	header := m.renderHeader()
	sections = append(sections, header)

	// Content area
	content := m.renderContent()
	sections = append(sections, content)

	// Footer with help
	footer := m.renderFooter()
	sections = append(sections, footer)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m LogViewerModel) renderHeader() string {
	if m.metadata == nil {
		return headerStyle.Render("Claude Log Viewer")
	}

	// Clean title without excessive decoration
	title := fmt.Sprintf("Execution: %s", m.metadata.ExecutionID)
	header := headerStyle.Render(title)

	// Simple info display
	statusStyled := m.getStyledStatus()
	startTime := m.metadata.StartTime.Format("2006-01-02 15:04:05")

	var infoLines []string
	infoLines = append(infoLines, fmt.Sprintf("Status: %s", statusStyled))
	infoLines = append(infoLines, fmt.Sprintf("Started: %s", startTime))

	// Duration if available
	if duration := m.getDurationString(); duration != "" {
		infoLines = append(infoLines, fmt.Sprintf("Duration: %s", duration))
	}

	// Repository if available
	if m.metadata.Repository != "" {
		infoLines = append(infoLines, fmt.Sprintf("Repository: %s", m.metadata.Repository))
	}

	info := infoStyle.Render(strings.Join(infoLines, " • "))

	return lipgloss.JoinVertical(lipgloss.Left, header, info)
}

func (m LogViewerModel) renderContent() string {
	if m.renderedView == "" {
		return "No content to display"
	}

	lines := strings.Split(m.renderedView, "\n")

	// Calculate visible lines
	start := m.scrollY
	end := start + m.contentArea
	if end > len(lines) {
		end = len(lines)
	}

	var visibleLines []string
	if start < len(lines) {
		visibleLines = lines[start:end]
	}

	return strings.Join(visibleLines, "\n")
}

func (m LogViewerModel) renderFooter() string {
	totalLines := len(strings.Split(m.renderedView, "\n"))
	currentEnd := min(m.scrollY+m.contentArea, totalLines)

	scrollInfo := scrollInfoStyle.Render(fmt.Sprintf("Line %d-%d of %d",
		m.scrollY+1, currentEnd, totalLines))

	help := helpStyle.Render("↑/k: up • ↓/j: down • PgUp/PgDn: page • Home/End: start/end • q/Esc: quit")

	footerContent := lipgloss.JoinHorizontal(lipgloss.Left,
		scrollInfo,
		strings.Repeat(" ", max(0, m.width-lipgloss.Width(scrollInfo)-lipgloss.Width(help)-4)),
		help)

	return footerStyle.Width(m.width).Render(footerContent)
}

func (m LogViewerModel) getStyledStatus() string {
	if m.metadata == nil {
		return "Unknown"
	}

	status := string(m.metadata.Status)
	icon := m.getStatusIcon()

	switch m.metadata.Status {
	case claude.ExecutionStatusRunning:
		return statusRunningStyle.Render(fmt.Sprintf("%s %s", icon, status))
	case claude.ExecutionStatusCompleted:
		return statusCompletedStyle.Render(fmt.Sprintf("%s %s", icon, status))
	case claude.ExecutionStatusFailed:
		return statusFailedStyle.Render(fmt.Sprintf("%s %s", icon, status))
	case claude.ExecutionStatusAborted:
		return statusAbortedStyle.Render(fmt.Sprintf("%s %s", icon, status))
	default:
		return fmt.Sprintf("%s %s", icon, status)
	}
}

func (m LogViewerModel) getStatusIcon() string {
	if m.metadata == nil {
		return "⚪"
	}

	switch m.metadata.Status {
	case claude.ExecutionStatusRunning:
		return "🔄"
	case claude.ExecutionStatusCompleted:
		return "✅"
	case claude.ExecutionStatusFailed:
		return "❌"
	case claude.ExecutionStatusAborted:
		return "⚠️"
	default:
		return "⚪"
	}
}

func (m LogViewerModel) getDurationString() string {
	if m.metadata == nil || m.metadata.EndTime.IsZero() {
		return ""
	}
	return m.metadata.EndTime.Sub(m.metadata.StartTime).String()
}

func (m *LogViewerModel) updateMaxScroll() {
	totalLines := len(strings.Split(m.renderedView, "\n"))
	m.maxScrollY = max(0, totalLines-m.contentArea)
}

func (m *LogViewerModel) renderSections() {
	if m.width == 0 {
		return
	}

	var renderedSections []string

	for _, section := range m.sections {
		if section.Content == "" {
			continue
		}

		// Simple title with consistent styling
		title := sectionTitleStyle.Render(section.Title)

		// Clean content without heavy styling
		content := sectionContentStyle.Render(section.Content)

		// Combine with natural spacing
		sectionText := lipgloss.JoinVertical(lipgloss.Left, title, content)
		renderedSections = append(renderedSections, sectionText)
	}

	m.renderedView = strings.Join(renderedSections, "\n")
}

// parseLogContent parses the log content into structured sections
func parseLogContent(content string) []LogSection {
	var sections []LogSection

	// Split content by common section markers
	lines := strings.Split(content, "\n")
	var currentSection LogSection
	var currentContent []string

	for _, line := range lines {
		// Check for section headers
		if strings.Contains(line, "💬 Prompt:") {
			// Save previous section if exists
			if currentSection.Title != "" && len(currentContent) > 0 {
				currentSection.Content = strings.TrimSpace(strings.Join(currentContent, "\n"))
				sections = append(sections, currentSection)
				currentContent = nil
			}
			// Start new prompt section
			currentSection = LogSection{
				Title: "💬 Prompt",
			}
		} else if strings.Contains(line, "🤖 Claude's Response:") {
			// Save previous section if exists
			if currentSection.Title != "" && len(currentContent) > 0 {
				currentSection.Content = strings.TrimSpace(strings.Join(currentContent, "\n"))
				sections = append(sections, currentSection)
				currentContent = nil
			}
			// Start new response section
			currentSection = LogSection{
				Title: "🤖 Claude's Response",
			}
		} else if strings.Contains(line, "⚡ Operation Flow:") {
			// Save previous section if exists
			if currentSection.Title != "" && len(currentContent) > 0 {
				currentSection.Content = strings.TrimSpace(strings.Join(currentContent, "\n"))
				sections = append(sections, currentSection)
				currentContent = nil
			}
			// Start new operation flow section
			currentSection = LogSection{
				Title: "⚡ Operation Flow",
			}
		} else if strings.Contains(line, "💰 Total Cost:") {
			// Save previous section if exists
			if currentSection.Title != "" && len(currentContent) > 0 {
				currentSection.Content = strings.TrimSpace(strings.Join(currentContent, "\n"))
				sections = append(sections, currentSection)
				currentContent = nil
			}
			// Start new cost section
			currentSection = LogSection{
				Title: "💰 Total Cost",
			}
		} else if strings.Contains(line, "📊 Summary:") {
			// Save previous section if exists
			if currentSection.Title != "" && len(currentContent) > 0 {
				currentSection.Content = strings.TrimSpace(strings.Join(currentContent, "\n"))
				sections = append(sections, currentSection)
				currentContent = nil
			}
			// Start new summary section
			currentSection = LogSection{
				Title: "📊 Summary",
			}
		} else if currentSection.Title != "" {
			// Add content to current section (skip the section header line)
			if !strings.Contains(line, "💬 Prompt:") &&
				!strings.Contains(line, "⚡ Operation Flow:") &&
				!strings.Contains(line, "🤖 Claude's Response:") &&
				!strings.Contains(line, "💰 Total Cost:") &&
				!strings.Contains(line, "📊 Summary:") &&
				!strings.Contains(line, "💰 Cost:") {
				currentContent = append(currentContent, line)
			}
		} else if currentSection.Title == "" {
			// If no section started yet, add to generic content
			currentContent = append(currentContent, line)
		}
	}

	// Save the last section
	if currentSection.Title != "" && len(currentContent) > 0 {
		currentSection.Content = strings.TrimSpace(strings.Join(currentContent, "\n"))
		sections = append(sections, currentSection)
	} else if len(currentContent) > 0 {
		// If no structured sections found, treat entire content as one section
		sections = append(sections, LogSection{
			Title:   "📄 Log Content",
			Content: strings.TrimSpace(strings.Join(currentContent, "\n")),
		})
	}

	// If no sections found at all, use entire content
	if len(sections) == 0 {
		sections = append(sections, LogSection{
			Title:   "📄 Log Content",
			Content: content,
		})
	}

	return sections
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// RunLogViewer starts the TUI log viewer
func RunLogViewer(metadata *claude.ExecutionMetadata, logContent string) error {
	model := NewLogViewerModel(metadata, logContent)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

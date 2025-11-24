package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bf/tg/internal/config"
	"github.com/bf/tg/internal/llm"
	"github.com/bf/tg/internal/taskwarrior"
)

type state int

const (
	stateLoading state = iota
	statePreview
	stateEditing
	stateConfirm
	stateDone
	stateError
)

type AddModel struct {
	cfg         *config.Config
	provider    llm.Provider
	twClient    *taskwarrior.Client
	original    string
	enrichment  *llm.Enrichment
	state       state
	spinner     spinner.Model
	err         error
	editField   int
	textInputs  []textinput.Model
	fieldNames  []string
	result      string
	skipEnrich  bool
}

type enrichmentMsg struct {
	enrichment *llm.Enrichment
	err        error
}

type taskAddedMsg struct {
	uuid string
	err  error
}

func NewAddModel(cfg *config.Config, provider llm.Provider, description string) *AddModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	// Create text inputs for editing
	fields := []string{"Description", "Beacons", "Directions", "Project", "Priority", "Due", "Scheduled", "Effort", "Impact", "Estimate", "Fun", "Blocks"}
	inputs := make([]textinput.Model, len(fields))
	for i := range inputs {
		ti := textinput.New()
		ti.Prompt = ""
		ti.CharLimit = 256
		if i == 0 {
			ti.Focus()
		}
		inputs[i] = ti
	}

	return &AddModel{
		cfg:        cfg,
		provider:   provider,
		twClient:   taskwarrior.New(),
		original:   description,
		state:      stateLoading,
		spinner:    s,
		textInputs: inputs,
		fieldNames: fields,
	}
}

func (m *AddModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.fetchEnrichment(),
	)
}

func (m *AddModel) fetchEnrichment() tea.Cmd {
	return func() tea.Msg {
		enrichment, err := m.provider.Enrich(
			context.Background(),
			m.original,
			m.cfg.Beacons,
			m.cfg.Projects,
		)
		return enrichmentMsg{enrichment: enrichment, err: err}
	}
}

func (m *AddModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case spinner.TickMsg:
		if m.state == stateLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case enrichmentMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = stateError
			return m, nil
		}
		m.enrichment = msg.enrichment
		m.populateInputs()
		m.state = statePreview
		return m, nil

	case taskAddedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = stateError
			return m, nil
		}
		m.result = msg.uuid
		m.state = stateDone
		return m, tea.Quit
	}

	// Update text inputs if editing
	if m.state == stateEditing {
		var cmd tea.Cmd
		m.textInputs[m.editField], cmd = m.textInputs[m.editField].Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *AddModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateLoading:
		if msg.String() == "ctrl+c" || msg.String() == "esc" {
			return m, tea.Quit
		}

	case statePreview:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			return m, tea.Quit
		case "enter", "a":
			// Accept and add task
			return m, m.addTask()
		case "e":
			// Enter edit mode
			m.state = stateEditing
			m.editField = 0
			m.textInputs[0].Focus()
			return m, nil
		case "s":
			// Skip enrichment, add original
			m.skipEnrich = true
			return m, m.addTask()
		}

	case stateEditing:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			// Exit edit mode
			m.state = statePreview
			return m, nil
		case "enter":
			// Save and go to next field or exit
			m.updateEnrichmentFromInputs()
			if m.editField < len(m.textInputs)-1 {
				m.textInputs[m.editField].Blur()
				m.editField++
				m.textInputs[m.editField].Focus()
			} else {
				m.state = statePreview
			}
			return m, nil
		case "tab":
			m.textInputs[m.editField].Blur()
			m.editField = (m.editField + 1) % len(m.textInputs)
			m.textInputs[m.editField].Focus()
			return m, nil
		case "shift+tab":
			m.textInputs[m.editField].Blur()
			m.editField = (m.editField - 1 + len(m.textInputs)) % len(m.textInputs)
			m.textInputs[m.editField].Focus()
			return m, nil
		}

	case stateError:
		if msg.String() == "ctrl+c" || msg.String() == "esc" || msg.String() == "q" || msg.String() == "enter" {
			return m, tea.Quit
		}
	}

	// Update current text input
	if m.state == stateEditing {
		var cmd tea.Cmd
		m.textInputs[m.editField], cmd = m.textInputs[m.editField].Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *AddModel) populateInputs() {
	if m.enrichment == nil {
		return
	}
	m.textInputs[0].SetValue(m.enrichment.Description)
	m.textInputs[1].SetValue(strings.Join(m.enrichment.Beacons, " "))
	m.textInputs[2].SetValue(strings.Join(m.enrichment.Directions, " "))
	m.textInputs[3].SetValue(m.enrichment.Project)
	m.textInputs[4].SetValue(m.enrichment.Priority)
	m.textInputs[5].SetValue(m.enrichment.Due)
	m.textInputs[6].SetValue(m.enrichment.Scheduled)
	m.textInputs[7].SetValue(m.enrichment.Effort)
	m.textInputs[8].SetValue(m.enrichment.Impact)
	m.textInputs[9].SetValue(m.enrichment.Estimate)
	m.textInputs[10].SetValue(m.enrichment.Fun)
	m.textInputs[11].SetValue(fmt.Sprintf("%d", m.enrichment.Blocks))
}

func (m *AddModel) updateEnrichmentFromInputs() {
	if m.enrichment == nil {
		m.enrichment = &llm.Enrichment{}
	}
	m.enrichment.Description = m.textInputs[0].Value()
	m.enrichment.Beacons = splitTags(m.textInputs[1].Value())
	m.enrichment.Directions = splitTags(m.textInputs[2].Value())
	m.enrichment.Project = m.textInputs[3].Value()
	m.enrichment.Priority = m.textInputs[4].Value()
	m.enrichment.Due = m.textInputs[5].Value()
	m.enrichment.Scheduled = m.textInputs[6].Value()
	m.enrichment.Effort = m.textInputs[7].Value()
	m.enrichment.Impact = m.textInputs[8].Value()
	m.enrichment.Estimate = m.textInputs[9].Value()
	m.enrichment.Fun = m.textInputs[10].Value()
	m.enrichment.Blocks = parseBlocks(m.textInputs[11].Value())
}

func splitTags(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Fields(s)
	var tags []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			tags = append(tags, p)
		}
	}
	return tags
}

func parseBlocks(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

func (m *AddModel) addTask() tea.Cmd {
	return func() tea.Msg {
		var task taskwarrior.Task

		if m.skipEnrich {
			task.Description = m.original
		} else {
			task.Description = m.enrichment.Description
			task.Project = m.enrichment.Project
			task.Priority = m.enrichment.Priority
			task.Due = m.enrichment.Due
			task.Scheduled = m.enrichment.Scheduled
			task.Effort = m.enrichment.Effort
			task.Impact = m.enrichment.Impact
			task.Estimate = m.enrichment.Estimate
			task.Fun = m.enrichment.Fun
			task.Blocks = m.enrichment.Blocks

			// Combine beacons and directions as tags
			task.Tags = append(task.Tags, m.enrichment.Beacons...)
			task.Tags = append(task.Tags, m.enrichment.Directions...)

			if m.enrichment.IsWaste {
				task.Tags = append(task.Tags, "waste")
			}
		}

		uuid, err := m.twClient.Add(&task)
		return taskAddedMsg{uuid: uuid, err: err}
	}
}

func (m *AddModel) View() string {
	switch m.state {
	case stateLoading:
		return m.viewLoading()
	case statePreview:
		return m.viewPreview()
	case stateEditing:
		return m.viewEditing()
	case stateDone:
		return m.viewDone()
	case stateError:
		return m.viewError()
	default:
		return ""
	}
}

func (m *AddModel) viewLoading() string {
	return fmt.Sprintf("\n  %s Analyzing task with LLM...\n\n  %s\n",
		m.spinner.View(),
		subtitleStyle.Render(m.original),
	)
}

func (m *AddModel) viewPreview() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("tg add") + "\n\n")
	sb.WriteString(labelStyle.Render("Original:") + " " + subtitleStyle.Render(m.original) + "\n\n")

	if m.enrichment.IsWaste {
		sb.WriteString(wasteTagStyle.Render(" WASTE ") + " " + subtitleStyle.Render("This task doesn't align with any beacon") + "\n\n")
	}

	// Build enrichment preview
	var content strings.Builder

	content.WriteString(labelStyle.Render("Description:") + " " + valueStyle.Render(m.enrichment.Description) + "\n")

	// Beacons
	content.WriteString(labelStyle.Render("Beacons:") + " ")
	if len(m.enrichment.Beacons) > 0 {
		tags := make([]string, len(m.enrichment.Beacons))
		for i, b := range m.enrichment.Beacons {
			tags[i] = tagStyle.Render(b)
		}
		content.WriteString(strings.Join(tags, " "))
	} else {
		content.WriteString(lipgloss.NewStyle().Foreground(mutedColor).Render("none"))
	}
	content.WriteString("\n")

	// Directions
	content.WriteString(labelStyle.Render("Directions:") + " ")
	if len(m.enrichment.Directions) > 0 {
		tags := make([]string, len(m.enrichment.Directions))
		for i, d := range m.enrichment.Directions {
			tags[i] = directionTagStyle.Render(d)
		}
		content.WriteString(strings.Join(tags, " "))
	} else {
		content.WriteString(lipgloss.NewStyle().Foreground(mutedColor).Render("none"))
	}
	content.WriteString("\n")

	// Project, Priority, Dates
	content.WriteString(labelStyle.Render("Project:") + " " + valueOrNone(m.enrichment.Project) + "\n")
	content.WriteString(labelStyle.Render("Priority:") + " " + valueOrNone(m.enrichment.Priority) + "\n")
	content.WriteString(labelStyle.Render("Due:") + " " + valueOrNone(m.enrichment.Due) + " " + lipgloss.NewStyle().Foreground(mutedColor).Render("(hard deadline)") + "\n")
	content.WriteString(labelStyle.Render("Scheduled:") + " " + valueOrNone(m.enrichment.Scheduled) + " " + lipgloss.NewStyle().Foreground(mutedColor).Render("(soft due date)") + "\n")

	// UDAs: Effort, Impact, Estimate, Fun, Blocks
	content.WriteString(labelStyle.Render("Effort:") + " " + formatUDA(m.enrichment.Effort, "E=Easy N=Normal D=Difficult") + "\n")
	content.WriteString(labelStyle.Render("Impact:") + " " + formatUDA(m.enrichment.Impact, "H=High M=Medium L=Low") + "\n")
	content.WriteString(labelStyle.Render("Estimate:") + " " + valueOrNone(m.enrichment.Estimate) + "\n")
	content.WriteString(labelStyle.Render("Fun:") + " " + formatUDA(m.enrichment.Fun, "H=Fun M=Neutral L=Boring") + "\n")
	content.WriteString(labelStyle.Render("Blocks:") + " " + formatBlocks(m.enrichment.Blocks) + "\n")

	// Reasoning
	if m.enrichment.Reasoning != "" {
		content.WriteString("\n" + subtitleStyle.Render(m.enrichment.Reasoning))
	}

	sb.WriteString(boxStyle.Render(content.String()))
	sb.WriteString("\n\n")
	sb.WriteString(helpStyle.Render("[enter/a] Accept  [e] Edit  [s] Skip LLM  [esc/q] Cancel"))

	return sb.String()
}

func (m *AddModel) viewEditing() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("tg add - Edit Mode") + "\n\n")

	for i, name := range m.fieldNames {
		style := labelStyle
		if i == m.editField {
			style = selectedStyle
		}
		sb.WriteString(style.Render(name+":") + " ")
		sb.WriteString(m.textInputs[i].View())
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(helpStyle.Render("[tab] Next field  [shift+tab] Previous  [enter] Save  [esc] Cancel edit"))

	return sb.String()
}

func (m *AddModel) viewDone() string {
	return successStyle.Render("Task added successfully!") + "\n"
}

func (m *AddModel) viewError() string {
	return errorStyle.Render("Error: "+m.err.Error()) + "\n\n" +
		helpStyle.Render("Press any key to exit")
}

func valueOrNone(s string) string {
	if s == "" {
		return lipgloss.NewStyle().Foreground(mutedColor).Render("--")
	}
	return valueStyle.Render(s)
}

func formatUDA(value, hint string) string {
	if value == "" {
		return lipgloss.NewStyle().Foreground(mutedColor).Render("--")
	}
	return valueStyle.Render(value) + " " + lipgloss.NewStyle().Foreground(mutedColor).Render("("+hint+")")
}

func formatBlocks(n int) string {
	if n == 0 {
		return lipgloss.NewStyle().Foreground(mutedColor).Render("0 (not blocking)")
	}
	var hint string
	switch {
	case n >= 6:
		hint = "critical blocker"
	case n >= 3:
		hint = "significant blocker"
	default:
		hint = "minor blocker"
	}
	return valueStyle.Render(fmt.Sprintf("%d", n)) + " " + lipgloss.NewStyle().Foreground(mutedColor).Render("("+hint+")")
}

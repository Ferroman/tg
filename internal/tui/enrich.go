package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bf/tg/internal/config"
	"github.com/bf/tg/internal/llm"
	"github.com/bf/tg/internal/taskwarrior"
)

type enrichState int

const (
	enrichStateLoading enrichState = iota
	enrichStateFetching
	enrichStatePreview
	enrichStateDone
	enrichStateError
)

type EnrichModel struct {
	cfg        *config.Config
	provider   llm.Provider
	twClient   *taskwarrior.Client
	filter     string
	tasks      []taskwarrior.Task
	current    int
	enrichment *llm.Enrichment
	state      enrichState
	spinner    spinner.Model
	err        error
	processed  int
	skipped    int
}

type tasksLoadedMsg struct {
	tasks []taskwarrior.Task
	err   error
}

type taskEnrichedMsg struct {
	enrichment *llm.Enrichment
	err        error
}

type taskModifiedMsg struct {
	err error
}

func NewEnrichModel(cfg *config.Config, provider llm.Provider, filter string) *EnrichModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	return &EnrichModel{
		cfg:      cfg,
		provider: provider,
		twClient: taskwarrior.New(),
		filter:   filter,
		state:    enrichStateLoading,
		spinner:  s,
	}
}

func (m *EnrichModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadTasks(),
	)
}

func (m *EnrichModel) loadTasks() tea.Cmd {
	return func() tea.Msg {
		var tasks []taskwarrior.Task
		var err error

		if m.filter != "" {
			tasks, err = m.twClient.Export(m.filter)
		} else {
			tasks, err = m.twClient.GetUntaggedTasks()
		}

		return tasksLoadedMsg{tasks: tasks, err: err}
	}
}

func (m *EnrichModel) enrichCurrent() tea.Cmd {
	if m.current >= len(m.tasks) {
		return nil
	}

	task := m.tasks[m.current]
	return func() tea.Msg {
		enrichment, err := m.provider.Enrich(
			context.Background(),
			task.Description,
			m.cfg.Beacons,
			m.cfg.Projects,
		)
		return taskEnrichedMsg{enrichment: enrichment, err: err}
	}
}

func (m *EnrichModel) applyEnrichment() tea.Cmd {
	task := m.tasks[m.current]
	enrichment := m.enrichment

	return func() tea.Msg {
		modified := &taskwarrior.Task{
			Description: enrichment.Description,
			Project:     enrichment.Project,
			Priority:    enrichment.Priority,
			Due:         enrichment.Due,
			Effort:      enrichment.Effort,
			Impact:      enrichment.Impact,
			Estimate:    enrichment.Estimate,
			Fun:         enrichment.Fun,
		}

		modified.Tags = append(modified.Tags, enrichment.Beacons...)
		modified.Tags = append(modified.Tags, enrichment.Directions...)

		if enrichment.IsWaste {
			modified.Tags = append(modified.Tags, "waste")
		}

		err := m.twClient.Modify(task.UUID, modified)
		return taskModifiedMsg{err: err}
	}
}

func (m *EnrichModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case spinner.TickMsg:
		if m.state == enrichStateLoading || m.state == enrichStateFetching {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case tasksLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = enrichStateError
			return m, nil
		}
		m.tasks = msg.tasks
		if len(m.tasks) == 0 {
			m.state = enrichStateDone
			return m, tea.Quit
		}
		m.state = enrichStateFetching
		return m, tea.Batch(m.spinner.Tick, m.enrichCurrent())

	case taskEnrichedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = enrichStateError
			return m, nil
		}
		m.enrichment = msg.enrichment
		m.state = enrichStatePreview
		return m, nil

	case taskModifiedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = enrichStateError
			return m, nil
		}
		m.processed++
		return m, m.nextTask()
	}

	return m, nil
}

func (m *EnrichModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case enrichStateLoading, enrichStateFetching:
		if msg.String() == "ctrl+c" || msg.String() == "esc" {
			return m, tea.Quit
		}

	case enrichStatePreview:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			// Done early
			m.state = enrichStateDone
			return m, tea.Quit
		case "enter", "a":
			// Accept and apply
			return m, m.applyEnrichment()
		case "s", "n":
			// Skip this task
			m.skipped++
			return m, m.nextTask()
		}

	case enrichStateError, enrichStateDone:
		return m, tea.Quit
	}

	return m, nil
}

func (m *EnrichModel) nextTask() tea.Cmd {
	m.current++
	if m.current >= len(m.tasks) {
		m.state = enrichStateDone
		return tea.Quit
	}
	m.state = enrichStateFetching
	m.enrichment = nil
	return tea.Batch(m.spinner.Tick, m.enrichCurrent())
}

func (m *EnrichModel) View() string {
	switch m.state {
	case enrichStateLoading:
		return m.viewLoading()
	case enrichStateFetching:
		return m.viewFetching()
	case enrichStatePreview:
		return m.viewPreview()
	case enrichStateDone:
		return m.viewDone()
	case enrichStateError:
		return m.viewError()
	default:
		return ""
	}
}

func (m *EnrichModel) viewLoading() string {
	return fmt.Sprintf("\n  %s Loading tasks...\n", m.spinner.View())
}

func (m *EnrichModel) viewFetching() string {
	task := m.tasks[m.current]
	return fmt.Sprintf("\n  %s Enriching task %d/%d...\n\n  %s\n",
		m.spinner.View(),
		m.current+1,
		len(m.tasks),
		subtitleStyle.Render(task.Description),
	)
}

func (m *EnrichModel) viewPreview() string {
	var sb strings.Builder
	task := m.tasks[m.current]

	sb.WriteString(titleStyle.Render(fmt.Sprintf("tg enrich (%d/%d)", m.current+1, len(m.tasks))) + "\n\n")
	sb.WriteString(labelStyle.Render("Task:") + " " + subtitleStyle.Render(task.Description) + "\n")

	if task.Project != "" {
		sb.WriteString(labelStyle.Render("Project:") + " " + valueStyle.Render(task.Project) + "\n")
	}
	sb.WriteString("\n")

	if m.enrichment.IsWaste {
		sb.WriteString(wasteTagStyle.Render(" WASTE ") + " " + subtitleStyle.Render("This task doesn't align with any beacon") + "\n\n")
	}

	// Build enrichment preview
	var content strings.Builder

	// Note: Description is shown for context but won't be modified (preserves bugwarrior sync)
	content.WriteString(labelStyle.Render("Description:") + " " + lipgloss.NewStyle().Foreground(mutedColor).Render("(unchanged)") + "\n")

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

	content.WriteString(labelStyle.Render("Project:") + " " + valueOrNone(m.enrichment.Project) + "\n")
	content.WriteString(labelStyle.Render("Priority:") + " " + valueOrNone(m.enrichment.Priority) + "\n")
	content.WriteString(labelStyle.Render("Due:") + " " + valueOrNone(m.enrichment.Due) + "\n")

	// UDAs: Effort, Impact, Estimate, Fun
	content.WriteString(labelStyle.Render("Effort:") + " " + formatUDA(m.enrichment.Effort, "E=Easy N=Normal D=Difficult") + "\n")
	content.WriteString(labelStyle.Render("Impact:") + " " + formatUDA(m.enrichment.Impact, "H=High M=Medium L=Low") + "\n")
	content.WriteString(labelStyle.Render("Estimate:") + " " + valueOrNone(m.enrichment.Estimate) + "\n")
	content.WriteString(labelStyle.Render("Fun:") + " " + formatUDA(m.enrichment.Fun, "H=Fun M=Neutral L=Boring") + "\n")

	if m.enrichment.Reasoning != "" {
		content.WriteString("\n" + subtitleStyle.Render(m.enrichment.Reasoning))
	}

	sb.WriteString(boxStyle.Render(content.String()))
	sb.WriteString("\n\n")
	sb.WriteString(helpStyle.Render("[enter/a] Accept  [s/n] Skip  [esc/q] Done"))

	return sb.String()
}

func (m *EnrichModel) viewDone() string {
	return fmt.Sprintf("\n%s\n  Processed: %d  Skipped: %d\n",
		successStyle.Render("Batch enrichment complete!"),
		m.processed,
		m.skipped,
	)
}

func (m *EnrichModel) viewError() string {
	return errorStyle.Render("Error: "+m.err.Error()) + "\n\n" +
		helpStyle.Render("Press any key to exit")
}

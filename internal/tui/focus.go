package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bf/tg/internal/config"
	"github.com/bf/tg/internal/taskwarrior"
)

type focusState int

const (
	focusStateLoading focusState = iota
	focusStateDisplay
	focusStateError
)

type FocusModel struct {
	cfg      *config.Config
	twClient *taskwarrior.Client
	tasks    []taskwarrior.Task
	groups   map[string][]taskwarrior.Task // Groups tasks by focus group or project
	state    focusState
	err      error
}

type focusTasksLoadedMsg struct {
	tasks []taskwarrior.Task
	err   error
}

func NewFocusModel(cfg *config.Config) *FocusModel {
	return &FocusModel{
		cfg:      cfg,
		twClient: taskwarrior.New(),
		state:    focusStateLoading,
		groups:   make(map[string][]taskwarrior.Task),
	}
}

func (m *FocusModel) Init() tea.Cmd {
	return m.loadTasks()
}

func (m *FocusModel) loadTasks() tea.Cmd {
	return func() tea.Msg {
		tasks, err := m.twClient.Export("status:pending")
		return focusTasksLoadedMsg{tasks: tasks, err: err}
	}
}

func (m *FocusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc", "enter":
			return m, tea.Quit
		}

	case focusTasksLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = focusStateError
			return m, nil
		}
		m.tasks = msg.tasks
		m.groupByProject()
		m.state = focusStateDisplay
		return m, nil
	}

	return m, nil
}

func (m *FocusModel) groupByProject() {
	useFocusGroups := len(m.cfg.FocusGroups) > 0

	// Group tasks by focus group or project
	for _, task := range m.tasks {
		project := task.Project
		if project == "" {
			project = "(no project)"
		}

		var groupName string
		if useFocusGroups {
			groupName = m.cfg.GetFocusGroup(project)
			if groupName == "" {
				// Project doesn't match any focus group (excluded or unmatched)
				// Skip this task - don't add to any group
				continue
			}
		} else {
			groupName = project
		}

		m.groups[groupName] = append(m.groups[groupName], task)
	}

	// Sort each group's tasks by urgency (descending)
	for groupName := range m.groups {
		tasks := m.groups[groupName]
		sort.Slice(tasks, func(i, j int) bool {
			return tasks[i].Urgency > tasks[j].Urgency
		})
		m.groups[groupName] = tasks
	}
}

func (m *FocusModel) View() string {
	switch m.state {
	case focusStateLoading:
		return "\n  Loading tasks...\n"
	case focusStateDisplay:
		return m.viewFocus()
	case focusStateError:
		return errorStyle.Render("Error: "+m.err.Error()) + "\n\n" +
			helpStyle.Render("Press any key to exit")
	default:
		return ""
	}
}

func (m *FocusModel) viewFocus() string {
	var sb strings.Builder
	useFocusGroups := len(m.cfg.FocusGroups) > 0

	sb.WriteString(titleStyle.Render("tg focus - Balanced Task List") + "\n\n")

	// Get sorted list of groups
	var groupNames []string
	for name := range m.groups {
		groupNames = append(groupNames, name)
	}
	sort.Strings(groupNames)

	totalSelected := 0
	selectedTasks := []taskwarrior.Task{}

	// Collect tasks respecting quotas
	for _, groupName := range groupNames {
		tasks := m.groups[groupName]
		var quota int
		if useFocusGroups {
			quota = m.cfg.GetFocusGroupQuota(groupName)
		} else {
			quota = m.cfg.GetProjectQuota(groupName)
		}

		// Take up to quota tasks from this group
		count := len(tasks)
		if count > quota {
			count = quota
		}

		for i := 0; i < count; i++ {
			selectedTasks = append(selectedTasks, tasks[i])
		}
		totalSelected += count
	}

	// Sort all selected tasks by urgency
	sort.Slice(selectedTasks, func(i, j int) bool {
		return selectedTasks[i].Urgency > selectedTasks[j].Urgency
	})

	// Summary
	if useFocusGroups {
		sb.WriteString(labelStyle.Render("Groups:") + " ")
	} else {
		sb.WriteString(labelStyle.Render("Projects:") + " ")
	}
	summaryParts := []string{}
	for _, groupName := range groupNames {
		tasks := m.groups[groupName]
		var quota int
		if useFocusGroups {
			quota = m.cfg.GetFocusGroupQuota(groupName)
		} else {
			quota = m.cfg.GetProjectQuota(groupName)
		}
		count := len(tasks)
		if count > quota {
			count = quota
		}
		summaryParts = append(summaryParts, fmt.Sprintf("%s: %d/%d", groupName, count, len(tasks)))
	}
	sb.WriteString(lipgloss.NewStyle().Foreground(mutedColor).Render(strings.Join(summaryParts, ", ")))
	sb.WriteString("\n")
	sb.WriteString(labelStyle.Render("Total focus:") + " " + valueStyle.Render(fmt.Sprintf("%d tasks", totalSelected)))
	sb.WriteString("\n\n")

	// Display tasks - group by focus group if configured, otherwise by project
	currentGroup := ""
	for _, task := range selectedTasks {
		var groupName string
		if useFocusGroups {
			groupName = m.cfg.GetFocusGroup(task.Project)
			if groupName == "" {
				groupName = "(other)"
			}
		} else {
			groupName = task.Project
			if groupName == "" {
				groupName = "(no project)"
			}
		}

		// Show group header when it changes
		if groupName != currentGroup {
			if currentGroup != "" {
				sb.WriteString("\n")
			}
			sb.WriteString(subtitleStyle.Render("─── "+groupName+" ───") + "\n")
			currentGroup = groupName
		}

		// Task line (includes project name when using focus groups)
		taskLine := m.formatTask(task, useFocusGroups)
		sb.WriteString(taskLine + "\n")
	}

	if len(selectedTasks) == 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(mutedColor).Render("  No pending tasks found\n"))
	}

	sb.WriteString("\n")
	sb.WriteString(helpStyle.Render("[q/esc/enter] Exit"))

	return sb.String()
}

func (m *FocusModel) formatTask(task taskwarrior.Task, showProject bool) string {
	var parts []string

	// ID
	idStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Width(4)
	parts = append(parts, idStyle.Render(fmt.Sprintf("%d", task.ID)))

	// Urgency
	urgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Width(6)
	parts = append(parts, urgStyle.Render(fmt.Sprintf("%.1f", task.Urgency)))

	// Priority indicator
	priStyle := lipgloss.NewStyle().Width(2)
	switch task.Priority {
	case "H":
		parts = append(parts, priStyle.Foreground(lipgloss.Color("1")).Render("H"))
	case "M":
		parts = append(parts, priStyle.Foreground(lipgloss.Color("3")).Render("M"))
	case "L":
		parts = append(parts, priStyle.Foreground(lipgloss.Color("4")).Render("L"))
	default:
		parts = append(parts, priStyle.Render(" "))
	}

	// Blocks indicator
	if task.Blocks > 0 {
		blockStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
		parts = append(parts, blockStyle.Render(fmt.Sprintf("B%d", task.Blocks)))
	} else {
		parts = append(parts, "  ")
	}

	// Project name (when using focus groups)
	if showProject && task.Project != "" {
		projStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Width(16)
		proj := task.Project
		if len(proj) > 14 {
			proj = proj[:14] + ".."
		}
		parts = append(parts, projStyle.Render(proj))
	}

	// Description (truncated)
	desc := task.Description
	maxLen := 50
	if showProject {
		maxLen = 35 // Shorter when showing project
	}
	if len(desc) > maxLen {
		desc = desc[:maxLen-3] + "..."
	}
	parts = append(parts, desc)

	// Due/Scheduled indicator
	if task.Due != "" {
		parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(" due:"+task.Due))
	} else if task.Scheduled != "" {
		parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Render(" sched:"+task.Scheduled))
	}

	return strings.Join(parts, " ")
}

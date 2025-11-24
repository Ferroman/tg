package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bf/tg/internal/config"
	"github.com/bf/tg/internal/llm"
	"github.com/bf/tg/internal/tui"
)

func main() {
	if len(os.Args) < 2 {
		// No args, show help
		printHelp()
		os.Exit(0)
	}

	cmd := os.Args[1]

	switch cmd {
	case "add":
		runAdd()
	case "enrich":
		runEnrich()
	case "focus":
		runFocus()
	case "help", "--help", "-h":
		printHelp()
	case "version", "--version", "-v":
		fmt.Println("tg v0.1.0 - Taskwarrior LLM Wrapper")
	default:
		// Passthrough to task
		passthrough()
	}
}

func runAdd() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: tg add <description>")
		os.Exit(1)
	}

	// Join remaining args as description
	description := strings.Join(os.Args[2:], " ")

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	provider, err := llm.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create LLM provider: %v\n", err)
		os.Exit(1)
	}

	model := tui.NewAddModel(cfg, provider, description)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runEnrich() {
	var filter string
	if len(os.Args) > 2 {
		filter = strings.Join(os.Args[2:], " ")
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	provider, err := llm.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create LLM provider: %v\n", err)
		os.Exit(1)
	}

	model := tui.NewEnrichModel(cfg, provider, filter)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runFocus() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	model := tui.NewFocusModel(cfg)
	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func passthrough() {
	// Pass all args to task command
	args := os.Args[1:]
	cmd := exec.Command("task", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}

func printHelp() {
	help := `tg - Taskwarrior LLM Wrapper

USAGE:
    tg <command> [arguments]

COMMANDS:
    add <description>    Add a new task with LLM enrichment
                         The LLM will suggest beacons, directions, project,
                         priority, and due date based on your goals

    enrich [filter]      Batch enrich existing tasks
                         Without filter: enriches all pending tasks without beacon tags
                         With filter: enriches tasks matching the taskwarrior filter

    focus                Show balanced focus list across projects
                         Respects per-project quotas from config
                         Sorted by urgency within each project's quota

    <any task command>   Passes through to taskwarrior
                         Example: tg list, tg done 5, tg project:work

CONFIGURATION:
    Config file: ~/.config/tg/config.yaml

    Example config:
        llm:
          provider: anthropic  # or openai, ollama
          model: claude-sonnet-4-5-20250929
          api_key_env: ANTHROPIC_API_KEY

        projects:
          - name: work
            keywords: ["JIRA-", "company"]
          - name: home
            keywords: ["personal", "home"]

ENVIRONMENT:
    Set your API key in the environment variable specified in config
    (default: ANTHROPIC_API_KEY)

EXAMPLES:
    tg add "Review PR for authentication changes"
    tg enrich
    tg enrich project:work
    tg list +b.great.dev
`
	fmt.Print(help)
}

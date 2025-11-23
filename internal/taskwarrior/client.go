package taskwarrior

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Task represents a Taskwarrior task
type Task struct {
	UUID        string   `json:"uuid,omitempty"`
	ID          int      `json:"id,omitempty"`
	Description string   `json:"description"`
	Project     string   `json:"project,omitempty"`
	Priority    string   `json:"priority,omitempty"`
	Due         string   `json:"due,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Status      string   `json:"status,omitempty"`
	Urgency     float64  `json:"urgency,omitempty"`
	// Custom UDAs
	Effort   string `json:"effort,omitempty"`   // E (easy), N (normal), D (difficult)
	Impact   string `json:"impact,omitempty"`   // H (high), M (medium), L (low)
	Estimate string `json:"est,omitempty"`      // 15m, 30m, 1h, 2h, 4h, 8h, 2d
	Fun      string `json:"fun,omitempty"`      // H (high), M (medium), L (low)
}

// Client interacts with the task command
type Client struct{}

func New() *Client {
	return &Client{}
}

// Add creates a new task and returns its UUID
func (c *Client) Add(t *Task) (string, error) {
	args := []string{"add"}

	// Add description
	args = append(args, t.Description)

	// Add project
	if t.Project != "" {
		args = append(args, "project:"+t.Project)
	}

	// Add priority
	if t.Priority != "" {
		args = append(args, "priority:"+t.Priority)
	}

	// Add due date
	if t.Due != "" {
		args = append(args, "due:"+t.Due)
	}

	// Add custom UDAs
	if t.Effort != "" {
		args = append(args, "effort:"+t.Effort)
	}
	if t.Impact != "" {
		args = append(args, "impact:"+t.Impact)
	}
	if t.Estimate != "" {
		args = append(args, "est:"+t.Estimate)
	}
	if t.Fun != "" {
		args = append(args, "fun:"+t.Fun)
	}

	// Add tags
	for _, tag := range t.Tags {
		args = append(args, "+"+tag)
	}

	cmd := exec.Command("task", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("task add failed: %w\nstderr: %s", err, stderr.String())
	}

	// Extract UUID from output (task outputs "Created task <id>." and we need to get UUID)
	// Run task export to get the UUID of the most recent task
	return c.getLastTaskUUID()
}

func (c *Client) getLastTaskUUID() (string, error) {
	cmd := exec.Command("task", "+LATEST", "export")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get last task: %w", err)
	}

	var tasks []Task
	if err := json.Unmarshal(stdout.Bytes(), &tasks); err != nil {
		return "", fmt.Errorf("failed to parse task output: %w", err)
	}

	if len(tasks) == 0 {
		return "", fmt.Errorf("no tasks found")
	}

	return tasks[0].UUID, nil
}

// Export returns tasks matching the filter
func (c *Client) Export(filter string) ([]Task, error) {
	args := []string{}
	if filter != "" {
		args = append(args, strings.Fields(filter)...)
	}
	args = append(args, "export")

	cmd := exec.Command("task", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("task export failed: %w\nstderr: %s", err, stderr.String())
	}

	var tasks []Task
	if err := json.Unmarshal(stdout.Bytes(), &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse tasks: %w", err)
	}

	return tasks, nil
}

// Modify updates an existing task (does NOT modify description - preserves bugwarrior sync)
func (c *Client) Modify(uuid string, t *Task) error {
	args := []string{uuid, "modify"}

	// NOTE: Description is intentionally NOT updated here
	// Bugwarrior-synced tasks have descriptions from external systems (Jira, GitHub, etc.)
	// that should not be overwritten

	if t.Project != "" {
		args = append(args, "project:"+t.Project)
	}

	if t.Priority != "" {
		args = append(args, "priority:"+t.Priority)
	}

	if t.Due != "" {
		args = append(args, "due:"+t.Due)
	}

	// Add custom UDAs
	if t.Effort != "" {
		args = append(args, "effort:"+t.Effort)
	}
	if t.Impact != "" {
		args = append(args, "impact:"+t.Impact)
	}
	if t.Estimate != "" {
		args = append(args, "est:"+t.Estimate)
	}
	if t.Fun != "" {
		args = append(args, "fun:"+t.Fun)
	}

	for _, tag := range t.Tags {
		args = append(args, "+"+tag)
	}

	cmd := exec.Command("task", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("task modify failed: %w\nstderr: %s", err, stderr.String())
	}

	return nil
}

// GetUntaggedTasks returns tasks without beacon tags (for batch enrichment)
func (c *Client) GetUntaggedTasks() ([]Task, error) {
	// Export pending tasks that don't have any beacon tags
	tasks, err := c.Export("status:pending")
	if err != nil {
		return nil, err
	}

	// Filter to tasks without beacon tags
	var untagged []Task
	for _, task := range tasks {
		hasBeacon := false
		for _, tag := range task.Tags {
			if strings.HasPrefix(tag, "b.") {
				hasBeacon = true
				break
			}
		}
		if !hasBeacon {
			untagged = append(untagged, task)
		}
	}

	return untagged, nil
}

// Run executes an arbitrary task command (for passthrough)
func (c *Client) Run(args []string) error {
	cmd := exec.Command("task", args...)
	cmd.Stdout = nil // Let it go to terminal
	cmd.Stderr = nil
	cmd.Stdin = nil

	return cmd.Run()
}

// RunInteractive executes task command with full terminal passthrough
func (c *Client) RunInteractive(args []string) error {
	cmd := exec.Command("task", args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	// Inherit terminal
	return cmd.Run()
}

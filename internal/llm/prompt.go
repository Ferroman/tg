package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bf/tg/internal/config"
)

func buildPrompt(taskDesc string, beacons []config.Beacon, projects []config.Project) string {
	var sb strings.Builder

	sb.WriteString(`You are a task enrichment assistant. Analyze the given task and suggest appropriate tags and metadata based on the user's personal goal system called "Beacons".

## Beacons System
The user organizes tasks around high-level life goals (Beacons) and specific paths to achieve them (Directions).
Tasks that align with MULTIPLE beacons should be prioritized higher.
Tasks that don't align with ANY beacon should be marked as "waste".

### Available Beacons and their Directions:
`)

	for _, beacon := range beacons {
		sb.WriteString(fmt.Sprintf("\n**%s** (`%s`): %s\n", beacon.Name, beacon.Tag, beacon.Description))
		sb.WriteString("Directions:\n")
		for _, dir := range beacon.Directions {
			sb.WriteString(fmt.Sprintf("  - %s (`%s`): %s\n", dir.Name, dir.Tag, dir.Description))
		}
	}

	if len(projects) > 0 {
		sb.WriteString("\n### Available Projects:\n")
		for _, proj := range projects {
			sb.WriteString(fmt.Sprintf("- %s (keywords: %s)\n", proj.Name, strings.Join(proj.Keywords, ", ")))
		}
	}

	sb.WriteString(fmt.Sprintf(`
## Task Assessment Dimensions

### Effort (mental/cognitive difficulty)
- E (Easy): Quick, straightforward, low cognitive load
- N (Normal): Standard complexity, moderate thinking required
- D (Difficult): Complex, requires deep focus, mentally taxing

### Impact (value delivered)
- H (High): Benefits many people, unlocks future progress, significant consequences if skipped
- M (Medium): Moderate value, helps some people or processes
- L (Low): Limited impact, nice-to-have

### Time Estimate (use pessimistic estimation)
Values: 15m, 30m, 1h, 2h, 4h, 8h, 2d
Ask: "Would X time be enough?" - when answer is "maybe", double it.

### Fun (enjoyment level)
- H (High): Enjoyable, engaging task
- M (Medium): Neutral
- L (Low): Boring, tedious (these get urgency bump to get them done)

### Blocking (how many things/people this unblocks)
- 0: Doesn't block anything
- 1-2: Blocks a few things (e.g., a feature that enables 1-2 other tasks)
- 3-5: Significant blocker (e.g., API that multiple features depend on, review blocking teammates)
- 6+: Critical blocker (e.g., infrastructure change blocking entire team, deployment blocker)

Examples:
- "Deploy API to production" might block=5 (multiple teams waiting)
- "Fix typo in docs" block=0 (nobody waiting)
- "Review PR for authentication" block=2 (author + downstream feature)
- "Set up CI pipeline" block=8 (blocks entire team from deploying)

### Due Dates
- **due**: Hard deadline - must be done by this date (external pressure, meetings, launches)
- **scheduled**: Soft due date - when you'd PREFER to do this task (internal preference)

Use scheduled for tasks without external deadlines but with desired timing.
Only set due when there's actual external pressure/deadline.

## Task to Analyze
"%s"

## Instructions
1. Analyze the task description
2. Identify which Beacons this task contributes to (can be multiple)
3. Identify specific Directions within those Beacons
4. Suggest a project if keywords match
5. Suggest priority (H=high, M=medium, L=low) based on external pressure/deadlines
6. Assess effort, impact, time estimate, fun level, and blocking count
7. Suggest due date only if there's a clear HARD deadline in the task
8. Suggest scheduled date for when you'd prefer to do the task (soft due date)
9. Estimate how many things/people this task unblocks (blocks field)
10. Optionally improve the description to be more actionable
11. If the task doesn't align with any beacon, mark it as waste

Respond with ONLY a JSON object in this exact format:
{
  "description": "improved task description or original if no improvement needed",
  "beacons": ["b.beacon1", "b.beacon2"],
  "directions": ["d.direction1", "d.direction2"],
  "project": "project-name or empty string",
  "priority": "H/M/L or empty string",
  "due": "hard deadline in taskwarrior format (e.g., '2024-12-01', 'friday') or empty string",
  "scheduled": "soft due date - when you'd prefer to work on it (e.g., 'monday', '2024-11-25') or empty string",
  "effort": "E/N/D",
  "impact": "H/M/L",
  "estimate": "15m/30m/1h/2h/4h/8h/2d",
  "fun": "H/M/L",
  "blocks": 0,
  "is_waste": false,
  "reasoning": "brief explanation of the assessment"
}
`, taskDesc))

	return sb.String()
}

func parseEnrichmentResponse(response string) (*Enrichment, error) {
	// Try to extract JSON from the response (in case there's extra text)
	response = strings.TrimSpace(response)

	// Find JSON object boundaries
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")

	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("no valid JSON found in response: %s", response)
	}

	jsonStr := response[start : end+1]

	var enrichment Enrichment
	if err := json.Unmarshal([]byte(jsonStr), &enrichment); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w\nResponse: %s", err, jsonStr)
	}

	return &enrichment, nil
}

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

## Task to Analyze
"%s"

## Instructions
1. Analyze the task description
2. Identify which Beacons this task contributes to (can be multiple)
3. Identify specific Directions within those Beacons
4. Suggest a project if keywords match
5. Suggest priority (H=high, M=medium, L=low) based on external pressure/deadlines
6. Assess effort, impact, time estimate, and fun level
7. Suggest due date only if there's a clear time reference in the task
8. Optionally improve the description to be more actionable
9. If the task doesn't align with any beacon, mark it as waste

Respond with ONLY a JSON object in this exact format:
{
  "description": "improved task description or original if no improvement needed",
  "beacons": ["b.beacon1", "b.beacon2"],
  "directions": ["d.direction1", "d.direction2"],
  "project": "project-name or empty string",
  "priority": "H/M/L or empty string",
  "due": "taskwarrior due format (e.g., 'tomorrow', '2024-12-01', 'eow') or empty string",
  "effort": "E/N/D",
  "impact": "H/M/L",
  "estimate": "15m/30m/1h/2h/4h/8h/2d",
  "fun": "H/M/L",
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

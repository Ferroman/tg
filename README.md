# tg - Taskwarrior LLM Wrapper

A CLI tool that wraps [Taskwarrior](https://taskwarrior.org) and uses LLM to automatically enrich tasks with tags, project assignments, priorities, and due dates based on your personal goal system.

## Features

- **LLM-powered task enrichment** - Analyzes task descriptions and suggests appropriate metadata
- **Beacons system** - Built-in goal-oriented tagging system with customizable life goals (Beacons) and paths to achieve them (Directions)
- **Multi-provider support** - Works with Anthropic Claude, OpenAI, or local Ollama models
- **Interactive TUI** - Beautiful terminal interface built with Bubble Tea
- **Batch enrichment** - Enrich existing tasks (great for bugwarrior-synced tickets)
- **Full passthrough** - Any unrecognized command passes through to `task`

## Installation

```bash
# Clone and build
git clone https://github.com/bf/tg
cd tg
go build -o tg ./cmd/tg

# Install to PATH
sudo cp tg /usr/local/bin/
# or
cp tg ~/.local/bin/
```

## Configuration

Create config file at `~/.config/tg/config.yaml`:

```yaml
llm:
  # Provider: anthropic, openai, or ollama
  provider: anthropic

  # Model (provider-specific)
  model: claude-sonnet-4-5-20250929

  # Environment variable containing API key
  api_key_env: ANTHROPIC_API_KEY

  # For Ollama only:
  # base_url: http://localhost:11434

# Project detection keywords
projects:
  - name: company-a
    keywords: ["JIRA-", "company-a"]
  - name: company-b
    keywords: ["LINEAR-", "company-b"]
  - name: home
    keywords: ["personal", "home"]
```

Set your API key:

```bash
export ANTHROPIC_API_KEY="your-key-here"
# or for OpenAI:
export OPENAI_API_KEY="your-key-here"
```

## Usage

### Add a task with LLM enrichment

```bash
tg add "Review PR for authentication changes"
```

The TUI will show LLM suggestions:
- Improved description
- Beacon tags (e.g., `+b.great.dev`)
- Direction tags (e.g., `+d.sw.design`)
- Project assignment
- Priority (H/M/L)
- Due date
- Effort (E/N/D) - cognitive difficulty
- Impact (H/M/L) - value delivered
- Estimate (15m-2d) - time estimate
- Fun (H/M/L) - enjoyment level

Press `enter` to accept, `e` to edit, `s` to skip enrichment, or `esc` to cancel.

### Batch enrich existing tasks

```bash
# Enrich all pending tasks without beacon tags
tg enrich

# Enrich tasks matching a filter
tg enrich project:work
tg enrich +bugwarrior
```

### Passthrough to Taskwarrior

Any other command passes through to `task`:

```bash
tg list
tg list +b.great.dev
tg done 5
tg project:work
```

## The Beacons System

Tasks are organized around high-level life goals called **Beacons** and specific paths to achieve them called **Directions**.

### Beacons (b.*)

| Tag | Goal |
|-----|------|
| `b.organized` | Be organized |
| `b.great.dev` | Be a great software developer |
| `b.great.devops` | Be a great DevOps engineer |
| `b.great.hardware` | Be great in hardware development |
| `b.great.rel` | Be great in relationships |
| `b.significant.field` | Be significant in a field |
| `b.healthy` | Be healthy |
| `b.prep.draft` | Be prepared to draft |
| `b.war.help` | Help to win the war |

### Directions (d.*)

Each beacon has associated directions. For example, `b.great.dev` includes:
- `d.algo` - Algorithm skills
- `d.sw.design` - Software design
- `d.prog.lang` - Programming languages
- `d.test.write` - Test writing
- `d.dev.tooling` - Development tooling

### Prioritization

- Tasks aligned with **multiple beacons** get higher priority
- Tasks not aligned with any beacon are marked as `waste`
- Use Taskwarrior's urgency model to sort by beacon alignment

```bash
# List tasks by beacon
tg list +b.great.dev

# Find high-value tasks (multiple beacons)
tg list +b.great.dev +b.organized
```

## Task Assessment UDAs

In addition to beacons and directions, the LLM assesses each task on four dimensions that affect urgency:

### Effort (mental difficulty)

| Value | Meaning | Urgency Effect |
|-------|---------|----------------|
| E | Easy - quick, low cognitive load | +1.0 (do easy tasks faster) |
| N | Normal - moderate complexity | 0.0 |
| D | Difficult - requires deep focus | -2.0 (schedule when fresh) |

### Impact (value delivered)

| Value | Meaning | Urgency Effect |
|-------|---------|----------------|
| H | High - benefits many, unlocks progress | +4.0 |
| M | Medium - moderate value | +2.0 |
| L | Low - nice-to-have | 0.0 |

### Time Estimate

| Value | Urgency Effect |
|-------|----------------|
| 15m | +1.0 (quick wins) |
| 30m | +0.5 |
| 1h, 2h | 0.0 |
| 4h | -1.0 |
| 8h | -1.5 |
| 2d | -3.0 (needs dedicated time) |

### Fun (enjoyment level)
| Value | Meaning | Urgency Effect |
|-------|---------|----------------|
| H | Enjoyable task | -0.2 (naturally motivating) |
| M | Neutral | 0.0 |
| L | Boring/tedious | +1.0 (bump to get done) |

## Integration with Bugwarrior

If you use [bugwarrior](https://bugwarrior.readthedocs.io/) to sync tasks from Jira, GitHub, Linear, etc., run batch enrichment after syncing:

```bash
bugwarrior-pull
tg enrich +bugwarrior
```

## License

MIT

# tg - Taskwarrior LLM Wrapper

> [!NOTE]
> This is project to little automation for my personal tasks management system based on my goal system.
> It is highly opinionated and requires some setup.

A CLI tool that wraps [Taskwarrior](https://taskwarrior.org) and uses LLM to automatically enrich tasks with tags, project assignments, priorities, and due dates based on your personal goal system.
Can be used with bugwarrior.


## Features

- **LLM-powered task enrichment** - Analyzes task descriptions and suggests appropriate metadata
- **Beacons system** - Built-in goal-oriented tagging system with customizable life goals (Beacons) and paths to achieve them (Directions)
- **Multi-provider support** - Works with Anthropic Claude, OpenAI, or local Ollama models
- **Interactive TUI** -  Terminal interface built with Bubble Tea
- **Batch enrichment** - Enrich existing tasks (great for bugwarrior-synced tickets)
- **Focus command** - Balanced task list respecting per-project quotas
- **Dual due dates** - Hard deadlines (due) vs soft preferences (scheduled)
- **Blocking awareness** - Track how many things/people a task unblocks
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

## Taskwarrior Setup

Add these UDA (User Defined Attribute) definitions to your `~/.taskrc`:

```bash
# Custom UDAs for task assessment
uda.effort.type=string
uda.effort.label=Effort
uda.effort.values=E,N,D
urgency.uda.effort.E.coefficient=1.0
urgency.uda.effort.D.coefficient=-2.0

# describe impact task has
uda.impact.type=string
uda.impact.label=Impact
uda.impact.values=H,M,L
urgency.uda.impact.H.coefficient=4.0
urgency.uda.impact.M.coefficient=2.0

# describe estimated task duration
uda.est.type=string
uda.est.label=Estimate
uda.est.values=15m,30m,1h,2h,4h,8h,2d
urgency.uda.est.15m.coefficient=1.0
urgency.uda.est.30m.coefficient=0.5
urgency.uda.est.4h.coefficient=-1.0
urgency.uda.est.8h.coefficient=-1.5
urgency.uda.est.2d.coefficient=-3.0

# describe how fun or boring a task is
uda.fun.type=string
uda.fun.label=Fun
uda.fun.values=H,M,L
urgency.uda.fun.H.coefficient=-0.2
urgency.uda.fun.L.coefficient=1.0

uda.blocks.type=numeric
uda.blocks.label=Blocks
urgency.uda.blocks.coefficient=2.0
```

## tg Configuration

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

# Default tasks per project in focus list
default_quota: 2

# Project detection keywords and quotas
projects:
  - name: company-a
    keywords: ["JIRA-", "company-a"]
    quota: 3  # Override default quota for this project
  - name: company-b
    keywords: ["LINEAR-", "company-b"]
  - name: home
    keywords: ["personal", "home"]
    quota: 1

# Focus groups - group multiple projects together for balanced focus
# Without focus_groups, each project gets its own quota (can be overwhelming)
# With focus_groups, related projects share a quota (projects.release, projects.sre → "work")
#
# Pattern matching:
#   "project.*"  - matches project.release, project.dev, etc.
#   "home"  - exact match only
#   "*"     - catch-all for any project
#
# Negative patterns (exclusions):
#   "!project.management" - exclude this specific project
#   Projects excluded from ALL groups are completely hidden from focus
#
# Groups are checked in order - first match wins
focus_groups:
  - name: work
    patterns: ["project.*", "!project.management" ]
    quota: 5  # Top 5 tasks across ALL work projects
  - name: personal
    patterns: ["personal.*", "family.*", "home.*"]
    quota: 3
  - name: war
    patterns: ["war.*"]
    quota: 2
  - name: other
    patterns: ["*", "!project.management"]  # Catch-all, but still exclude project.management
    quota: 2
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
- Due (hard deadline)
- Scheduled (soft due date - when you'd prefer to do it)
- Effort (E/N/D) - cognitive difficulty
- Impact (H/M/L) - value delivered
- Estimate (15m-2d) - time estimate
- Fun (H/M/L) - enjoyment level
- Blocks (number) - how many things/people this unblocks

Press `enter` to accept, `e` to edit, `s` to skip enrichment, or `esc` to cancel.

### Batch enrich existing tasks

```bash
# Enrich all pending tasks without beacon tags
tg enrich

# Enrich tasks matching a filter
tg enrich project:work
tg enrich +bugwarrior
```

**Safety features:**
- **Existing projects are preserved** - Tasks with a project won't have it overwritten
  - Preview shows: `Project: project (preserved)`
  - Great for bugwarrior-synced tasks that already have projects from Jira/GitHub
- **Description never modified** - Only tags and metadata are updated
- **Edit before accepting** - Press `e` to edit any suggested values
- **Skip option** - Press `s` to skip enrichment for a task

### Focus list (balanced view across projects)

```bash
tg focus
```

Shows a balanced task list respecting quotas:
- **With focus_groups**: Groups projects together (e.g., all `project.*` projects share one quota)
  - Problem: You have `project.release`, `project1.sre`, `project.dev` - with quota 2 each = 6 tasks from `project`
  - Solution: Group them under "work" with quota 5 total
- **Without focus_groups**: Uses individual project quotas
- Sorted by urgency globally, displayed with group headers
- Great for deciding what to work on next without overwhelming yourself

**Example output with focus groups:**
```
Groups: work: 3/78, personal: 2/18
Total focus: 5 tasks

─── work ───
92   24.2 H      project.dev      Unified logging implementation
79   20.9 H  B3  project.security Prepare API DAST test for services
145  19.0 M      project.sre      Create deployment rollback script

─── personal ───
87   23.6 H      personal.job    Prepare and send invoices  due:20250902
120  18.2 H      family.son      Schedule training appointment
```

**Key observations:**
- 5 tasks from "work" group (across all projects)
- Tasks with `B3`, `B5` are blocking 3 and 5 things respectively
- Sorted by urgency (27.2 → 7.4), grouped for context
- `project.management` is excluded (not shown)

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

### Blocks (blocking importance)

How many things/people this task unblocks. The LLM infers this from task descriptions.

| Value | Meaning | Examples |
|-------|---------|----------|
| 0 | Not blocking anything | "Fix typo in docs", "Research options" |
| 1-2 | Minor blocker | "Complete feature X" (blocks testing) |
| 3-5 | Significant blocker | "Deploy API" (blocks multiple features) |
| 6+ | Critical blocker | "Fix production outage" (blocks entire team) |

**Note:** See Taskwarrior Setup section for the required `.taskrc` configuration with urgency coefficient.

## Due Dates

tg distinguishes between two types of due dates:

### Due (hard deadline)
External pressure, real deadlines. Examples:
- Meeting deadlines
- Launch dates
- Client commitments

### Scheduled (soft due date)
When you'd **prefer** to work on this task. Internal preference without external pressure. Examples:
- "I'd like to finish this by Friday"
- "Would be nice to have before the sprint ends"

The LLM suggests both based on task context. Only `due` is set when there's actual external pressure.

## Integration with Bugwarrior

If you use [bugwarrior](https://bugwarrior.readthedocs.io/) to sync tasks from Jira, GitHub, Linear, etc., run batch enrichment after syncing:

```bash
bugwarrior-pull
tg enrich +bugwarrior
```

## License

MIT

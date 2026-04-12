# Graft Skills

Agent skills (Claude Code `SKILL.md` format) that teach AI coding agents how to use Graft correctly.

## Available skills

| Skill | When it activates |
|-------|-------------------|
| [`graft`](./graft/SKILL.md) | Go code importing `github.com/delavalom/graft` or any sub-package. Enforces functional options, generic `NewTool`, role constants, handoff semantics, and the runtime-swap pattern. |

## Install

### User-wide (all your projects)

```bash
git clone https://github.com/delavalom/graft.git /tmp/graft
cp -r /tmp/graft/skills/graft ~/.claude/skills/
```

### Project-only

From a repo that uses Graft:

```bash
mkdir -p .claude/skills
curl -L https://raw.githubusercontent.com/delavalom/graft/main/skills/graft/SKILL.md \
  -o .claude/skills/graft/SKILL.md
```

Claude Code loads skills from `~/.claude/skills/` (global) and `.claude/skills/` (per-repo) on session start.

## Format

Each skill is a single `SKILL.md` file with YAML frontmatter:

```markdown
---
name: graft
description: Use when... (activation trigger)
---

# Skill body
```

The `description` is what Claude Code matches against the current task to decide whether to load the skill. Keep it specific — generic descriptions cause false activations.

## Contributing

When Graft's conventions change, update `skills/graft/SKILL.md` in the same PR. The skill is the canonical source of truth for agents; drift from `CLAUDE.md` will cause bad code generation.

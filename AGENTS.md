# Agent Instructions

Entry point for all AI coding agents working in this repository (Cursor, Claude Code, Codex, GitHub Copilot, etc.).

## Project

**go-json-xml-toolbox** — Go utilities for working with JSON and XML (parse, format, validate, convert, and related helpers).

See [README.md](./README.md) for scope and usage.

## Repository layout

```
cmd/              CLI tools (as added)
internal/         JSON/XML library code (as added)
docs/agent/       Agent rules, plan, and progress tracking
skills/           Reusable agent workflows (SKILL.md)
```

- **Module:** `go-json-xml-toolbox`
- **Go version:** 1.25+ (when `go.mod` is added)
- **Production code:** `cmd/`, `internal/`, or package roots at repo level
- **Tests:** `*_test.go` alongside source, or under `tests/`

## Required reading (before any code change)

Read these in order:

1. [docs/agent/rules/edit-first.md](./docs/agent/rules/edit-first.md) — operating contract and execution loop
2. [docs/agent/rules/coding-standards.md](./docs/agent/rules/coding-standards.md) — engineering principles

## Skills (workflows)

For structured, multi-step workflows, read and follow the matching skill:

| Skill | When to use | Path |
|-------|-------------|------|
| Edit-first loop | User wants a minimal, planned change with progress logging | [skills/edit-first-loop/SKILL.md](./skills/edit-first-loop/SKILL.md) |

## Planning and progress

| Document | Purpose |
|----------|---------|
| [docs/agent/plan.md](./docs/agent/plan.md) | Project plan and architecture decisions |
| [docs/agent/progress.md](./docs/agent/progress.md) | Development log — update after each meaningful change |

## Quick rules (summary)

- Prefer editing existing files over creating new ones.
- Read relevant code and tests before changing anything.
- Keep changes minimal and focused.
- Run tests and linters after changes.
- No dead code, unused imports, or commented-out blocks.
- Security over convenience.

For full details, see the rule files linked above.

## Tool notes

| Tool | How it loads this repo |
|------|------------------------|
| **Codex / Copilot / Gemini** | Reads `AGENTS.md` automatically |
| **Claude Code** | Reads `AGENTS.md`; optionally symlink `CLAUDE.md` → `AGENTS.md` |
| **Cursor** | Reads `AGENTS.md` and skills under `skills/` |

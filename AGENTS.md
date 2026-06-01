# Agent Instructions

Entry point for all AI coding agents working in this repository (Cursor, Claude Code, Codex, GitHub Copilot, etc.).

## Project

**go-json-xml-tool** — Go module (`jsonxmltool` package) with HTTP helpers for JSON, XML, uploads, and related web utilities.

Repository: `git@github.com:dunky-star/go-json-xml-tool.git`

See [README.md](./README.md) for API overview and examples.

## Repository layout

```
jsonxmltool.go      Core Kit API (JSON, XML, uploads, slugs, etc.)
jsonxmltool_test.go Tests
testdata/           PNG/JPEG fixtures (see testdata/README.md)
docs/agent/         Agent rules, plan, and progress tracking
skills/             Reusable agent workflows (SKILL.md)
.github/workflows/  CI (Go 1.26)
```

- **Module:** `github.com/dunky-star/go-json-xml-tool`
- **Go version:** 1.26+
- **Production code:** package root (`jsonxmltool.go`) and future `cmd/` / `internal/` as needed
- **Tests:** `*_test.go` alongside source; fixtures in `testdata/`

## Required reading (before any code change)

Read these in order:

1. [docs/agent/rules/edit-first.md](./docs/agent/rules/edit-first.md) — operating contract and execution loop
2. [docs/agent/rules/coding-standards.md](./docs/agent/rules/coding-standards.md) — engineering principles

## Skills (workflows)

| Skill | When to use | Path |
|-------|-------------|------|
| Edit-first loop | Minimal planned change with progress logging | [skills/edit-first-loop/SKILL.md](./skills/edit-first-loop/SKILL.md) |

## Planning and progress

| Document | Purpose |
|----------|---------|
| [docs/agent/plan.md](./docs/agent/plan.md) | Architecture and scope |
| [docs/agent/progress.md](./docs/agent/progress.md) | Development log |

## Quick rules (summary)

- Prefer editing existing files over creating new ones.
- Read relevant code and tests before changing anything.
- Keep changes minimal and focused.
- Run `go test ./...` and `go vet ./...` after changes.
- No dead code, unused imports, or commented-out blocks.
- Security over convenience.

For full details, see the rule files linked above.

## Tool notes

| Tool | How it loads this repo |
|------|------------------------|
| **Codex / Copilot / Gemini** | Reads `AGENTS.md` automatically |
| **Claude Code** | Reads `AGENTS.md`; optionally symlink `CLAUDE.md` → `AGENTS.md` |
| **Cursor** | Reads `AGENTS.md` and skills under `skills/` |

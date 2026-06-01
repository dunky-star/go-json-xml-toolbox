---
name: edit-first-loop
description: Runs a structured edit-first workflow — read, summarize, plan, minimal diff, progress log. Use when the user asks for a careful incremental change, mentions edit-first, or wants a small planned diff with progress tracking.
---

# Edit-First Loop

Structured workflow for making minimal, well-reasoned code changes.

## When to use

- User asks for a small, focused change
- User mentions "edit-first" or wants a planned diff
- User wants progress logged after the change

## Instructions

1. Read the files the user mentions and any obvious related tests.
2. Summarize current behavior in 3–5 sentences.
3. Propose a small, concrete plan (1–3 steps).
4. Show a minimal diff for the change instead of full files.
5. Suggest an update line for [docs/agent/progress.md](../../docs/agent/progress.md), including date, file(s), and purpose.

## References

- Operating contract: [docs/agent/rules/edit-first.md](../../docs/agent/rules/edit-first.md)
- Coding standards: [docs/agent/rules/coding-standards.md](../../docs/agent/rules/coding-standards.md)
- Entry point: [AGENTS.md](../../AGENTS.md)

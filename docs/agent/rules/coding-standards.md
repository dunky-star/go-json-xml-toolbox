# Coding Standards

You are a senior engineer working in this repository.

## Core principles

- **DRY** — Don't Repeat Yourself
- **KISS** — Keep It Simple
- **YAGNI** — You Aren't Gonna Need It
- **SoC** — Separation of Concerns
- **SRP** — Single Responsibility Principle

## Code quality

- Prefer editing existing code over rewriting from scratch
- Before any change: read relevant files, search for usages, propose a short plan
- Keep code clean, secure, and succinct
- No dead code, unused imports, or commented-out blocks left behind
- Imports must not appear inside functions or methods
- Security always takes priority over convenience

## Testing

- Every change that affects behavior should be covered by or aligned with tests
- After changes, suggest an update to [docs/agent/progress.md](../progress.md) describing what changed and why

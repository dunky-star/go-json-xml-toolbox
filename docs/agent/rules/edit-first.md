# Edit-First Operating Contract

Mandatory rules for any AI coding agent working in this repository.

## Primary policy — edit-first

- Always prefer editing existing files over creating new ones.
- Create a new file only when no existing file logically fits the change.
- Incremental changes are preferred over large rewrites.

## Execution loop

**Before any change:**

1. Read relevant files
2. Search for related logic and usages
3. Plan the minimal correct change
4. Edit the existing file

**After each change:**

1. Run tests
2. Lint
3. Summarize the change
4. Log progress in [docs/agent/progress.md](../progress.md)

## Repository conventions

- Production code lives in `cmd/`, `internal/`, or package roots at repo level
- Tests live in `*_test.go` files alongside source, or under `tests/`
- Every behavioral change should include or align with a test when feasible

## Logging and traceability

- Meaningful progress → [docs/agent/progress.md](../progress.md)
- Planning decisions → [docs/agent/plan.md](../plan.md)

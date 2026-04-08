AI Development System for `go-core`

This repository includes an AI-assisted workflow for planning, implementing, fixing, testing, and reviewing changes in `go-core`.

The `.ai` folder is intended to be consumed by `forge`, so prompts and context should optimize for bounded, repository-aware execution.

Unlike service repositories, `go-core` is a reusable framework/library. AI context for this repo must preserve:

- domain-agnostic design
- framework-oriented boundaries
- minimal public API growth
- no product-specific assumptions

Primary lens

Read `.ai/go-core.md` first. It defines the highest-priority framing for AI work in this repository.

Repository positioning

`go-core` is the standard application foundation for Go services in this ecosystem:

- startup and runtime composition should live here
- service business logic should not
- generic utilities should go to `utils-shared`, not `go-core`

Folder Structure

`.ai/context`
Contains framework context, architecture rules, ownership boundaries, and repository map.

`.ai/prompts`
Defines AI roles for planning, execution, fixes, tests, and review.

`.ai/tasks`
Contains task definitions for bounded changes.

Workflow

1. Create a task

`.ai/tasks/refactor_example.md`

2. Plan the task

`forge plan refactor_example`

3. Execute the task

`forge run refactor_example`

4. Run verification

`go test ./...`

5. Fix follow-up issue

`forge fix refactor_example --error "error message"`

6. Review diff or path

`forge review config/load.go`

Safety Guards

Tasks must define:

- goal
- scope layers
- allowed paths
- constraints

AI must not introduce framework behavior that should live in downstream services.

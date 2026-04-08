# AI-Assisted Development System for `go-core`

## Overview

This repository uses an AI workflow to help maintain `go-core` as a reusable infrastructure foundation.

The workflow exists to keep changes:

- safe
- bounded
- framework-oriented
- domain-agnostic
- foundation-first

## Layer Model

```text
Engineer
   |
   v
Forge CLI
   |
   v
Prompt Layer
(.ai/prompts)
   |
   v
Task Layer
(.ai/tasks)
   |
   v
Context Layer
(.ai/context)
   |
   v
AI Model
```

## Repository Rules

`go-core` owns:

- startup composition and runtime wiring
- config loading and validation
- lifecycle and shutdown behavior
- transport wrappers
- logging, metrics, tracing baseline
- infra connectors and contracts
- migration helper behavior
- technical error contract and mapping

`go-core` does not own:

- business fields
- service schema choices
- product-specific workflow assumptions
- service-specific default names such as `transaction`, `history`, `payment`
- generic utilities that belong in `utils-shared`

## Prompt Roles

- `breakdown.md`: planner
- `execute.md`: framework engineer
- `fix.md`: debugger
- `test.md`: tester
- `review.md`: reviewer

## Execution Principles

- prefer safe evolution
- allow bounded refactors when they improve the framework shape
- avoid hidden side effects
- preserve exported behavior when reasonable, but broad backward compatibility is not yet the top constraint
- keep defaults generic
- keep service-specific semantics out of framework code and docs
- keep generic utilities out of `go-core`
- target Go `1.24`

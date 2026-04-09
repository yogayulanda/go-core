Status: done

Task: improve resilience contracts

Goal:
connect retry and timeout helpers more clearly to the foundation’s logging and operational guidance

Scope Layers:

runtime
tests
docs
ai

Allowed Paths:

resilience/
docs/
.ai/
README.md

Constraints:

keep APIs small
prefer additive guidance and light hooks over large redesign
align resilience usage with service and DB flow observability

Expected Output:

- clearer resilience usage story
- better tests and docs for retry/timeout behavior
- `.ai` context aligned with the intended usage

Implemented Notes:

- retry now supports additive retry hooks and timeout now has an observed variant
- resilience package provides logger-backed service-log hooks for retry and timeout events
- tests cover retry hook scheduling, timeout hook behavior, and service-log helper output
- reliability guidance now points services to the logger-aware resilience path when diagnostics are needed

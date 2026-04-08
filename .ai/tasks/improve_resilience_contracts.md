Status: pending

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

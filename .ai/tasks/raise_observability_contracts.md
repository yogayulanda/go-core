Status: done

Task: raise observability contracts

Goal:
balance service and DB observability with additive metric contracts that match the new logging flavors

Scope Layers:

observability
docs
tests
ai

Allowed Paths:

observability/
docs/
.ai/
README.md

Constraints:

metric names must be additive and treated as stable once introduced
keep transaction observability intact
align docs with metric names and intended usage

Expected Output:

- additive service and DB metrics
- docs explaining when logs, metrics, and tracing work together
- `.ai` context updated for future `forge` usage

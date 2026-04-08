Status: done

Task: improve runtime orchestration

Goal:
raise `app/` and `server/` to the same foundation quality as the newer logging and bootstrap guidance

Scope Layers:

runtime
tests
docs
ai

Allowed Paths:

app/
server/
docs/
.ai/
README.md

Constraints:

preserve explicit lifecycle ownership
avoid hidden background behavior
align startup, shutdown, readiness, and logging contracts

Expected Output:

- clearer orchestration ownership
- better lifecycle and readiness signals
- docs/tests aligned with the runtime behavior

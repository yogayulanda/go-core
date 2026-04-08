Status: done

Task: clarify repository boundary and align docs

Goal:
make boundary guidance consistent across README, docs, legacy AI shims, and `.ai`

Scope Layers:

docs
ai

Allowed Paths:

.ai/
docs/
README.md
AI_RULES.md
CONTEXT.md

Constraints:

keep `go-core` foundation-oriented
allow selected platform-standard technical contracts only when intentionally standardized
do not move business rules into `go-core`
keep `utils-shared` as the home for generic utilities

Expected Output:

- one consistent repository boundary narrative
- legacy AI files reduced to compatibility shims or concise summaries
- docs that distinguish `dbtx` from transaction observability

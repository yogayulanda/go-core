Status: done

Task: formalize messaging and outbox service pattern

Goal:
make direct publish vs outbox usage explicit for consuming services

Scope Layers:

docs
examples
ai

Allowed Paths:

docs/
examples/
.ai/
README.md
messaging/

Constraints:

keep worker ownership explicit in the consuming service
do not introduce hidden startup behavior
prefer additive helpers or docs over broad API churn

Expected Output:

- a blessed messaging pattern document
- example of write + outbox in one SQL transaction
- `.ai` context updated to point to the pattern

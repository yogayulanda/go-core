Status: done

Task: improve messaging and outbox runtime behavior

Goal:
bring publisher, consumer, and outbox worker runtime behavior up to the same observability and ownership standard as the rest of the foundation

Scope Layers:

runtime
tests
docs
ai

Allowed Paths:

messaging/
app/
observability/
examples/
docs/
.ai/
README.md

Constraints:

keep service-owned startup explicit
avoid hidden worker startup behavior
prefer additive metrics and helpers over broad API churn

Expected Output:

- messaging and outbox runtime emit aligned `ServiceLog`
- additive messaging and outbox metrics are available
- docs/examples reflect explicit ownership and current runtime behavior

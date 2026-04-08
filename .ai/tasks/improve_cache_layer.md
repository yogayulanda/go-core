Status: pending

Task: improve cache layer ergonomics and observability

Goal:
make Redis and Memcached feel like first-class foundation dependencies, not just adapters

Scope Layers:

runtime
tests
docs
ai

Allowed Paths:

cache/
docs/
.ai/
README.md

Constraints:

keep optional dependency semantics explicit
improve operational visibility without adding hidden behavior
align cache health and observability with the rest of the foundation

Expected Output:

- clearer cache operational behavior
- stronger tests for health/failure expectations
- docs aligned with cache usage in service foundations

Status: done

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

Implemented Notes:

- Redis and Memcached now emit aligned `cache_connect` `ServiceLog` on startup success and failure
- cache health semantics are documented explicitly, including Memcached cache-miss-as-healthy behavior
- readiness tests cover enabled, disabled, healthy, and failed cache dependency states
- docs and README now describe enabled caches as explicit required runtime dependencies

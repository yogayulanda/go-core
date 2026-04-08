Task: improve config and database defaults

Goal:

Remove service-biased defaults and make config/database behavior more generic for all consuming services.

Scope Layers:

config
runtime
docs
tests

Allowed Paths:

config/
app/
database/
migration/
README.md
docs/

Constraints:

prefer safe evolution over accidental churn
do not hardcode service-specific DB aliases
do not introduce product-specific logic
do not move generic utility helpers into `go-core`
add tests for any config behavior changes

Expected Output:

- generic migration defaults or explicit migration requirements
- database config behavior aligned with service-supplied aliases
- updated README and tests

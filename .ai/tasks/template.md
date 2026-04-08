Status: pending

Task Context

Task:
describe the task

Goal:
what should be achieved

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
server/
dbtx/
errors/
logger/
observability/
security/
resilience/
docs/
README.md

Do NOT modify:

downstream services
product-specific schemas
generic utility code better suited for `utils-shared`

Constraints:

keep go-core domain-agnostic
keep it foundation-oriented, not utility-oriented
allow approved platform-standard technical contracts when clearly scoped
prefer safe evolution; bounded refactor is allowed when it improves the framework
update tests and docs with code changes

Expected Output:

describe expected framework result

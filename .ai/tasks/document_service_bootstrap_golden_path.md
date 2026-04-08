Status: done

Task: document canonical service bootstrap path

Goal:
provide one clear onboarding path for service teams consuming `go-core`

Scope Layers:

docs
examples
templates

Allowed Paths:

docs/
examples/
templates/
README.md

Constraints:

show `config.Load -> Validate -> app.New -> transport wiring -> server.Run`
keep transaction observability opt-in for transaction-oriented services only
do not imply every service must use every optional dependency

Expected Output:

- golden-path service bootstrap guidance
- examples and templates aligned with the intended onboarding flow

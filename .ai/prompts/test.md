# Prompt: Write Tests

> **When to use:** Adding or updating tests for a module or exported behavior.

---

```
You are a senior Go engineer writing tests for go-core.

go-core is an infrastructure foundation library for Go microservices.
Tests here guard public contracts that all downstream services depend on.

Read: .ai/context.md

== SCOPE ==
{{ Name the module/function to test, or "full coverage" for the whole file }}

== RULES ==
- Table-driven tests for multiple cases
- Isolate external dependencies: DB → sqlmock, HTTP → httptest, functions → interface mocks
- Cover for every exported symbol:
  ✅ Happy path
  ✅ All documented error paths
  ✅ Edge cases (nil, empty, zero, boundary values)
- Test names: Test<FunctionName>_<Scenario>
- Test public behavior, not internal implementation
- Must pass: go test ./...
- No real external services required (DB, Redis, Kafka, etc.)

== OUTPUT ==
1. Complete test file — paste-ready, all imports included
2. Coverage summary — what is covered, what edge cases included
3. Gaps — any exported behavior that cannot be tested without refactoring
```

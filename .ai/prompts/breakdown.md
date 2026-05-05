# Prompt: Task Breakdown

> **When to use:** Before writing any code. Plan the change, assess risk, define scope.

---

```
You are a senior Go framework engineer planning a change to go-core.

go-core is an infrastructure foundation library for Go microservices.
NOT a business service. NOT a generic utils repo. Domain-agnostic.
Every change here affects all downstream consuming services.

Read: .ai/context.md

== TASK ==
{{ Describe the requested change }}

== DELIVER ==

1. BOUNDARY CHECK
   Does this belong in go-core or in a consuming service?
   Is it domain-agnostic? Can multiple services use it without modification?
   → If NOT in go-core, explain where it belongs instead. Stop here.

2. AFFECTED LAYERS
   [ ] config / env contract
   [ ] app bootstrap / lifecycle
   [ ] transport (grpc / gateway)
   [ ] security / auth
   [ ] errors contract
   [ ] logger / observability
   [ ] database / dbtx / migration
   [ ] cache / messaging / resilience

3. CONTRACT RISK
   [ ] Public API change → semver impact (patch / minor / major)
   [ ] Config/env change → MIGRATION.md required
   [ ] Runtime behavior change → test coverage required
   [ ] Metric name/label change → dashboard coordination required

4. IMPLEMENTATION STEPS
   Ordered from lowest to highest risk.

5. FILES TO CHANGE
   List specific files and why.

6. TESTS REQUIRED

7. DOCS TO UPDATE
   [ ] README.md  [ ] MIGRATION.md  [ ] CHANGELOG.md  [ ] docs/  [ ] .ai/

8. ACCEPTANCE CRITERIA

Do not implement yet.
```

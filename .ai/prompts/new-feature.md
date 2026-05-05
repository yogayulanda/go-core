# Prompt: Add New Feature / Module

> **When to use:** Adding a new module or feature to go-core.

---

```
You are a senior Go framework engineer adding a new feature to go-core.

go-core is an infrastructure foundation library for Go microservices.
NOT a business service. NOT a generic utils repo. Target Go 1.24.

Read: .ai/context.md, .ai/modules.md, .ai/conventions.md

== FEATURE ==
{{ Describe the new feature or module }}

== BOUNDARY CHECK — answer before implementing ==
1. Domain-agnostic? (usable by multiple services without modification)
2. Infrastructure/framework concern, not business logic?
3. NOT a generic utility that belongs in utils-shared?

All YES → implement. Any NO → explain where it belongs instead. Stop here.

== IMPLEMENTATION ==

1. PACKAGE & FILE STRUCTURE

2. PUBLIC API
   Exported types, functions, interfaces. Minimal surface. Options pattern for extensibility.

3. CODE — full file(s), paste-ready

4. TESTS — full test file, paste-ready

5. APP INTEGRATION (if needed)
   Injection into App struct or lifecycle? Show app.go change.

6. DOCS TO UPDATE
   - .ai/modules.md — add module entry
   - .ai/context.md — add to module table if significant
   - .ai/integrations.md — add env config if new vars needed
   - CHANGELOG.md

== CONSTRAINTS ==
- ctx context.Context always first parameter
- Options pattern for constructors with more than 2 config values
- Register cleanup in lifecycle if resource needs closing
- No goroutines started automatically
- No hardcoded names (service, DB alias, topic)
- No new external dependencies without justification
```

Status: done

Task: improve config ergonomics and validation DX

Goal:
add structured config validation guidance without breaking the compact public validation path

Scope Layers:

config
docs
tests
ai

Allowed Paths:

config/
docs/
.ai/
README.md

Constraints:

keep `Validate()` available as the compact public entry point
make structured validation additive
update docs and tests with any validation behavior change

Expected Output:

- structured validation issues
- grouped configuration onboarding guidance
- tests proving both compact and structured validation paths

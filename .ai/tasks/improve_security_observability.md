Status: pending

Task: improve security observability

Goal:
make auth extraction and JWT verification easier to operate, diagnose, and document

Scope Layers:

runtime
tests
docs
ai

Allowed Paths:

security/
server/grpc/
docs/
.ai/
README.md

Constraints:

do not weaken auth behavior
keep external auth failures sanitized
improve internal diagnosability with explicit logging and guidance

Expected Output:

- stronger auth observability and docs
- tests covering operationally relevant auth behavior
- clearer separation between generic extractor mode and JWT verifier mode

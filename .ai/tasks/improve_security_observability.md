Status: done

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

Implemented Notes:

- gRPC startup now emits `auth_config` with mode and policy metadata
- auth interceptor logs stable `auth_request` failure reasons internally while keeping client-facing auth failures sanitized
- metadata extraction mode and JWT verification mode are documented more explicitly
- tests cover sanitized auth failures, metadata-mode injection, and verifier config metadata

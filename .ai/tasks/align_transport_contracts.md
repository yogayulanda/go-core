Status: done

Task: align transport contracts

Goal:
bring gRPC and gateway behavior in line with the newer foundation contracts for logging, metrics, request ID, and errors

Scope Layers:

runtime
tests
docs
ai

Allowed Paths:

server/grpc/
server/gateway/
docs/
.ai/
README.md

Constraints:

keep transport behavior explicit
preserve compact external error responses
align request ID, metrics, and service logging across transports

Expected Output:

- better transport consistency
- tests covering aligned behavior
- docs that explain the transport contract clearly

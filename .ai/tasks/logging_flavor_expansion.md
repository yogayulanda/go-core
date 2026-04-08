Status: done

Task: expand logging flavors

Goal:
introduce `ServiceLog` and `DBLog` as foundation logging contracts while keeping `TransactionLog` as the platform-standard transaction contract

Scope Layers:

runtime
docs
tests
ai

Allowed Paths:

logger/
database/
docs/
examples/
.ai/
README.md

Constraints:

keep `Info/Error/Debug/Warn`
keep `EventLog`
keep `TransactionLog` scoped to transaction-oriented services
keep top-level fields small and stable
use `Metadata` as the extension area
do not create a competing source of truth outside `.ai/`

Expected Output:

- logger API with `LogService(...)` and `LogDB(...)`
- initial framework adoption in DB initialization
- docs and `.ai` aligned on the 3 logging flavors

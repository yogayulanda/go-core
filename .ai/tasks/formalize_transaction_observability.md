Status: done

Task: formalize transaction observability contract

Goal:
lock the transaction monitoring contract so downstream transaction-oriented services implement it consistently

Scope Layers:

runtime
docs
tests

Allowed Paths:

logger/
observability/
docs/
README.md

Constraints:

keep `TransactionLog`, `LogTransaction(...)`, and `app_transaction_total`
keep top-level fields small and stable
keep `UserID` top-level
use `Metadata` as the extension area
do not conflate transaction observability with `dbtx`

Expected Output:

- explicit contract guidance for `TransactionLog`
- docs that define field meaning and usage boundaries
- tests updated if contract behavior changes

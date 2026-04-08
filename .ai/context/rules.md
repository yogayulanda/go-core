Rules for AI modifications

modify only necessary files

do not introduce product-specific assumptions

do not rename public interfaces without strong reason

keep docs, tests, and code aligned

prefer minimal surface-area changes

never move service business logic into `go-core`

prefer additive changes over breaking changes

internal refactors are allowed when they make the foundation cleaner, safer, or more coherent

keep runtime wiring explicit; avoid hidden lifecycle behavior

sanitize external-facing errors and preserve internals in logs

do not turn `go-core` into a shared utils repository

prefer repository-aware examples and docs over product-specific examples

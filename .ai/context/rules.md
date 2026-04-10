Rules for AI modifications

modify only necessary files

do not rename public interfaces without strong reason

keep docs, tests, and code aligned
keep versioning and migration notes aligned with public contract changes

prefer minimal surface-area changes

internal refactors are allowed when they make the foundation cleaner, safer, or more coherent

sanitize external-facing errors and preserve internals in logs

prefer repository-aware examples and docs over product-specific examples

respect task `Allowed Paths` and stated scope layers

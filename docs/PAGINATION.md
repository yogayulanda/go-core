# Pagination

Default for public APIs: cursor-based.

For transaction-history-service (current): offset cursor string is acceptable (`cursor=offset`) with:
- capped `pageSize`
- deterministic sort
- `hasMore` + `nextCursor`

Keep strategy consistent per endpoint.

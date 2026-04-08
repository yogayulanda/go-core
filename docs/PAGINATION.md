# Pagination

Default for public APIs: cursor-based.

For internal or legacy endpoints, offset-style cursor string is acceptable (`cursor=offset`) with:
- capped `pageSize`
- deterministic sort
- `hasMore` + `nextCursor`

Keep strategy consistent per endpoint.

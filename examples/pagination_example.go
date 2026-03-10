package examples

import (
	"strconv"
	"strings"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

type OffsetPageRequest struct {
	Cursor   string
	PageSize int
}

type OffsetPageResult struct {
	Offset     int
	Limit      int
	NextCursor string
	HasMore    bool
}

func NormalizeOffsetPage(req OffsetPageRequest) (offset int, limit int, err error) {
	limit = req.PageSize
	if limit <= 0 {
		limit = DefaultPageSize
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}

	cursor := strings.TrimSpace(req.Cursor)
	if cursor == "" {
		return 0, limit, nil
	}

	offset, err = strconv.Atoi(cursor)
	if err != nil || offset < 0 {
		return 0, 0, strconv.ErrSyntax
	}
	return offset, limit, nil
}

func BuildOffsetPageResult(offset int, limit int, returned int) OffsetPageResult {
	hasMore := returned > limit
	if hasMore {
		returned = limit
	}

	nextCursor := ""
	if hasMore {
		nextCursor = strconv.Itoa(offset + limit)
	}

	return OffsetPageResult{
		Offset:     offset,
		Limit:      limit,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}
}

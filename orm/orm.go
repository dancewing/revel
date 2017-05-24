package orm

import (
	"errors"
	"time"
)

var (
	DefaultTimeLoc   = time.Local
	DefaultRelsDepth = 2
	ErrNotImplement  = errors.New("have not implement")
)

const (
	formatTime     = "15:04:05"
	formatDate     = "2006-01-02"
	formatDateTime = "2006-01-02 15:04:05"
)

var (
	operators = map[string]bool{
		"exact":     true,
		"iexact":    true,
		"contains":  true,
		"icontains": true,
		// "regex":       true,
		// "iregex":      true,
		"gt":          true,
		"gte":         true,
		"lt":          true,
		"lte":         true,
		"eq":          true,
		"nq":          true,
		"ne":          true,
		"startswith":  true,
		"endswith":    true,
		"istartswith": true,
		"iendswith":   true,
		"in":          true,
		"between":     true,
		// "year":        true,
		// "month":       true,
		// "day":         true,
		// "week_day":    true,
		"isnull": true,
		// "search":      true,
	}
)

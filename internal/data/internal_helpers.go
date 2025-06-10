package data

import (
	"context"
	"database/sql"
	"time"
)

func contextGenerator(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

func convertValueToNullInt32(value int32) sql.NullInt32 {
	if value <= 0 {
		return sql.NullInt32{Valid: false}
	}
	return sql.NullInt32{Int32: value, Valid: true}
}

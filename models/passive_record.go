package models

import (
	"context"
	"database/sql"
	"fmt"
)

type QueryExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type (
	PassiveRecordScanFunc func(cols []interface{}) error
)

var (
	ErrRecordNotPersisted = fmt.Errorf("record has not been saved to the database")
	ErrRecordDeleted      = fmt.Errorf("record has been deleted")
)

type PassiveRecord interface {
	Load(PassiveRecordScanFunc)
	Reload(context.Context, QueryExecutor) error
	Save(context.Context, QueryExecutor) error
	Delete(context.Context, QueryExecutor) error
}

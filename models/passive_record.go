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
	ErrRecordDeleted      = sql.ErrNoRows
)

type PassiveRecord interface {
	Load(PassiveRecordScanFunc) error
	Reload(context.Context, QueryExecutor) error
	Save(context.Context, QueryExecutor) error
	Delete(context.Context, QueryExecutor) error
}

func LoadRecord[R PassiveRecord](row *sql.Row) (R, error) {
	var r R

	if err := r.Load(func(cols []interface{}) error {
		return row.Scan(cols...)
	}); err != nil {
		return r, err
	}

	return r, nil
}

func LoadRecords[R PassiveRecord](rows *sql.Rows) ([]R, error) {
	var dest []R

	for rows.Next() {
		var t R

		err := t.Load(func(cols []interface{}) error {
			return rows.Scan(cols...)
		})
		if err != nil {
			return nil, err
		}

		dest = append(dest, t)
	}

	return dest, nil
}

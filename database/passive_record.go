package database

import (
	"context"
	"database/sql"
	"fmt"
)

type (
	PassiveRecordScanFunc     func(cols []interface{}) error
	PassiveRecordExecFunc     func(query string, args ...any) (sql.Result, error)
	PassiveRecordQueryFunc    func(query string, args ...any) (*sql.Rows, error)
	PassiveRecordQueryRowFunc func(query string, args ...any) *sql.Row
)

var ErrRecordNotPersisted = fmt.Errorf("record has not been saved to the database")

type PassiveRecord interface {
	Load(Scan PassiveRecordScanFunc)
	Reload(QueryRow PassiveRecordQueryRowFunc) error
	Save(Exec PassiveRecordExecFunc) error
}

func ExecWithDatabase(ctx context.Context, db *sql.DB) PassiveRecordExecFunc {
	return func(query string, args ...any) (sql.Result, error) {
		return db.ExecContext(ctx, query, args...)
	}
}

func QueryRowWithDatabase(ctx context.Context, db *sql.DB) PassiveRecordQueryRowFunc {
	return func(query string, args ...any) *sql.Row {
		return db.QueryRowContext(ctx, query, args...)
	}
}

func QueryWithDatabase(ctx context.Context, db *sql.DB) PassiveRecordQueryFunc {
	return func(query string, args ...any) (*sql.Rows, error) {
		return db.QueryContext(ctx, query, args...)
	}
}

func ExecInTransaction(ctx context.Context, tx *sql.Tx) PassiveRecordExecFunc {
	return func(query string, args ...any) (sql.Result, error) {
		return tx.ExecContext(ctx, query, args...)
	}
}

func QueryRowInTransaction(ctx context.Context, tx *sql.Tx) PassiveRecordQueryRowFunc {
	return func(query string, args ...any) *sql.Row {
		return tx.QueryRowContext(ctx, query, args...)
	}
}

func QueryInTransaction(ctx context.Context, tx *sql.Tx) PassiveRecordQueryFunc {
	return func(query string, args ...any) (*sql.Rows, error) {
		return tx.QueryContext(ctx, query, args...)
	}
}

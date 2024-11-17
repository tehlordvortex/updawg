package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrations embed.FS

func migrate(ctx context.Context, db *sql.DB, id int64, migration, query string) {
	fail := func(err error) {
	}

	logger.Printf("migrating id=%d migration=%s", id, migration)
	logger.Print(query)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		fail(err)
	}
	defer tx.Rollback()

	var existingMigrationId int64
	err = tx.QueryRowContext(ctx, "SELECT id FROM migrations WHERE id = ?", id).Scan(&existingMigrationId)
	if err != sql.ErrNoRows {
		if err != nil {
			logger.Fatalf("migrate(%d, %s) failed: %v", id, migration, err)
		}

		if existingMigrationId == id {
			logger.Fatalf("migrate(%d, %s) failed: already ran", id, migration)
		}
	}

	_, err = tx.ExecContext(ctx, query)
	if err != nil {
		logger.Fatalf("migrate(%d, %s) failed: %v", id, migration, err)
	}

	_, err = tx.ExecContext(ctx, "INSERT INTO migrations (id) VALUES (?)", id)
	if err != nil {
		logger.Fatalf("migrate(%d, %s) failed: %v", id, migration, err)
	}

	err = tx.Commit()
	if err != nil {
		logger.Fatalf("migrate(%d, %s) failed: %v", id, migration, err)
	}
}

func getMigration(name string) (id int64, migration string, query string, err error) {
	fail := func(err error) (int64, string, string, error) {
		return 0, "", "", err
	}

	left, migration, found := strings.Cut(name, "_")
	if !found {
		return fail(fmt.Errorf("malformed name: %s", name))
	}

	id, err = strconv.ParseInt(left, 10, 64)
	if err != nil || id < 0 {
		return fail(fmt.Errorf("malformed name: %s: %v", name, err))
	}

	sql, err := migrations.ReadFile("migrations/" + name)
	if err != nil {
		return fail(err)
	}
	query = string(sql)

	return id, migration, query, nil
}

func getLastMigrationId(ctx context.Context, db *sql.DB) (bool, int64, error) {
	var id int64
	err := db.QueryRowContext(ctx, "SELECT id FROM migrations ORDER BY id DESC LIMIT 1").Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return true, 0, nil
		}

		return false, 0, fmt.Errorf("getLastMigrationId: %v", err)
	}

	return false, id, nil
}

package database

import (
	"context"
	"database/sql"
	"log"

	"github.com/tehlordvortex/updawg/config"
	_ "modernc.org/sqlite"
)

var logger *log.Logger

func init() {
	logger = log.New(config.GetLogFile(), "", log.Default().Flags()|log.Lmsgprefix|log.Lshortfile)
}

func Connect(ctx context.Context) *sql.DB {
	uri := config.GetDatabaseUri()

	db, err := sql.Open("sqlite", uri)
	if err != nil {
		logger.Fatalf("sql.Open(%s): %v", uri, err)
	}

	err = db.PingContext(ctx)
	if err != nil {
		logger.Fatalf("db.PingContext(%s): %v", uri, err)
	}

	logger.Printf("connected db=%s", uri)
	setupDatabase(ctx, db)

	return db
}

func setupDatabase(ctx context.Context, db *sql.DB) {
	_, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS migrations (
  id integer NOT NULL PRIMARY KEY
)`)
	if err != nil {
		logger.Fatalf("setupDatabase: %v", err)
	}

	firstRun, lastMigrationId, err := getLastMigrationId(ctx, db)
	if err != nil {
		logger.Fatalf("setupDatabase: %v", err)
	}

	files, err := migrations.ReadDir("migrations")
	if err != nil {
		logger.Fatalf("setupDatabase: %v", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		id, migration, query, err := getMigration(name)
		if err != nil {
			logger.Fatalf("setupDatabase: invalid migration %s: %v", name, err)
		}

		if id > lastMigrationId || firstRun {
			migrate(ctx, db, id, migration, query)

			lastMigrationId = id
			firstRun = false
		}
	}
}

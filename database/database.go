package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"github.com/tehlordvortex/updawg/config"
)

var logger = log.New(config.GetLogFile(), "", log.Default().Flags()|log.Lmsgprefix|log.Lshortfile)

type RWDB struct {
	r *sql.DB
	w *sql.DB
}

func (r *RWDB) Read() *sql.DB  { return r.r }
func (r *RWDB) Write() *sql.DB { return r.w }

func Connect(ctx context.Context, path string) (*RWDB, error) {
	connect := func(writeable bool) (*sql.DB, error) {
		var uri string

		if writeable {
			uri = fmt.Sprintf("file:%s?mode=rwc&_journal_mode=wal&_txlock=immediate&_busy_timeout=5000", path)
		} else {
			uri = fmt.Sprintf("file:%s?mode=ro&_journal_mode=wal&_txlock=deferred", path)
		}

		db, err := sql.Open("sqlite3", uri)
		if err != nil {
			return nil, fmt.Errorf("sql.Open(%s): %v", uri, err)
		}

		if writeable {
			db.SetMaxOpenConns(1)
			db.SetMaxIdleConns(1)
			db.SetConnMaxLifetime(0)
			db.SetConnMaxIdleTime(0)
		}

		err = db.PingContext(ctx)
		if err != nil {
			return nil, fmt.Errorf("db.PingContext(%s): %v", uri, err)
		}

		logger.Printf("connected path=%s writeable=%t", path, writeable)

		return db, nil
	}

	// Connect to database in R/W mode first to ensure it gets created
	w, err := connect(true)
	if err != nil {
		return nil, err
	}

	r, err := connect(false)
	if err != nil {
		return nil, err
	}

	return &RWDB{r, w}, nil
}

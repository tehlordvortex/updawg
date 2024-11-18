package workers

import (
	"context"
	"database/sql"
	"log"

	"github.com/tehlordvortex/updawg/config"
)

var logger = log.New(config.GetLogFile(), "", log.Default().Flags()|log.Lmsgprefix|log.Llongfile)

func Run(ctx context.Context, db *sql.DB) {
	go runTargetsWorker(ctx, db)
}

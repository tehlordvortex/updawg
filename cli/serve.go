package cli

import (
	"context"
	"database/sql"

	"github.com/tehlordvortex/updawg/workers"
)

func runServeCommand(ctx context.Context, db *sql.DB, args []string) {
	go workers.Run(ctx, db)

	<-ctx.Done()
}

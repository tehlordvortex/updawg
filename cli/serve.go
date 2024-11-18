package cli

import (
	"context"

	"github.com/tehlordvortex/updawg/database"
	"github.com/tehlordvortex/updawg/workers"
)

func runServeCommand(ctx context.Context, args []string) {
	db := database.Connect(ctx)

	go workers.Run(ctx, db)

	<-ctx.Done()
}

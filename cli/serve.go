package cli

import (
	"context"

	"github.com/tehlordvortex/updawg/database"
	"github.com/tehlordvortex/updawg/pubsub"
	"github.com/tehlordvortex/updawg/workers"
)

func runServeCommand(ctx context.Context, rwdb *database.RWDB, ps *pubsub.PubSub, args []string) {
	go workers.Run(ctx, rwdb, ps)

	<-ctx.Done()
}

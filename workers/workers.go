package workers

import (
	"context"
	"log"

	"github.com/tehlordvortex/updawg/config"
	"github.com/tehlordvortex/updawg/database"
	"github.com/tehlordvortex/updawg/pubsub"
)

var logger = log.New(config.GetLogFile(), "", log.Default().Flags()|log.Lmsgprefix|log.Lshortfile)

func Run(ctx context.Context, rwdb *database.RWDB, ps *pubsub.PubSub) {
	go runTargetsWorker(ctx, rwdb, ps)
}

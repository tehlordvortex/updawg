package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/tehlordvortex/updawg/cli"
	"github.com/tehlordvortex/updawg/config"
	_ "github.com/tehlordvortex/updawg/config"
	"github.com/tehlordvortex/updawg/database"
	"github.com/tehlordvortex/updawg/pubsub"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	rwdb, err := database.Connect(ctx, config.GetDatabasePath())
	if err != nil {
		log.Panic(err)
	}

	pubsub, err := pubsub.Connect(ctx, config.GetPubsubDatabasePath())
	if err != nil {
		log.Panic(err)
	}
	defer pubsub.Wait()

	cli.Run(ctx, rwdb, pubsub)
}

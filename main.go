package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/tehlordvortex/updawg/cli"
	_ "github.com/tehlordvortex/updawg/config"
	"github.com/tehlordvortex/updawg/database"
	"github.com/tehlordvortex/updawg/pubsub"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := database.Connect(ctx)
	pubsub.Run(ctx)

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt)

		select {
		case <-sigChan:
			cancel()
		case <-ctx.Done():
		}
	}()

	cli.Run(ctx, db)
}

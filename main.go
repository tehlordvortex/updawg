package main

import (
	"context"
	"os"
	"os/signal"

	_ "github.com/tehlordvortex/updawg/config"
	_ "modernc.org/sqlite"

	"github.com/tehlordvortex/updawg/database"
	_ "github.com/tehlordvortex/updawg/database"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	db := database.Connect(ctx)
	defer db.Close()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	// TODO: A lot

	<-sigChan

	cancel()
}

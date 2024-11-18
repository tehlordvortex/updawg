package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/tehlordvortex/updawg/cli"
	_ "github.com/tehlordvortex/updawg/config"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt)

		select {
		case <-sigChan:
			cancel()
		case <-ctx.Done():
		}
	}()

	cli.Run(ctx)
}

package cli

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/tehlordvortex/updawg/config"
	"github.com/tehlordvortex/updawg/database"
	"github.com/tehlordvortex/updawg/pubsub"
)

const (
	Header = "updawg - Simple uptime monitoring\n\n"
)

var logger = log.New(config.GetLogFile(), "", log.Default().Flags()|log.Lmsgprefix|log.Llongfile)

func Run(ctx context.Context, rwdb *database.RWDB, ps *pubsub.PubSub) {
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		logger.Println("no command provided")
		printUsage()
		os.Exit(1)
	}

	command := args[0]
	subArgs := args[1:]

	switch command {
	case "targets":
		runTargetsCommand(ctx, rwdb, ps, subArgs)
	case "serve":
		runServeCommand(ctx, rwdb, ps, subArgs)
	default:
		logger.Println("unknown command:", command)
		printUsage()
		os.Exit(1)
	}

	// TODO: A lot
}

func printUsage() {
	fmt.Fprintf(flag.CommandLine.Output(), Header)
	fmt.Fprintf(flag.CommandLine.Output(), "targets\t\tManage monitoring targets\n")
	flag.PrintDefaults()
}

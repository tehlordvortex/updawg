package cli

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"

	"github.com/tehlordvortex/updawg/config"
	"github.com/tehlordvortex/updawg/models"
)

func runTargetsCommand(ctx context.Context, db *sql.DB, args []string) {
	fs := flag.NewFlagSet("targets", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		logger.Fatalln(err)
	}

	args = fs.Args()
	if len(args) == 0 {
		logger.Println("no command provided")
		printTargetsUsage(fs)
		os.Exit(1)
	}

	command := args[0]
	subArgs := args[1:]

	switch command {
	case "create":
		runCreateCommand(ctx, db, subArgs)
	case "modify":
		runModifyCommand(ctx, db, subArgs)
	case "list":
		runListCommand(ctx, db, subArgs)
	case "delete":
		runDeleteCommand(ctx, db, subArgs)
	default:
		logger.Println("unknown command:", command)
		printTargetsUsage(fs)
		os.Exit(1)
	}
}

func printTargetsUsage(fs *flag.FlagSet) {
	fmt.Fprintf(flag.CommandLine.Output(), Header)
	fmt.Fprintln(flag.CommandLine.Output(), "targets - Manage monitoring targets")
	fmt.Fprintln(flag.CommandLine.Output(), "create\t\tCreate new target")
	fmt.Fprintln(flag.CommandLine.Output(), "modify\t\tModify a target")
	fmt.Fprintln(flag.CommandLine.Output(), "list\t\tList existing targets")
	fmt.Fprintln(flag.CommandLine.Output(), "delete\t\tDelete a target")
	fs.PrintDefaults()
	flag.PrintDefaults()
}

func runCreateCommand(ctx context.Context, db *sql.DB, args []string) {
	fs := flag.NewFlagSet("targets create", flag.ExitOnError)
	name := fs.String("name", "", "An optional name for the target")
	uri := fs.String("uri", "", "The URI to make requests to")
	method := fs.String("method", config.DefaultMethod, "The HTTP method to use")
	period := fs.Uint("period", config.DefaultPeriod, "The interval (in seconds) in which requests are made")

	if err := fs.Parse(args); err != nil {
		logger.Fatalln(err)
	}

	if *uri == "" {
		fs.Usage()
		os.Exit(1)
	}

	target := models.Target{
		Name:   *name,
		Uri:    *uri,
		Method: *method,
		Period: int64(*period),
	}

	if err := target.Save(ctx, db); err != nil {
		logger.Fatalln(err)
	}

	logger.Println("target created id=" + target.Id())
}

func runModifyCommand(ctx context.Context, db *sql.DB, args []string) {
	fs := flag.NewFlagSet("targets modify", flag.ExitOnError)
	id := fs.String("id", "", "The ID of the target to modify (can be partial)")
	name := fs.String("name", "", "An optional name for the target")
	uri := fs.String("uri", "", "The URI to make requests to")
	method := fs.String("method", config.DefaultMethod, "The HTTP method to use")
	period := fs.Uint("period", config.DefaultPeriod, "The interval (in seconds) in which requests are made")

	if err := fs.Parse(args); err != nil {
		logger.Fatalln(err)
	}

	if *id == "" {
		fs.Usage()
		os.Exit(1)
	}

	targets, err := models.FindTargetsByIdPrefix(ctx, db, *id)
	if err != nil {
		logger.Fatalln(err)
	}

	if len(targets) == 0 {
		logger.Fatalln("not found:", id)
	} else if len(targets) != 1 {
		logger.Fatalln(id, "is ambiguous")
	}

	target := targets[0]

	if *name != "" {
		target.Name = *name
	}

	if *uri != "" {
		target.Uri = *uri
	}

	if *method != "" {
		target.Method = *method
	}

	if *period != 0 {
		target.Period = int64(*period)
	}

	if err := target.Save(ctx, db); err != nil {
		logger.Fatalln(err)
	}

	logger.Println("target updated:", target)
}

func runListCommand(ctx context.Context, db *sql.DB, args []string) {
	targets, err := models.FindAllTargets(ctx, db)
	if err != nil {
		logger.Fatalln(err)
	}

	for _, target := range targets {
		logger.Println(target)
	}
}

func runDeleteCommand(ctx context.Context, db *sql.DB, args []string) {
	if len(args) == 0 {
		logger.Fatalln("missing target id")
	}

	id := args[0]

	targets, err := models.FindTargetsByIdPrefix(ctx, db, id)
	if err != nil {
		logger.Fatalln(err)
	}

	if len(targets) == 0 {
		logger.Fatalln("not found:", id)
	} else if len(targets) != 1 {
		logger.Fatalln(id, "is ambiguous")
	}

	target := targets[0]
	if err := target.Delete(ctx, db); err != nil {
		logger.Fatalln(err)
	}

	logger.Println("deleted:", target)
}

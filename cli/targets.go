package cli

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/tehlordvortex/updawg/config"
	"github.com/tehlordvortex/updawg/database"
	"github.com/tehlordvortex/updawg/models"
)

func runTargetsCommand(ctx context.Context, args []string) {
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
		runCreateCommand(ctx, subArgs)
	case "list":
		runListCommand(ctx, subArgs)
	case "delete":
		runDeleteCommand(ctx, subArgs)
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
	fmt.Fprintln(flag.CommandLine.Output(), "list\t\tList existing targets")
	fmt.Fprintln(flag.CommandLine.Output(), "delete\t\tDelete a target")
	fs.PrintDefaults()
	flag.PrintDefaults()
}

func runCreateCommand(ctx context.Context, args []string) {
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

	db := database.Connect(ctx)

	target := models.Target{
		Name:   *name,
		Uri:    *uri,
		Method: *method,
		Period: int64(*period),
	}

	if err := target.Save(models.ExecWithDatabase(ctx, db)); err != nil {
		logger.Fatalln(err)
	}

	logger.Println("target created id=" + target.Id())
}

func runListCommand(ctx context.Context, args []string) {
	db := database.Connect(ctx)

	rows, err := db.QueryContext(ctx, "SELECT * FROM targets")
	if err != nil {
		logger.Fatalln(err)
	}

	for rows.Next() {
		var target models.Target
		if err := target.Load(func(cols []interface{}) error {
			return rows.Scan(cols...)
		}); err != nil {
			logger.Fatalln(err)
		}

		fmt.Println(target)
	}
}

func runDeleteCommand(ctx context.Context, args []string) {
	db := database.Connect(ctx)

	if len(args) == 0 {
		logger.Fatalln("missing target id")
	}

	id := args[0]
	rows, err := db.QueryContext(ctx, "SELECT * FROM targets WHERE id LIKE ?", id+"%")
	if err != nil {
		logger.Fatalln(err)
	}

	var target models.Target
	if !rows.Next() {
		logger.Fatalln("not found:", id)
	}

	err = target.Load(func(cols []interface{}) error {
		return rows.Scan(cols...)
	})
	if err != nil {
		logger.Fatalln(err)
	}

	if rows.Next() {
		logger.Fatalln(id, "is ambiguous")
	}

	if err := target.Delete(models.ExecWithDatabase(ctx, db)); err != nil {
		logger.Fatalln(err)
	}

	fmt.Println("deleted:", target)
}

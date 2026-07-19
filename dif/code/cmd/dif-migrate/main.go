package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/aaraminds/dif/libs/migrations"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "dif-migrate: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		databaseURL   string
		migrationsDir string
		psqlPath      string
		timeout       time.Duration
	)
	flag.StringVar(&databaseURL, "database-url", os.Getenv("DIF_DATABASE_URL"), "PostgreSQL database URL; defaults to DIF_DATABASE_URL")
	flag.StringVar(&migrationsDir, "migrations-dir", migrations.DefaultDir, "directory containing ordered DIF SQL migrations")
	flag.StringVar(&psqlPath, "psql", migrations.DefaultPSQLPath, "path to psql executable")
	flag.DurationVar(&timeout, "timeout", 30*time.Second, "migration command timeout")
	flag.Parse()

	if flag.NArg() != 1 {
		return fmt.Errorf("usage: dif-migrate [flags] apply|check")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	runner := migrations.PSQLRunner{
		DatabaseURL:   databaseURL,
		MigrationsDir: migrationsDir,
		PSQLPath:      psqlPath,
	}

	switch flag.Arg(0) {
	case "apply":
		return runner.Apply(ctx)
	case "check":
		return runner.CheckInventory(ctx)
	default:
		return fmt.Errorf("unknown command %q; expected apply or check", flag.Arg(0))
	}
}

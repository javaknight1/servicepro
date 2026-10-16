// Command migrate applies, rolls back, inspects or repairs the embedded
// golang-migrate migrations (backend/migrations) against DATABASE_URL.
//
// It is the migration entrypoint for local development and for the release
// pipeline, which runs `migrate up` BEFORE the new Cloud Run revision gets
// traffic — code must never land ahead of its schema. It reuses
// pkg/database so the embedded migration FS and URL handling are identical to
// what the server does when RUN_MIGRATIONS_ON_STARTUP=true.
//
// Usage (from backend/):
//
//	go run ./cmd/migrate up         # apply all pending migrations
//	go run ./cmd/migrate down       # roll everything back (refused when ENV=production)
//	go run ./cmd/migrate version    # print applied version + dirty flag
//	go run ./cmd/migrate force N    # set version N and clear dirty flag (repair)
//
// DATABASE_URL is read from the environment (backend/.env is loaded when
// present). It defaults to the local docker-compose database, matching
// config.Load().
package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/database"
)

func main() {
	// Load backend/.env for local runs; absent in CI/production.
	_ = godotenv.Load()

	if len(os.Args) < 2 {
		usage("missing subcommand")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = config.DefaultDatabaseURL
	}

	switch cmd := os.Args[1]; cmd {
	case "up":
		if err := database.RunMigrations(databaseURL); err != nil {
			fail("migrate up failed: %v", err)
		}
		printVersion(databaseURL)

	case "down":
		// Rolling every migration back drops every table. Production
		// rollback is a forward-fix migration, so require an explicit opt-in.
		if isProduction() && os.Getenv("ALLOW_PRODUCTION_DOWN") != "true" {
			fail("refusing to run `down` with ENV=production; production rollback is a forward-fix migration. " +
				"Set ALLOW_PRODUCTION_DOWN=true only if you are certain.")
		}
		if err := database.MigrateDown(databaseURL); err != nil {
			fail("migrate down failed: %v", err)
		}
		fmt.Println("all migrations rolled back")

	case "version":
		printVersion(databaseURL)

	case "force":
		if len(os.Args) < 3 {
			usage("force requires a version number")
		}
		n, err := strconv.Atoi(os.Args[2])
		if err != nil || n < -1 {
			usage(fmt.Sprintf("invalid version %q", os.Args[2]))
		}
		if err := database.ForceMigrationVersion(databaseURL, n); err != nil {
			fail("migrate force failed: %v", err)
		}
		fmt.Printf("forced version=%d (dirty flag cleared)\n", n)

	default:
		usage(fmt.Sprintf("unknown subcommand %q", cmd))
	}
}

// printVersion prints the applied version and exits non-zero when the schema
// is dirty, so a pipeline pre-flight check fails loudly.
func printVersion(databaseURL string) {
	version, dirty, err := database.MigrationVersion(databaseURL)
	if err != nil {
		fail("migrate version failed: %v", err)
	}
	latest, _ := database.LatestMigrationVersion()
	fmt.Printf("version=%d dirty=%t latest_embedded=%d\n", version, dirty, latest)
	if dirty {
		fail("schema is dirty: a prior migration failed partway; repair it by hand, then run `migrate force N`")
	}
}

func isProduction() bool {
	return os.Getenv("ENV") == "production" || os.Getenv("ENVIRONMENT") == "production"
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

func usage(problem string) {
	fmt.Fprintf(os.Stderr, `error: %s

Usage:
  migrate up         apply all pending migrations
  migrate down       roll everything back (refused when ENV=production unless ALLOW_PRODUCTION_DOWN=true)
  migrate version    print applied version + dirty flag
  migrate force N    set the recorded version to N and clear the dirty flag (no SQL is run)
`, problem)
	os.Exit(2)
}

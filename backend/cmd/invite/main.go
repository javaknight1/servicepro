// Command invite manages platform invite codes for REGISTRATION_MODE=invite_only.
//
// Usage (from backend/):
//
//	go run ./cmd/invite create [-max-uses 1] [-expires 168h] [-note "for Jane"]
//	go run ./cmd/invite list
//	go run ./cmd/invite revoke <id>
//
// The plaintext code is printed once by `create`; only its SHA-256 hash is
// stored. DATABASE_URL is read from the environment (backend/.env is loaded
// when present), defaulting to the local docker-compose database. The same
// operations are available to platform super admins at /api/v1/admin/invite-codes.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

func main() {
	_ = godotenv.Load()
	if len(os.Args) < 2 {
		usage("missing subcommand")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = config.DefaultDatabaseURL
	}
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: databaseURL, PreferSimpleProtocol: true}),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		fail("connect: %v", err)
	}
	svc := services.NewInviteCodeService(db)
	ctx := context.Background()

	switch os.Args[1] {
	case "create":
		fs := flag.NewFlagSet("create", flag.ExitOnError)
		maxUses := fs.Int("max-uses", 1, "number of registrations allowed (0 = unlimited)")
		expires := fs.Duration("expires", 168*time.Hour, "validity from now (0 = never)")
		note := fs.String("note", "", "who/what the code is for")
		_ = fs.Parse(os.Args[2:])
		var notePtr *string
		if *note != "" {
			notePtr = note
		}
		code, ic, err := svc.Create(ctx, *maxUses, *expires, nil, notePtr)
		if err != nil {
			fail("%v", err)
		}
		fmt.Printf("invite code: %s\nid:          %s\nmax uses:    %d\n", code, ic.ID, ic.MaxUses)
		if ic.ExpiresAt != nil {
			fmt.Printf("expires:     %s\n", ic.ExpiresAt.Format(time.RFC3339))
		}

	case "list":
		codes, err := svc.List(ctx, 200)
		if err != nil {
			fail("%v", err)
		}
		for _, c := range codes {
			state := "active"
			switch {
			case c.RevokedAt != nil:
				state = "revoked"
			case c.ExpiresAt != nil && c.ExpiresAt.Before(time.Now()):
				state = "expired"
			case c.MaxUses > 0 && c.Uses >= c.MaxUses:
				state = "used up"
			}
			note := ""
			if c.Note != nil {
				note = *c.Note
			}
			fmt.Printf("%s  %-8s uses=%d/%d  created=%s  %s\n", c.ID, state, c.Uses, c.MaxUses, c.CreatedAt.Format("2006-01-02"), note)
		}

	case "revoke":
		if len(os.Args) < 3 {
			usage("revoke requires an id")
		}
		id, err := uuid.Parse(os.Args[2])
		if err != nil {
			usage("invalid id")
		}
		if err := svc.Revoke(ctx, id); err != nil {
			fail("%v", err)
		}
		fmt.Println("revoked", id)

	default:
		usage(fmt.Sprintf("unknown subcommand %q", os.Args[1]))
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

func usage(problem string) {
	fmt.Fprintf(os.Stderr, `error: %s

Usage:
  invite create [-max-uses N] [-expires 168h] [-note TEXT]
  invite list
  invite revoke ID
`, problem)
	os.Exit(2)
}

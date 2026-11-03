// Command stripe-catalog creates ServicePro's plan catalog in Stripe (D15,
// T210): products sp_solo / sp_team / sp_pro, monthly and annual prices with
// lookup keys sp_<tier>_<monthly|annual>, and the founding coupon
// sp_founding_team ($40 off Team, forever, max 25 redemptions). It is
// idempotent: existing objects are left alone and reported; a price whose
// amount differs from the catalog is reported as a mismatch, never changed.
//
// Usage (from backend/, or `make stripe-catalog`):
//
//	go run ./cmd/stripe-catalog [-env-file .env] [-live]
//
// It runs in the mode of STRIPE_SECRET_KEY. A live key is refused unless
// -live is passed, so the catalog lands in test mode first.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/stripe/stripe-go/v87"

	stripeClient "github.com/javaknight1/servicepro/backend/internal/services/stripe"
)

func main() {
	envFile := flag.String("env-file", ".env", "dotenv file to load when present (existing env vars win)")
	live := flag.Bool("live", false, "allow a live-mode key (sk_live_/rk_live_)")
	flag.Parse()

	if *envFile != "" {
		if _, err := os.Stat(*envFile); err == nil {
			if err := godotenv.Load(*envFile); err != nil {
				fail("load %s: %v", *envFile, err)
			}
		}
	}
	key := os.Getenv("STRIPE_SECRET_KEY")
	mode, err := keyMode(key, *live)
	if err != nil {
		fail("%v", err)
	}
	fmt.Printf("Stripe catalog sync (%s mode)\n", mode)
	// "No such product" 404s are how missing objects are found; real
	// failures come back as errors below.
	stripe.DefaultLeveledLogger = &stripe.LeveledLogger{Level: stripe.LevelNull}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	results, err := stripeClient.EnsureCatalog(ctx, stripeClient.NewBillingClient(key))
	for _, r := range results {
		line := fmt.Sprintf("  %-8s %-18s %-9s %s", r.Kind, r.Key, r.Action, r.ID)
		if r.Note != "" {
			line += "  (" + r.Note + ")"
		}
		fmt.Println(line)
	}
	if err != nil {
		fail("%v", err)
	}
	for _, r := range results {
		if r.Action == "mismatch" {
			fail("catalog mismatch: fix it in the Stripe Dashboard (or move the lookup key) and run again")
		}
	}
	fmt.Println("Done.")
}

// keyMode reports the key's mode, refusing live keys unless allowLive.
func keyMode(key string, allowLive bool) (string, error) {
	switch {
	case key == "":
		return "", fmt.Errorf("STRIPE_SECRET_KEY is not set")
	case strings.HasPrefix(key, "sk_test_"), strings.HasPrefix(key, "rk_test_"):
		return "test", nil
	case strings.HasPrefix(key, "sk_live_"), strings.HasPrefix(key, "rk_live_"):
		if !allowLive {
			return "", fmt.Errorf("refusing a live-mode key without -live (create the catalog in test mode first)")
		}
		return "live", nil
	default:
		return "", fmt.Errorf("STRIPE_SECRET_KEY is not a Stripe secret key")
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "stripe-catalog: "+format+"\n", args...)
	os.Exit(1)
}

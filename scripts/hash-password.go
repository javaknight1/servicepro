//go:build ignore

// hash-password.go generates a bcrypt hash for a given password.
// Usage: cd backend && go run ../scripts/hash-password.go [password]
// If no password provided, defaults to "password123"
package main

import (
	"fmt"
	"os"

	"github.com/javaknight1/servicepro/backend/pkg/auth"
)

const defaultPassword = "password123"

func main() {
	password := defaultPassword
	if len(os.Args) > 1 {
		password = os.Args[1]
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating hash: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Password: %s\n", password)
	fmt.Printf("Hash:     %s\n", hash)
	fmt.Println()
	fmt.Println("To use in 002_seed_dev.sql, update the password_hash value.")
}

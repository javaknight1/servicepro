package repository

import "os"

// testDatabaseURL is the Postgres the repository tests use: TEST_DATABASE_URL
// (set by CI, which migrates it first), else the local docker-compose test db.
func testDatabaseURL() string {
	if u := os.Getenv("TEST_DATABASE_URL"); u != "" {
		return u
	}
	return "postgres://postgres:postgres@localhost:5432/servicepro_test?sslmode=disable"
}

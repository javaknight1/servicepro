//go:build integration

package integration

import (
	"os"
	"testing"

	"github.com/javaknight1/servicepro/backend/tests/integration/fixtures"
	"github.com/javaknight1/servicepro/backend/tests/integration/helpers"
	"github.com/javaknight1/servicepro/backend/tests/integration/mocks"
)

// =============================================================================
// Test Suite Setup
// =============================================================================

// TestSuite holds shared test resources
type TestSuite struct {
	Config   *TestConfig
	DB       *helpers.TestDB
	Auth     *helpers.AuthHelper
	Fixtures *fixtures.FixtureManager
	HTTP     *helpers.TestClient
	Stripe   *mocks.StripeMock
	AWS      *mocks.AWSMock
	MailHog  *mocks.MailHogClient
}

var testSuite *TestSuite

// TestMain sets up and tears down the test environment
func TestMain(m *testing.M) {
	// Load configuration
	config := LoadTestConfig()
	SetTestEnv()

	// Exit code
	code := m.Run()

	// Cleanup if needed
	if testSuite != nil {
		if testSuite.DB != nil {
			testSuite.DB.Close()
		}
		if testSuite.Stripe != nil {
			testSuite.Stripe.Close()
		}
	}

	os.Exit(code)
}

// SetupTestSuite initializes the test suite
func SetupTestSuite(t *testing.T) *TestSuite {
	t.Helper()

	if testSuite != nil {
		return testSuite
	}

	config := GetTestConfig()

	// Initialize database
	db := helpers.NewTestDB(t, config.DatabaseURL)

	// Run migrations
	migrationsPath := "../../migrations"
	db.RunMigrations(t, migrationsPath)

	// Initialize helpers
	authHelper := helpers.NewAuthHelper(db.DB, config.JWTSecret)
	fixtureManager := fixtures.NewFixtureManager(db.DB)

	// Initialize HTTP client
	httpClient := helpers.NewTestClient(config.APIBaseURL)

	// Initialize mocks
	stripeMock := mocks.NewStripeMock(t)
	awsMock := mocks.NewAWSMock(
		config.AWSEndpointURL,
		config.AWSRegion,
		config.AWSAccessKeyID,
		config.AWSSecretAccessKey,
	)
	mailHog := mocks.NewMailHogClient(config.MailHogAPI)

	testSuite = &TestSuite{
		Config:   config,
		DB:       db,
		Auth:     authHelper,
		Fixtures: fixtureManager,
		HTTP:     httpClient,
		Stripe:   stripeMock,
		AWS:      awsMock,
		MailHog:  mailHog,
	}

	return testSuite
}

// SetupTest prepares a clean state for each test
func SetupTest(t *testing.T) *TestSuite {
	t.Helper()

	suite := SetupTestSuite(t)

	// Clean database before each test
	suite.DB.CleanAllTables(t)

	// Clear email inbox
	suite.MailHog.DeleteAllMessages(t)

	// Reset AWS mock state
	suite.AWS.Reset()

	return suite
}

// =============================================================================
// Test Helpers
// =============================================================================

// CreateAuthenticatedClient creates an HTTP client with an authenticated user
func (s *TestSuite) CreateAuthenticatedClient(t *testing.T, role string) (*helpers.TestClient, *helpers.TestUser) {
	t.Helper()

	var user *helpers.TestUser
	switch role {
	case "admin":
		user = s.Auth.CreateAdminUser(t)
	case "manager":
		user = s.Auth.CreateManagerUser(t)
	case "technician":
		user = s.Auth.CreateTechnicianUser(t)
	default:
		user = s.Auth.CreateRegularUser(t)
	}

	client := helpers.NewTestClient(s.Config.APIBaseURL)
	token := s.Auth.GenerateToken(t, user)
	client.SetAuthToken(token)

	return client, user
}

// CreateTestCustomer creates a test customer with the fixture manager
func (s *TestSuite) CreateTestCustomer(t *testing.T) *fixtures.Customer {
	t.Helper()
	return s.Fixtures.CreateCustomer(t, &fixtures.Customer{})
}

// CreateTestJob creates a test job with the fixture manager
func (s *TestSuite) CreateTestJob(t *testing.T, customerID string) *fixtures.Job {
	t.Helper()
	return s.Fixtures.CreateJob(t, &fixtures.Job{CustomerID: customerID})
}

// CreateTestInvoice creates a test invoice with line items
func (s *TestSuite) CreateTestInvoice(t *testing.T, customerID string) *fixtures.Invoice {
	t.Helper()
	return s.Fixtures.CreateInvoiceWithLineItems(t, customerID)
}

//go:build integration

package api

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/tests/integration"
	"github.com/javaknight1/servicepro/backend/tests/integration/fixtures"
	"github.com/javaknight1/servicepro/backend/tests/integration/helpers"
)

// =============================================================================
// Customer API Tests
// =============================================================================

func TestCustomerAPI(t *testing.T) {
	suite := integration.SetupTest(t)
	client, _ := suite.CreateAuthenticatedClient(t, "admin")

	t.Run("Create Customer", func(t *testing.T) {
		t.Run("successful creation", func(t *testing.T) {
			resp := client.Post(t, "/api/v1/customers", map[string]interface{}{
				"email":      "newcustomer@test.com",
				"first_name": "John",
				"last_name":  "Doe",
				"phone":      "555-1234",
				"address":    "123 Main St",
				"city":       "Anytown",
				"state":      "CA",
				"zip_code":   "12345",
			})

			helpers.AssertCreated(t, resp)

			var result map[string]interface{}
			require.NoError(t, resp.JSON(&result))

			assert.NotEmpty(t, result["id"])
			assert.Equal(t, "newcustomer@test.com", result["email"])
			assert.Equal(t, "John", result["first_name"])
			assert.Equal(t, "Doe", result["last_name"])
		})

		t.Run("creation with missing required fields fails", func(t *testing.T) {
			resp := client.Post(t, "/api/v1/customers", map[string]interface{}{
				"email": "incomplete@test.com",
			})

			helpers.AssertBadRequest(t, resp)
		})

		t.Run("creation with invalid email fails", func(t *testing.T) {
			resp := client.Post(t, "/api/v1/customers", map[string]interface{}{
				"email":      "notanemail",
				"first_name": "Test",
				"last_name":  "User",
			})

			helpers.AssertBadRequest(t, resp)
		})

		t.Run("creation with duplicate email fails", func(t *testing.T) {
			// Create existing customer
			suite.Fixtures.CreateCustomer(t, &fixtures.Customer{
				Email: "duplicate@test.com",
			})

			resp := client.Post(t, "/api/v1/customers", map[string]interface{}{
				"email":      "duplicate@test.com",
				"first_name": "Test",
				"last_name":  "User",
			})

			helpers.AssertConflict(t, resp)
		})
	})

	t.Run("Get Customer", func(t *testing.T) {
		customer := suite.CreateTestCustomer(t)

		t.Run("successful retrieval", func(t *testing.T) {
			resp := client.Get(t, "/api/v1/customers/"+customer.ID)

			helpers.AssertOK(t, resp)

			var result map[string]interface{}
			require.NoError(t, resp.JSON(&result))

			assert.Equal(t, customer.ID, result["id"])
			assert.Equal(t, customer.Email, result["email"])
		})

		t.Run("retrieval of non-existent customer fails", func(t *testing.T) {
			resp := client.Get(t, "/api/v1/customers/non-existent-id")

			helpers.AssertNotFound(t, resp)
		})
	})

	t.Run("Update Customer", func(t *testing.T) {
		customer := suite.CreateTestCustomer(t)

		t.Run("successful update", func(t *testing.T) {
			resp := client.Put(t, "/api/v1/customers/"+customer.ID, map[string]interface{}{
				"first_name": "Updated",
				"last_name":  "Name",
			})

			helpers.AssertOK(t, resp)

			var result map[string]interface{}
			require.NoError(t, resp.JSON(&result))

			assert.Equal(t, "Updated", result["first_name"])
			assert.Equal(t, "Name", result["last_name"])
		})

		t.Run("partial update", func(t *testing.T) {
			resp := client.Patch(t, "/api/v1/customers/"+customer.ID, map[string]interface{}{
				"phone": "555-9999",
			})

			helpers.AssertOK(t, resp)

			var result map[string]interface{}
			require.NoError(t, resp.JSON(&result))

			assert.Equal(t, "555-9999", result["phone"])
		})

		t.Run("update non-existent customer fails", func(t *testing.T) {
			resp := client.Put(t, "/api/v1/customers/non-existent-id", map[string]interface{}{
				"first_name": "Test",
			})

			helpers.AssertNotFound(t, resp)
		})
	})

	t.Run("Delete Customer", func(t *testing.T) {
		t.Run("successful deletion", func(t *testing.T) {
			customer := suite.CreateTestCustomer(t)

			resp := client.Delete(t, "/api/v1/customers/"+customer.ID)

			helpers.AssertNoContent(t, resp)

			// Verify deletion
			getResp := client.Get(t, "/api/v1/customers/"+customer.ID)
			helpers.AssertNotFound(t, getResp)
		})

		t.Run("delete non-existent customer fails", func(t *testing.T) {
			resp := client.Delete(t, "/api/v1/customers/non-existent-id")

			helpers.AssertNotFound(t, resp)
		})
	})

	t.Run("List Customers", func(t *testing.T) {
		// Create multiple customers
		for i := 0; i < 5; i++ {
			suite.Fixtures.CreateCustomer(t, &fixtures.Customer{
				FirstName: fmt.Sprintf("Customer%d", i),
			})
		}

		t.Run("list all customers", func(t *testing.T) {
			resp := client.Get(t, "/api/v1/customers")

			helpers.AssertOK(t, resp)

			var result map[string]interface{}
			require.NoError(t, resp.JSON(&result))

			data, ok := result["data"].([]interface{})
			require.True(t, ok)
			assert.GreaterOrEqual(t, len(data), 5)
		})

		t.Run("list with pagination", func(t *testing.T) {
			resp := client.NewRequest("GET", "/api/v1/customers").
				WithQuery("page", "1").
				WithQuery("limit", "2").
				Do(t)

			helpers.AssertOK(t, resp)

			var result map[string]interface{}
			require.NoError(t, resp.JSON(&result))

			data, ok := result["data"].([]interface{})
			require.True(t, ok)
			assert.LessOrEqual(t, len(data), 2)
		})

		t.Run("search customers", func(t *testing.T) {
			// Create a customer with unique name
			suite.Fixtures.CreateCustomer(t, &fixtures.Customer{
				FirstName: "UniqueSearchName",
			})

			resp := client.NewRequest("GET", "/api/v1/customers").
				WithQuery("search", "UniqueSearchName").
				Do(t)

			helpers.AssertOK(t, resp)

			var result map[string]interface{}
			require.NoError(t, resp.JSON(&result))

			data, ok := result["data"].([]interface{})
			require.True(t, ok)
			assert.GreaterOrEqual(t, len(data), 1)
		})
	})

	t.Run("Customer Jobs", func(t *testing.T) {
		customer := suite.CreateTestCustomer(t)

		// Create jobs for the customer
		for i := 0; i < 3; i++ {
			suite.Fixtures.CreateJob(t, &fixtures.Job{
				CustomerID: customer.ID,
				Title:      fmt.Sprintf("Job %d", i),
			})
		}

		t.Run("list customer jobs", func(t *testing.T) {
			resp := client.Get(t, "/api/v1/customers/"+customer.ID+"/jobs")

			helpers.AssertOK(t, resp)

			var result map[string]interface{}
			require.NoError(t, resp.JSON(&result))

			data, ok := result["data"].([]interface{})
			require.True(t, ok)
			assert.Equal(t, 3, len(data))
		})
	})

	t.Run("Customer Invoices", func(t *testing.T) {
		customer := suite.CreateTestCustomer(t)

		// Create invoices for the customer
		for i := 0; i < 2; i++ {
			suite.Fixtures.CreateInvoice(t, &fixtures.Invoice{
				CustomerID: customer.ID,
			})
		}

		t.Run("list customer invoices", func(t *testing.T) {
			resp := client.Get(t, "/api/v1/customers/"+customer.ID+"/invoices")

			helpers.AssertOK(t, resp)

			var result map[string]interface{}
			require.NoError(t, resp.JSON(&result))

			data, ok := result["data"].([]interface{})
			require.True(t, ok)
			assert.Equal(t, 2, len(data))
		})
	})
}

// =============================================================================
// Customer API Error Handling Tests
// =============================================================================

func TestCustomerAPIErrorHandling(t *testing.T) {
	suite := integration.SetupTest(t)

	t.Run("Unauthenticated requests fail", func(t *testing.T) {
		client := helpers.NewTestClient(suite.Config.APIBaseURL)

		resp := client.Get(t, "/api/v1/customers")

		helpers.AssertUnauthorized(t, resp)
	})

	t.Run("Invalid JSON body fails", func(t *testing.T) {
		client, _ := suite.CreateAuthenticatedClient(t, "admin")

		// Send request with invalid JSON
		client.SetHeader("Content-Type", "application/json")
		resp := client.NewRequest("POST", "/api/v1/customers").
			WithBody("invalid json{").
			Do(t)

		helpers.AssertBadRequest(t, resp)
	})
}

// =============================================================================
// Customer Validation Tests
// =============================================================================

func TestCustomerValidation(t *testing.T) {
	suite := integration.SetupTest(t)
	client, _ := suite.CreateAuthenticatedClient(t, "admin")

	validationCases := []struct {
		name     string
		data     map[string]interface{}
		expected int
	}{
		{
			name:     "missing email",
			data:     map[string]interface{}{"first_name": "Test", "last_name": "User"},
			expected: 400,
		},
		{
			name:     "invalid email format",
			data:     map[string]interface{}{"email": "invalid", "first_name": "Test", "last_name": "User"},
			expected: 400,
		},
		{
			name:     "missing first_name",
			data:     map[string]interface{}{"email": "test@test.com", "last_name": "User"},
			expected: 400,
		},
		{
			name:     "missing last_name",
			data:     map[string]interface{}{"email": "test@test.com", "first_name": "Test"},
			expected: 400,
		},
		{
			name:     "invalid phone format",
			data:     map[string]interface{}{"email": "test@test.com", "first_name": "Test", "last_name": "User", "phone": "abc"},
			expected: 400,
		},
		{
			name:     "invalid zip code",
			data:     map[string]interface{}{"email": "test@test.com", "first_name": "Test", "last_name": "User", "zip_code": "invalid"},
			expected: 400,
		},
	}

	for _, tc := range validationCases {
		t.Run(tc.name, func(t *testing.T) {
			resp := client.Post(t, "/api/v1/customers", tc.data)
			helpers.AssertStatus(t, resp, tc.expected)
		})
	}
}

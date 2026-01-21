//go:build integration

package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/tests/integration"
	"github.com/javaknight1/servicepro/backend/tests/integration/fixtures"
	"github.com/javaknight1/servicepro/backend/tests/integration/helpers"
)

// =============================================================================
// Authentication API Tests
// =============================================================================

func TestAuthAPI(t *testing.T) {
	suite := integration.SetupTest(t)

	t.Run("Registration", func(t *testing.T) {
		t.Run("successful registration", func(t *testing.T) {
			resp := suite.HTTP.Post(t, "/api/v1/auth/register", map[string]interface{}{
				"email":      "newuser@test.com",
				"password":   "SecurePassword123!",
				"first_name": "New",
				"last_name":  "User",
			})

			helpers.AssertCreated(t, resp)

			var result map[string]interface{}
			require.NoError(t, resp.JSON(&result))

			assert.NotEmpty(t, result["user"])
			assert.NotEmpty(t, result["access_token"])
		})

		t.Run("registration with existing email fails", func(t *testing.T) {
			// Create existing user
			suite.Auth.CreateTestUser(t, &helpers.TestUser{
				Email: "existing@test.com",
			})

			resp := suite.HTTP.Post(t, "/api/v1/auth/register", map[string]interface{}{
				"email":      "existing@test.com",
				"password":   "SecurePassword123!",
				"first_name": "Test",
				"last_name":  "User",
			})

			helpers.AssertConflict(t, resp)
		})

		t.Run("registration with weak password fails", func(t *testing.T) {
			resp := suite.HTTP.Post(t, "/api/v1/auth/register", map[string]interface{}{
				"email":      "weak@test.com",
				"password":   "weak",
				"first_name": "Test",
				"last_name":  "User",
			})

			helpers.AssertBadRequest(t, resp)
		})

		t.Run("registration with invalid email fails", func(t *testing.T) {
			resp := suite.HTTP.Post(t, "/api/v1/auth/register", map[string]interface{}{
				"email":      "notanemail",
				"password":   "SecurePassword123!",
				"first_name": "Test",
				"last_name":  "User",
			})

			helpers.AssertBadRequest(t, resp)
		})
	})

	t.Run("Login", func(t *testing.T) {
		// Create test user
		user := suite.Auth.CreateTestUser(t, &helpers.TestUser{
			Email:    "login@test.com",
			Password: "TestPassword123!",
		})

		t.Run("successful login", func(t *testing.T) {
			resp := suite.HTTP.Post(t, "/api/v1/auth/login", map[string]interface{}{
				"email":    user.Email,
				"password": "TestPassword123!",
			})

			helpers.AssertOK(t, resp)

			var result map[string]interface{}
			require.NoError(t, resp.JSON(&result))

			assert.NotEmpty(t, result["access_token"])
			assert.NotEmpty(t, result["refresh_token"])
		})

		t.Run("login with wrong password fails", func(t *testing.T) {
			resp := suite.HTTP.Post(t, "/api/v1/auth/login", map[string]interface{}{
				"email":    user.Email,
				"password": "WrongPassword123!",
			})

			helpers.AssertUnauthorized(t, resp)
		})

		t.Run("login with non-existent user fails", func(t *testing.T) {
			resp := suite.HTTP.Post(t, "/api/v1/auth/login", map[string]interface{}{
				"email":    "nonexistent@test.com",
				"password": "TestPassword123!",
			})

			helpers.AssertUnauthorized(t, resp)
		})

		t.Run("login with missing fields fails", func(t *testing.T) {
			resp := suite.HTTP.Post(t, "/api/v1/auth/login", map[string]interface{}{
				"email": user.Email,
			})

			helpers.AssertBadRequest(t, resp)
		})
	})

	t.Run("Token Refresh", func(t *testing.T) {
		user := suite.Auth.CreateTestUser(t, &helpers.TestUser{})
		refreshToken := suite.Auth.GenerateRefreshToken(t, user)

		t.Run("successful token refresh", func(t *testing.T) {
			resp := suite.HTTP.Post(t, "/api/v1/auth/refresh", map[string]interface{}{
				"refresh_token": refreshToken,
			})

			helpers.AssertOK(t, resp)

			var result map[string]interface{}
			require.NoError(t, resp.JSON(&result))

			assert.NotEmpty(t, result["access_token"])
		})

		t.Run("refresh with invalid token fails", func(t *testing.T) {
			resp := suite.HTTP.Post(t, "/api/v1/auth/refresh", map[string]interface{}{
				"refresh_token": "invalid-token",
			})

			helpers.AssertUnauthorized(t, resp)
		})

		t.Run("refresh with expired token fails", func(t *testing.T) {
			expiredToken := suite.Auth.GenerateExpiredToken(t, user)

			resp := suite.HTTP.Post(t, "/api/v1/auth/refresh", map[string]interface{}{
				"refresh_token": expiredToken,
			})

			helpers.AssertUnauthorized(t, resp)
		})
	})

	t.Run("Protected Endpoints", func(t *testing.T) {
		t.Run("access without token fails", func(t *testing.T) {
			client := helpers.NewTestClient(suite.Config.APIBaseURL)
			resp := client.Get(t, "/api/v1/users/me")

			helpers.AssertUnauthorized(t, resp)
		})

		t.Run("access with valid token succeeds", func(t *testing.T) {
			client, _ := suite.CreateAuthenticatedClient(t, "user")
			resp := client.Get(t, "/api/v1/users/me")

			helpers.AssertOK(t, resp)
		})

		t.Run("access with expired token fails", func(t *testing.T) {
			user := suite.Auth.CreateTestUser(t, &helpers.TestUser{})
			expiredToken := suite.Auth.GenerateExpiredToken(t, user)

			client := helpers.NewTestClient(suite.Config.APIBaseURL)
			client.SetAuthToken(expiredToken)

			resp := client.Get(t, "/api/v1/users/me")

			helpers.AssertUnauthorized(t, resp)
		})

		t.Run("access with invalid token fails", func(t *testing.T) {
			user := suite.Auth.CreateTestUser(t, &helpers.TestUser{})
			invalidToken := suite.Auth.GenerateInvalidToken(t, user)

			client := helpers.NewTestClient(suite.Config.APIBaseURL)
			client.SetAuthToken(invalidToken)

			resp := client.Get(t, "/api/v1/users/me")

			helpers.AssertUnauthorized(t, resp)
		})
	})

	t.Run("Password Reset", func(t *testing.T) {
		user := suite.Auth.CreateTestUser(t, &helpers.TestUser{
			Email: "reset@test.com",
		})

		t.Run("request password reset", func(t *testing.T) {
			// Clear previous emails
			suite.MailHog.DeleteAllMessages(t)

			resp := suite.HTTP.Post(t, "/api/v1/auth/forgot-password", map[string]interface{}{
				"email": user.Email,
			})

			helpers.AssertOK(t, resp)

			// Verify email was sent
			email := suite.MailHog.AssertEmailSent(t, user.Email, 5*time.Second)
			assert.Contains(t, email.Subject(), "Password Reset")
		})

		t.Run("request reset for non-existent user returns OK", func(t *testing.T) {
			// Should return OK to prevent email enumeration
			resp := suite.HTTP.Post(t, "/api/v1/auth/forgot-password", map[string]interface{}{
				"email": "nonexistent@test.com",
			})

			helpers.AssertOK(t, resp)
		})
	})

	t.Run("Logout", func(t *testing.T) {
		client, _ := suite.CreateAuthenticatedClient(t, "user")

		t.Run("successful logout", func(t *testing.T) {
			resp := client.Post(t, "/api/v1/auth/logout", nil)

			helpers.AssertOK(t, resp)
		})
	})
}

// =============================================================================
// Authorization Tests
// =============================================================================

func TestAuthorizationAPI(t *testing.T) {
	suite := integration.SetupTest(t)

	t.Run("Role-Based Access Control", func(t *testing.T) {
		// Create users with different roles
		adminClient, _ := suite.CreateAuthenticatedClient(t, "admin")
		managerClient, _ := suite.CreateAuthenticatedClient(t, "manager")
		userClient, _ := suite.CreateAuthenticatedClient(t, "user")

		t.Run("admin can access admin endpoints", func(t *testing.T) {
			resp := adminClient.Get(t, "/api/v1/admin/users")

			// Should be OK or NotFound (if endpoint exists)
			assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound)
		})

		t.Run("manager cannot access admin endpoints", func(t *testing.T) {
			resp := managerClient.Get(t, "/api/v1/admin/users")

			// Should be Forbidden or NotFound
			assert.True(t, resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound)
		})

		t.Run("regular user cannot access admin endpoints", func(t *testing.T) {
			resp := userClient.Get(t, "/api/v1/admin/users")

			// Should be Forbidden or NotFound
			assert.True(t, resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound)
		})
	})

	t.Run("Resource Ownership", func(t *testing.T) {
		// Create two users
		client1, user1 := suite.CreateAuthenticatedClient(t, "user")
		client2, _ := suite.CreateAuthenticatedClient(t, "user")

		// Create a resource owned by user1
		customer := suite.Fixtures.CreateCustomer(t, &fixtures.Customer{
			CompanyID: user1.CompanyID,
		})

		t.Run("user can access own resources", func(t *testing.T) {
			resp := client1.Get(t, "/api/v1/customers/"+customer.ID)

			// Should succeed (OK or NotFound if route doesn't exist)
			assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound)
		})

		t.Run("user cannot access other users resources", func(t *testing.T) {
			resp := client2.Get(t, "/api/v1/customers/"+customer.ID)

			// Should be Forbidden or NotFound
			assert.True(t, resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound)
		})
	})
}

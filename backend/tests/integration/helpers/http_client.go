//go:build integration

package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test HTTP Client
// =============================================================================

// TestClient provides HTTP client functionality for integration tests
type TestClient struct {
	baseURL    string
	httpClient *http.Client
	authToken  string
	headers    map[string]string
}

// NewTestClient creates a new test HTTP client
func NewTestClient(baseURL string) *TestClient {
	return &TestClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		headers: make(map[string]string),
	}
}

// NewTestClientWithServer creates a test client using httptest.Server
func NewTestClientWithServer(router *gin.Engine) (*TestClient, *httptest.Server) {
	server := httptest.NewServer(router)
	return NewTestClient(server.URL), server
}

// SetAuthToken sets the authorization token for subsequent requests
func (c *TestClient) SetAuthToken(token string) {
	c.authToken = token
}

// ClearAuthToken clears the authorization token
func (c *TestClient) ClearAuthToken() {
	c.authToken = ""
}

// SetHeader sets a custom header for subsequent requests
func (c *TestClient) SetHeader(key, value string) {
	c.headers[key] = value
}

// ClearHeaders clears all custom headers
func (c *TestClient) ClearHeaders() {
	c.headers = make(map[string]string)
}

// =============================================================================
// Response Wrapper
// =============================================================================

// Response wraps HTTP response for easier testing
type Response struct {
	*http.Response
	Body    []byte
	RawBody io.ReadCloser
}

// JSON decodes the response body into the given struct
func (r *Response) JSON(v interface{}) error {
	return json.Unmarshal(r.Body, v)
}

// JSONMap decodes the response body into a map
func (r *Response) JSONMap() (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal(r.Body, &result)
	return result, err
}

// String returns the response body as a string
func (r *Response) String() string {
	return string(r.Body)
}

// =============================================================================
// HTTP Methods
// =============================================================================

// Get performs a GET request
func (c *TestClient) Get(t *testing.T, path string) *Response {
	t.Helper()
	return c.do(t, http.MethodGet, path, nil)
}

// Post performs a POST request with JSON body
func (c *TestClient) Post(t *testing.T, path string, body interface{}) *Response {
	t.Helper()
	return c.do(t, http.MethodPost, path, body)
}

// Put performs a PUT request with JSON body
func (c *TestClient) Put(t *testing.T, path string, body interface{}) *Response {
	t.Helper()
	return c.do(t, http.MethodPut, path, body)
}

// Patch performs a PATCH request with JSON body
func (c *TestClient) Patch(t *testing.T, path string, body interface{}) *Response {
	t.Helper()
	return c.do(t, http.MethodPatch, path, body)
}

// Delete performs a DELETE request
func (c *TestClient) Delete(t *testing.T, path string) *Response {
	t.Helper()
	return c.do(t, http.MethodDelete, path, nil)
}

// DeleteWithBody performs a DELETE request with JSON body
func (c *TestClient) DeleteWithBody(t *testing.T, path string, body interface{}) *Response {
	t.Helper()
	return c.do(t, http.MethodDelete, path, body)
}

// =============================================================================
// Internal Methods
// =============================================================================

func (c *TestClient) do(t *testing.T, method, path string, body interface{}) *Response {
	t.Helper()

	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err, "Failed to marshal request body")
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	require.NoError(t, err, "Failed to create request")

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	// Perform request
	resp, err := c.httpClient.Do(req)
	require.NoError(t, err, "Request failed")

	// Read body
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")
	resp.Body.Close()

	return &Response{
		Response: resp,
		Body:     respBody,
	}
}

// =============================================================================
// Request Builder (for more complex requests)
// =============================================================================

// RequestBuilder allows building complex HTTP requests
type RequestBuilder struct {
	client  *TestClient
	method  string
	path    string
	body    interface{}
	headers map[string]string
	query   map[string]string
}

// NewRequest creates a new request builder
func (c *TestClient) NewRequest(method, path string) *RequestBuilder {
	return &RequestBuilder{
		client:  c,
		method:  method,
		path:    path,
		headers: make(map[string]string),
		query:   make(map[string]string),
	}
}

// WithBody sets the request body
func (rb *RequestBuilder) WithBody(body interface{}) *RequestBuilder {
	rb.body = body
	return rb
}

// WithHeader adds a header
func (rb *RequestBuilder) WithHeader(key, value string) *RequestBuilder {
	rb.headers[key] = value
	return rb
}

// WithQuery adds a query parameter
func (rb *RequestBuilder) WithQuery(key, value string) *RequestBuilder {
	rb.query[key] = value
	return rb
}

// Do executes the request
func (rb *RequestBuilder) Do(t *testing.T) *Response {
	t.Helper()

	// Build query string
	queryString := ""
	if len(rb.query) > 0 {
		queryString = "?"
		for key, value := range rb.query {
			if len(queryString) > 1 {
				queryString += "&"
			}
			queryString += fmt.Sprintf("%s=%s", key, value)
		}
	}

	// Temporarily add custom headers
	originalHeaders := rb.client.headers
	for key, value := range rb.headers {
		rb.client.headers[key] = value
	}
	defer func() { rb.client.headers = originalHeaders }()

	return rb.client.do(t, rb.method, rb.path+queryString, rb.body)
}

// =============================================================================
// Assertion Helpers
// =============================================================================

// AssertStatus asserts the response status code
func AssertStatus(t *testing.T, resp *Response, expected int) {
	t.Helper()
	require.Equal(t, expected, resp.StatusCode, "Unexpected status code. Body: %s", resp.String())
}

// AssertOK asserts HTTP 200 OK
func AssertOK(t *testing.T, resp *Response) {
	t.Helper()
	AssertStatus(t, resp, http.StatusOK)
}

// AssertCreated asserts HTTP 201 Created
func AssertCreated(t *testing.T, resp *Response) {
	t.Helper()
	AssertStatus(t, resp, http.StatusCreated)
}

// AssertNoContent asserts HTTP 204 No Content
func AssertNoContent(t *testing.T, resp *Response) {
	t.Helper()
	AssertStatus(t, resp, http.StatusNoContent)
}

// AssertBadRequest asserts HTTP 400 Bad Request
func AssertBadRequest(t *testing.T, resp *Response) {
	t.Helper()
	AssertStatus(t, resp, http.StatusBadRequest)
}

// AssertUnauthorized asserts HTTP 401 Unauthorized
func AssertUnauthorized(t *testing.T, resp *Response) {
	t.Helper()
	AssertStatus(t, resp, http.StatusUnauthorized)
}

// AssertForbidden asserts HTTP 403 Forbidden
func AssertForbidden(t *testing.T, resp *Response) {
	t.Helper()
	AssertStatus(t, resp, http.StatusForbidden)
}

// AssertNotFound asserts HTTP 404 Not Found
func AssertNotFound(t *testing.T, resp *Response) {
	t.Helper()
	AssertStatus(t, resp, http.StatusNotFound)
}

// AssertConflict asserts HTTP 409 Conflict
func AssertConflict(t *testing.T, resp *Response) {
	t.Helper()
	AssertStatus(t, resp, http.StatusConflict)
}

// AssertTooManyRequests asserts HTTP 429 Too Many Requests
func AssertTooManyRequests(t *testing.T, resp *Response) {
	t.Helper()
	AssertStatus(t, resp, http.StatusTooManyRequests)
}

// AssertInternalServerError asserts HTTP 500 Internal Server Error
func AssertInternalServerError(t *testing.T, resp *Response) {
	t.Helper()
	AssertStatus(t, resp, http.StatusInternalServerError)
}

// AssertJSONContains asserts the response JSON contains expected fields
func AssertJSONContains(t *testing.T, resp *Response, expected map[string]interface{}) {
	t.Helper()
	actual, err := resp.JSONMap()
	require.NoError(t, err, "Failed to parse response as JSON")

	for key, expectedValue := range expected {
		actualValue, exists := actual[key]
		require.True(t, exists, "Key '%s' not found in response", key)
		require.Equal(t, expectedValue, actualValue, "Value mismatch for key '%s'", key)
	}
}

// AssertHeader asserts a response header value
func AssertHeader(t *testing.T, resp *Response, key, expected string) {
	t.Helper()
	actual := resp.Header.Get(key)
	require.Equal(t, expected, actual, "Header '%s' mismatch", key)
}

// AssertHeaderExists asserts a response header exists
func AssertHeaderExists(t *testing.T, resp *Response, key string) {
	t.Helper()
	actual := resp.Header.Get(key)
	require.NotEmpty(t, actual, "Header '%s' not found", key)
}

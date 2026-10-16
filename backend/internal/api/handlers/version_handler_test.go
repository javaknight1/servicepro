package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("APP_VERSION", "0.9.0")
	t.Setenv("GIT_SHA", "deadbeef")

	r := gin.New()
	r.GET("/version", GetVersion)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/version", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["git_sha"] != "deadbeef" || body["app_version"] != "0.9.0" {
		t.Fatalf("body = %v", body)
	}
}

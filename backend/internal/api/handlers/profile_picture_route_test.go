package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// /users/me and /users/:id/profile-picture share a prefix; gin must accept
// both, and a bad ID is a 404, not a 500.
func TestGetProfilePicture_RouteAndBadID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewProfilePictureHandler(nil) // rejected before the service is reached
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", uuid.New()) })
	r.GET("/users/me", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/users/:id/profile-picture", h.GetProfilePicture)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users/not-a-uuid/profile-picture", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

package handlers

import (
	"bytes"
	"mime/multipart"
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

// T160: an oversized avatar upload is refused with 413 before the multipart
// form is spooled, not read in full.
func TestUploadProfilePicture_RejectsOversizedBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewProfilePictureHandler(nil) // rejected before the service is reached

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", "big.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(bytes.Repeat([]byte{0}, 3<<20))
	_ = mw.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", uuid.New())
	c.Request = httptest.NewRequest(http.MethodPost, "/users/me/profile-picture", &body)
	c.Request.Header.Set("Content-Type", mw.FormDataContentType())

	h.UploadProfilePicture(c)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413 (%s)", w.Code, w.Body.String())
	}
}

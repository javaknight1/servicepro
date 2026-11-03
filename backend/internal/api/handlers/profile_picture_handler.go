package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// ProfilePictureHandler handles profile picture operations
type ProfilePictureHandler struct {
	service *services.ProfilePictureService
}

// NewProfilePictureHandler creates a new profile picture handler
func NewProfilePictureHandler(service *services.ProfilePictureService) *ProfilePictureHandler {
	return &ProfilePictureHandler{service: service}
}

// UploadProfilePicture handles POST /api/v1/users/me/profile-picture
// @Summary		Upload profile picture
// @Description	Uploads a new profile picture for the current user (max 2MB, JPEG or PNG)
// @Tags			Users
// @Accept			multipart/form-data
// @Produce		json
// @Security		BearerAuth
// @Param			file	formData	file	true	"Profile picture file"
// @Success		200		{object}	models.ProfilePictureUploadResponse
// @Failure		400		{object}	models.ErrorResponse
// @Failure		401		{object}	models.ErrorResponse
// @Failure		413		{object}	models.ErrorResponse
// @Failure		500		{object}	models.ErrorResponse
// @Router			/users/me/profile-picture [post]
func (h *ProfilePictureHandler) UploadProfilePicture(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userIDStr, exists := c.Get("user_id")
	if !exists {
		apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	userID, ok := userIDStr.(uuid.UUID)
	if !ok {
		// Try parsing as string
		if idStr, ok := userIDStr.(string); ok {
			var err error
			userID, err = uuid.Parse(idStr)
			if err != nil {
				apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
					Error:   "unauthorized",
					Message: "Invalid user ID",
				})
				return
			}
		} else {
			apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid user ID format",
			})
			return
		}
	}

	// Get uploaded file
	// Cap the body before the multipart form is parsed (T160), so an
	// oversized upload is refused instead of being spooled to disk.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, services.MaxUploadSize+64<<10)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			apierror.WriteResponse(c, http.StatusRequestEntityTooLarge, models.ErrorResponse{
				Error:   "file_too_large",
				Message: "File size exceeds maximum allowed (2MB)",
			})
			return
		}
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "bad_request",
			Message: "No file uploaded",
		})
		return
	}
	defer file.Close()

	// Get content type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Upload profile picture
	url, err := h.service.UploadProfilePicture(c.Request.Context(), userID, file, header.Size, contentType)
	if err != nil {
		if errors.Is(err, services.ErrFileTooLarge) {
			apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
				Error:   "file_too_large",
				Message: "File size exceeds maximum allowed (2MB)",
			})
			return
		}
		if errors.Is(err, services.ErrImageTooLarge) {
			apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
				Error:   "image_too_large",
				Message: "Image dimensions exceed the maximum allowed (4000x4000)",
			})
			return
		}
		if errors.Is(err, services.ErrInvalidImageFormat) {
			apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_format",
				Message: "Invalid image format. Only JPEG and PNG are supported",
			})
			return
		}
		apierror.Internal(c, err, "Failed to upload profile picture")
		return
	}

	c.JSON(http.StatusOK, models.ProfilePictureUploadResponse{
		ProfilePictureURL: url,
	})
}

// DeleteProfilePicture handles DELETE /api/v1/users/me/profile-picture
// @Summary		Delete profile picture
// @Description	Deletes the current user's profile picture
// @Tags			Users
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Success		200	{object}	map[string]string
// @Failure		401	{object}	models.ErrorResponse
// @Failure		500	{object}	models.ErrorResponse
// @Router			/users/me/profile-picture [delete]
func (h *ProfilePictureHandler) DeleteProfilePicture(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userIDStr, exists := c.Get("user_id")
	if !exists {
		apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	userID, ok := userIDStr.(uuid.UUID)
	if !ok {
		// Try parsing as string
		if idStr, ok := userIDStr.(string); ok {
			var err error
			userID, err = uuid.Parse(idStr)
			if err != nil {
				apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
					Error:   "unauthorized",
					Message: "Invalid user ID",
				})
				return
			}
		} else {
			apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid user ID format",
			})
			return
		}
	}

	// Delete profile picture
	if err := h.service.DeleteProfilePicture(c.Request.Context(), userID); err != nil {
		apierror.Internal(c, err, "Failed to delete profile picture")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile picture deleted successfully",
	})
}

// GetProfilePicture handles GET /api/v1/users/{id}/profile-picture
// @Summary		Get profile picture
// @Description	Redirects to a short-lived link to the user's profile picture. The caller must be that user or share an active organization with them; otherwise, or when there is no picture, 404. The stored profile_picture_url points here.
// @Tags			Users
// @Security		BearerAuth
// @Param			id	path	string	true	"User ID"
// @Success		302
// @Failure		401	{object}	models.ErrorResponse
// @Failure		404	{object}	models.ErrorResponse
// @Router			/users/{id}/profile-picture [get]
func (h *ProfilePictureHandler) GetProfilePicture(c *gin.Context) {
	viewer, err := getUserIDFromContext(c)
	if err != nil {
		apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{Error: "unauthorized", Message: "User not authenticated"})
		return
	}
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.WriteResponse(c, http.StatusNotFound, models.ErrorResponse{Error: "not_found", Message: "Profile picture not found"})
		return
	}

	url, err := h.service.PictureURL(c.Request.Context(), viewer, userID)
	if errors.Is(err, services.ErrNoProfilePicture) {
		apierror.WriteResponse(c, http.StatusNotFound, models.ErrorResponse{Error: "not_found", Message: "Profile picture not found"})
		return
	}
	if err != nil {
		apierror.Internal(c, err, "Failed to load profile picture")
		return
	}
	// The link lives an hour; let the browser reuse the redirect for a while.
	c.Header("Cache-Control", "private, max-age=600")
	c.Redirect(http.StatusFound, url)
}

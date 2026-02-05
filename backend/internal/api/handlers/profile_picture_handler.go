package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

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
// @Failure		500		{object}	models.ErrorResponse
// @Router			/users/me/profile-picture [post]
func (h *ProfilePictureHandler) UploadProfilePicture(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
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
				c.JSON(http.StatusUnauthorized, models.ErrorResponse{
					Error:   "unauthorized",
					Message: "Invalid user ID",
				})
				return
			}
		} else {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid user ID format",
			})
			return
		}
	}

	// Get uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
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
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "file_too_large",
				Message: "File size exceeds maximum allowed (2MB)",
			})
			return
		}
		if errors.Is(err, services.ErrInvalidImageFormat) {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_format",
				Message: "Invalid image format. Only JPEG and PNG are supported",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "upload_failed",
			Message: "Failed to upload profile picture",
		})
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
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
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
				c.JSON(http.StatusUnauthorized, models.ErrorResponse{
					Error:   "unauthorized",
					Message: "Invalid user ID",
				})
				return
			}
		} else {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid user ID format",
			})
			return
		}
	}

	// Delete profile picture
	if err := h.service.DeleteProfilePicture(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "delete_failed",
			Message: "Failed to delete profile picture",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile picture deleted successfully",
	})
}

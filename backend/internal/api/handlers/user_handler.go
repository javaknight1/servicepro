package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
)

// UserHandler handles user-related endpoints
type UserHandler struct {
	userRepo repository.UserRepositoryGormInterface
}

// NewUserHandler creates a new user handler
func NewUserHandler(userRepo repository.UserRepositoryGormInterface) *UserHandler {
	return &UserHandler{userRepo: userRepo}
}

// GetCurrentUser handles GET /api/v1/users/me
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	var userID uuid.UUID
	switch v := userIDStr.(type) {
	case uuid.UUID:
		userID = v
	case string:
		var err error
		userID, err = uuid.Parse(v)
		if err != nil {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid user ID",
			})
			return
		}
	default:
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid user ID format",
		})
		return
	}

	// Get user from repository
	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "not_found",
			Message: "User not found",
		})
		return
	}

	// Return user data (User model already has json tags that exclude sensitive fields)
	c.JSON(http.StatusOK, user)
}

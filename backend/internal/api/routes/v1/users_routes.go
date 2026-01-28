package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
)

// SetupUserRoutes configures user profile routes
func SetupUserRoutes(router *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	users := router.Group("/users")
	users.Use(cfg.Middleware.PermMiddleware.RequireAuth())
	{
		// Current user endpoint
		users.GET("/me", cfg.Handlers.User.GetCurrentUser)

		// Update profile endpoint
		users.PATCH("/me/profile", cfg.Handlers.User.UpdateProfile)

		// Profile picture endpoints (optional - depends on storage)
		if cfg.Handlers.ProfilePicture != nil {
			users.POST("/me/profile-picture", cfg.Handlers.ProfilePicture.UploadProfilePicture)
			users.DELETE("/me/profile-picture", cfg.Handlers.ProfilePicture.DeleteProfilePicture)
		}
	}
}

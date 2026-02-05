package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/version"
)

// VersionResponse represents the version information returned by the API
type VersionResponse struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildTime string `json:"build_time"`
}

// GetVersion handles GET /api/v1/version
// @Summary		Get API version
// @Description	Returns the API version, git commit, and build time
// @Tags			System
// @Accept			json
// @Produce		json
// @Success		200	{object}	VersionResponse
// @Router			/version [get]
func GetVersion(c *gin.Context) {
	c.JSON(http.StatusOK, VersionResponse{
		Version:   version.Version,
		GitCommit: version.GitCommit,
		BuildTime: version.BuildTime,
	})
}

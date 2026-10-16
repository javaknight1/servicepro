package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/version"
)

// GetVersion handles GET /version (unversioned, at the server root so the
// edge Worker can proxy it and deploy checks can confirm what is live).
// @Summary		Build version
// @Description	Returns the git SHA and app version of the running build (from the GIT_SHA / APP_VERSION env baked into the image). Mounted at the server root, outside /api.
// @Tags			System
// @Produce		json
// @Success		200	{object}	version.Info
// @Router			/version [get]
func GetVersion(c *gin.Context) {
	c.JSON(http.StatusOK, version.Get())
}

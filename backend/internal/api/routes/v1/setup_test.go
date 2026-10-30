package v1

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
)

// T259: the export-job API has no screen, so it must not be routable unless
// EXPORT_JOBS_ENABLED is set.
func TestExportJobRoutesOffByDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for name, cfg := range map[string]*routeconfigs.RouteConfig{
		"nil config":   {},
		"flag not set": {Config: &config.Config{}},
	} {
		t.Run(name, func(t *testing.T) {
			engine := gin.New()
			setupExportJobRoutes(engine.Group("/api/v1"), cfg)
			for _, r := range engine.Routes() {
				if strings.HasPrefix(r.Path, "/api/v1/exports") {
					t.Fatalf("export-job route registered with the flag off: %s %s", r.Method, r.Path)
				}
			}
		})
	}
}

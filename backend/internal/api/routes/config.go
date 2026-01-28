package routes

import (
	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
)

// Type aliases for backwards compatibility - actual types are in routeconfigs package
type (
	RouteConfig  = routeconfigs.RouteConfig
	Clients      = routeconfigs.Clients
	Repositories = routeconfigs.Repositories
	Services     = routeconfigs.Services
	Middleware   = routeconfigs.Middleware
	Handlers     = routeconfigs.Handlers
)

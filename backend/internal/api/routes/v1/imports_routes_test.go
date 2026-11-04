package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/javaknight1/servicepro/backend/internal/imports"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
)

// Each entity's import routes check that entity's own permission; an
// unknown entity is 404 before any handler runs.
func TestImportRoutes_PermissionPerEntity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var checked string
	check := entityPermission(func(e imports.Entity) gin.HandlerFunc {
		perm := importPermissions[e][0]
		return func(c *gin.Context) { checked = perm; c.Next() }
	})
	r := gin.New()
	r.POST("/imports/:entity/preview", check, func(c *gin.Context) { c.Status(http.StatusOK) })

	for entity, want := range map[string]string{
		"customers": permissions.CustomersCreate,
		"sites":     permissions.CustomersUpdate,
		"jobs":      permissions.JobsCreate,
		"invoices":  permissions.InvoicesCreate,
	} {
		checked = ""
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/imports/"+entity+"/preview", nil))
		assert.Equal(t, http.StatusOK, w.Code, entity)
		assert.Equal(t, want, checked, entity)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/imports/users/preview", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)

	for e, perms := range importPermissions {
		assert.NotEmpty(t, perms[1], "read permission for %s", e)
	}
}

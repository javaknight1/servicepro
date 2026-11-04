package v1

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/api/handlers"
	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/services"
	permissionsSvc "github.com/javaknight1/servicepro/backend/internal/services/permissions"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
	"github.com/javaknight1/servicepro/backend/pkg/auth"
	storagemock "github.com/javaknight1/servicepro/backend/pkg/clients/storage/mock"
	"github.com/javaknight1/servicepro/backend/pkg/database"
)

// T035 end to end: real middleware (auth, tenant, permissions with the
// access scope), real job handler/service/repository, migrated Postgres.
// Needs SERVICEPRO_TEST_PG_URL (a Postgres server URL whose role can CREATE
// DATABASE); skipped otherwise.

// techTestDB creates a throwaway database with every migration applied.
func techTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	base := os.Getenv("SERVICEPRO_TEST_PG_URL")
	if base == "" {
		t.Skip("SERVICEPRO_TEST_PG_URL not set; skipping Postgres-backed test")
	}
	admin, err := gorm.Open(postgres.Open(base), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	buf := make([]byte, 6)
	_, _ = rand.Read(buf)
	name := "sp_test_" + hex.EncodeToString(buf)
	require.NoError(t, admin.Exec("CREATE DATABASE "+name).Error)

	u, err := url.Parse(base)
	require.NoError(t, err)
	u.Path = "/" + name
	require.NoError(t, database.RunMigrations(u.String()))

	db, err := gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, tenancy.Register(db))
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		_ = admin.Exec("DROP DATABASE IF EXISTS " + name + " WITH (FORCE)").Error
		if sqlDB, err := admin.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

type techFixture struct {
	db     *gorm.DB
	engine *gin.Engine
	jwt    *auth.JWTManager
	// store is the in-memory object store behind job attachments (T174).
	store *storagemock.Client

	tenantA, tenantB               uuid.UUID
	ownerA, officeA, techA, techA2 uuid.UUID
	ownerB                         uuid.UUID
	customerA                      uuid.UUID
	assignedJob, otherJob, jobB    *models.Job
}

func roleID(t *testing.T, db *gorm.DB, name string) uuid.UUID {
	t.Helper()
	var role models.Role
	require.NoError(t, db.Where("name = ?", name).First(&role).Error, "role %s", name)
	return role.ID
}

func setupTechFixture(t *testing.T) *techFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := techTestDB(t)
	sys := tenancy.WithSystem(context.Background())
	f := &techFixture{db: db,
		tenantA: uuid.New(), tenantB: uuid.New(),
		ownerA: uuid.New(), officeA: uuid.New(), techA: uuid.New(), techA2: uuid.New(), ownerB: uuid.New(),
		customerA: uuid.New(),
	}
	exec := func(sql string, args ...interface{}) {
		require.NoError(t, db.WithContext(sys).Exec(sql, args...).Error)
	}
	for _, u := range []uuid.UUID{f.ownerA, f.officeA, f.techA, f.techA2, f.ownerB} {
		exec(`INSERT INTO users (id, email, password_hash) VALUES (?, ?, 'x')`, u, "u-"+u.String()[:8]+"@example.com")
	}
	exec(`INSERT INTO tenants (id, name, slug, owner_id) VALUES (?, 'A', ?, ?)`, f.tenantA, "a-"+f.tenantA.String()[:8], f.ownerA)
	exec(`INSERT INTO tenants (id, name, slug, owner_id) VALUES (?, 'B', ?, ?)`, f.tenantB, "b-"+f.tenantB.String()[:8], f.ownerB)
	member := func(tenant, user uuid.UUID, role string) {
		exec(`INSERT INTO tenant_users (tenant_id, user_id, role_id, is_active, accepted_at) VALUES (?, ?, ?, TRUE, NOW())`,
			tenant, user, roleID(t, db, role))
	}
	member(f.tenantA, f.ownerA, "admin")
	member(f.tenantA, f.officeA, "manager")
	member(f.tenantA, f.techA, "technician")
	member(f.tenantA, f.techA2, "technician")
	member(f.tenantB, f.ownerB, "admin")

	exec(`INSERT INTO customers (id, tenant_id, first_name, last_name, email, phone_primary,
			billing_address_street, billing_address_city, billing_address_state, billing_address_zip, nte_limit)
		 VALUES (?, ?, 'Pat', 'Customer', 'pat@example.com', '555-555-5555', '1 Main St', 'Springfield', 'IL', '62701', 500)`,
		f.customerA, f.tenantA)
	customerB := uuid.New()
	exec(`INSERT INTO customers (id, tenant_id, first_name, last_name, email, phone_primary,
			billing_address_street, billing_address_city, billing_address_state, billing_address_zip)
		 VALUES (?, ?, 'Bo', 'Other', 'bo@example.com', '555-555-0000', '2 Main St', 'Springfield', 'IL', '62701')`,
		customerB, f.tenantB)

	newJob := func(tenant, customer uuid.UUID, number string, assignee *uuid.UUID) *models.Job {
		ctx := tenancy.WithTenant(context.Background(), tenant)
		end := time.Now().Add(-time.Hour).Unix()
		job := &models.Job{
			ID: uuid.New(), TenantID: tenant, CustomerID: customer, JobNumber: number,
			Title: "Job " + number, JobType: models.JobTypeMaintenance, Status: models.JobStatusScheduled,
			Priority: models.JobPriorityNormal, EstimatedCost: 1234.5, ScheduledEndAt: &end,
			CreatedBy: f.ownerA,
		}
		require.NoError(t, db.WithContext(ctx).Omit("Customer").Create(job).Error)
		if assignee != nil {
			require.NoError(t, db.WithContext(ctx).Create(&models.JobAssignment{
				JobID: job.ID, UserID: *assignee, Role: "technician", AssignedAt: time.Now().Unix(),
			}).Error)
		}
		return job
	}
	f.assignedJob = newJob(f.tenantA, f.customerA, "TECH-1", &f.techA)
	f.otherJob = newJob(f.tenantA, f.customerA, "TECH-2", &f.techA2)
	f.jobB = newJob(f.tenantB, customerB, "TECH-3", nil)

	// Middleware exactly as routes.setupMiddleware wires it.
	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	f.jwt = auth.NewJWTManager(&config.JWTConfig{Secret: "tech-test-secret", AccessExpiration: time.Hour, RefreshExpiration: time.Hour})
	checker := permissionsSvc.NewPermissionChecker(repository.NewPermissionRepository(db), redisClient)
	permMW := middleware.NewPermissionMiddleware(checker, f.jwt, nil, "access_token")
	tenantMW := middleware.NewTenantMiddleware(repository.NewTenantRepository(db))
	tenantMW.SetAccessScope(middleware.AccessScope(checker))

	f.store = storagemock.NewClient(nil)
	jobService := services.NewJobService(repository.NewJobRepository(db), repository.NewCustomerRepository(db), db)
	cfg := &routeconfigs.RouteConfig{
		Middleware: &routeconfigs.Middleware{PermMiddleware: permMW, TenantMW: tenantMW, PermChecker: checker},
		// Handlers other than jobs stay nil: every route a technician may not
		// use must be refused before its handler runs.
		Handlers: &routeconfigs.Handlers{
			Job: handlers.NewJobHandler(jobService),
			JobApproval: handlers.NewJobApprovalHandler(
				services.NewJobApprovalService(db, repository.NewJobApprovalRepository(db))),
			JobAttachment: handlers.NewJobAttachmentHandler(
				services.NewJobAttachmentService(repository.NewJobAttachmentRepository(db), f.store)),
		},
	}
	f.engine = gin.New()
	api := f.engine.Group("/api/v1")
	SetupJobRoutes(api, cfg)
	SetupCustomerRoutes(api, cfg)
	SetupInvoiceRoutes(api, cfg)
	SetupQuoteRoutes(api, cfg)
	SetupReportRoutes(api, cfg)
	SetupAgreementRoutes(api, cfg)
	SetupConflictRoutes(api, cfg)
	SetupTenantRoutes(api, cfg)
	return f
}

// do sends a request as user in tenant and returns the recorder.
func (f *techFixture) do(t *testing.T, user, tenant uuid.UUID, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, "/api/v1"+path, &buf)
	token, err := f.jwt.GenerateAccessToken(user.String(), "u@example.com")
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", tenant.String())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	f.engine.ServeHTTP(w, req)
	return w
}

type jobList struct {
	Jobs  []models.JobResponse `json:"jobs"`
	Total int64                `json:"total"`
}

func TestPostgresTechnicianSeesOnlyAssignedJobs(t *testing.T) {
	f := setupTechFixture(t)
	tech, A := f.techA, f.tenantA
	assigned, other := f.assignedJob.ID.String(), f.otherJob.ID.String()

	t.Run("list returns only assigned jobs, without money", func(t *testing.T) {
		w := f.do(t, tech, A, http.MethodGet, "/jobs", nil)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var got jobList
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
		require.Len(t, got.Jobs, 1)
		assert.Equal(t, int64(1), got.Total)
		assert.Equal(t, f.assignedJob.ID, got.Jobs[0].ID)
		assert.Zero(t, got.Jobs[0].EstimatedCost, "technicians never see prices")
		require.NotNil(t, got.Jobs[0].Customer, "customer contact stays visible")
		assert.Equal(t, "555-555-5555", got.Jobs[0].Customer.PhonePrimary)
		assert.Nil(t, got.Jobs[0].Customer.NTELimit, "billing rules are hidden")
	})

	t.Run("repository: an assignee filter can't widen the row filter", func(t *testing.T) {
		ctx := tenancy.WithAssignedJobsOnly(tenancy.WithTenant(context.Background(), A), tech)
		repo := repository.NewJobRepository(f.db).WithContext(ctx)
		jobs, total, err := repo.List(&models.JobListFilter{AssignedUserID: &f.techA2})
		require.NoError(t, err)
		assert.Empty(t, jobs)
		assert.Zero(t, total)
		_, err = repo.GetByID(f.otherJob.ID)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		_, err = repo.GetByJobNumber("TECH-2")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		require.Error(t, repo.AddAssignment(&models.JobAssignment{JobID: f.otherJob.ID, UserID: tech, Role: "technician"}),
			"can't attach themselves to someone else's job")
	})

	t.Run("get by id and number: assigned 200, others 404", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, f.do(t, tech, A, http.MethodGet, "/jobs/"+assigned, nil).Code)
		assert.Equal(t, http.StatusNotFound, f.do(t, tech, A, http.MethodGet, "/jobs/"+other, nil).Code)
		assert.Equal(t, http.StatusNotFound, f.do(t, tech, A, http.MethodGet, "/jobs/number/TECH-2", nil).Code)
		assert.Equal(t, http.StatusNotFound, f.do(t, tech, A, http.MethodGet, "/jobs/"+other+"/status-history", nil).Code)
		assert.Equal(t, http.StatusOK, f.do(t, tech, A, http.MethodGet, "/jobs/"+assigned+"/status-history", nil).Code)
	})

	t.Run("updates, notes, materials and transitions on other jobs are 404", func(t *testing.T) {
		note := map[string]interface{}{"note": "on site"}
		material := map[string]interface{}{"name": "Filter", "quantity": 2, "unit": "each"}
		for _, tc := range []struct {
			method, path string
			body         interface{}
		}{
			{http.MethodPut, "/jobs/" + other, map[string]interface{}{"completion_notes": "done"}},
			{http.MethodPost, "/jobs/" + other + "/notes", note},
			{http.MethodPost, "/jobs/" + other + "/materials", material},
			{http.MethodPost, "/jobs/" + other + "/transition", map[string]interface{}{"to_status": "in_progress"}},
		} {
			w := f.do(t, tech, A, tc.method, tc.path, tc.body)
			assert.Equal(t, http.StatusNotFound, w.Code, "%s %s: %s", tc.method, tc.path, w.Body.String())
		}
	})

	t.Run("works their own job: notes, materials, status, completion notes", func(t *testing.T) {
		w := f.do(t, tech, A, http.MethodPost, "/jobs/"+assigned+"/notes", map[string]interface{}{"note": "on site"})
		assert.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		w = f.do(t, tech, A, http.MethodPost, "/jobs/"+assigned+"/materials",
			map[string]interface{}{"name": "Filter", "quantity": 2, "unit": "each"})
		assert.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		w = f.do(t, tech, A, http.MethodPost, "/jobs/"+assigned+"/transition", map[string]interface{}{"to_status": "in_progress"})
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
		w = f.do(t, tech, A, http.MethodPut, "/jobs/"+assigned, map[string]interface{}{"completion_notes": "replaced filter"})
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	})

	t.Run("can't set prices or costs", func(t *testing.T) {
		w := f.do(t, tech, A, http.MethodPut, "/jobs/"+assigned, map[string]interface{}{"estimated_cost": 1})
		assert.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
		w = f.do(t, tech, A, http.MethodPost, "/jobs/"+assigned+"/materials",
			map[string]interface{}{"name": "Filter", "quantity": 1, "unit": "each", "unit_cost": 9.5})
		assert.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	})

	t.Run("stats and overdue count only assigned jobs", func(t *testing.T) {
		w := f.do(t, tech, A, http.MethodGet, "/jobs/stats", nil)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var stats models.JobStats
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &stats))
		assert.Equal(t, int64(1), stats.TotalJobs)

		w = f.do(t, f.techA2, A, http.MethodGet, "/jobs/overdue", nil)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.Contains(t, w.Body.String(), other)
		assert.NotContains(t, w.Body.String(), assigned)
	})
}

func TestPostgresTechnicianIsRefusedMoneyAndAdmin(t *testing.T) {
	f := setupTechFixture(t)
	tech, A := f.techA, f.tenantA
	assigned := f.assignedJob.ID.String()
	someID := uuid.New().String()

	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/invoices"},
		{http.MethodGet, "/invoices/" + someID},
		{http.MethodPost, "/invoices/" + someID + "/payments"},
		{http.MethodGet, "/quotes"},
		{http.MethodGet, "/reports/revenue"},
		{http.MethodGet, "/reports/ar-aging"},
		{http.MethodGet, "/customers"},
		{http.MethodGet, "/customers/" + f.customerA.String()},
		{http.MethodPost, "/customers"},
		{http.MethodPut, "/customers/" + f.customerA.String()},
		{http.MethodGet, "/customers/" + f.customerA.String() + "/sites"},
		{http.MethodGet, "/agreements"},
		{http.MethodGet, "/agreements/" + someID + "/preview"},
		{http.MethodGet, "/jobs/" + assigned + "/approvals"},
		{http.MethodPost, "/jobs/" + assigned + "/approval"},
		{http.MethodPost, "/jobs"},
		{http.MethodDelete, "/jobs/" + assigned},
		{http.MethodPost, "/jobs/" + assigned + "/assign"},
		{http.MethodGet, "/jobs/technicians/" + f.techA2.String() + "/workload"},
		{http.MethodPost, "/conflicts/check"},
		{http.MethodPost, "/jobs/" + assigned + "/invoice"},
		{http.MethodPost, "/quotes/" + someID + "/convert-to-job"},
	} {
		w := f.do(t, tech, A, tc.method, tc.path, map[string]interface{}{})
		assert.Equal(t, http.StatusForbidden, w.Code, "%s %s: %s", tc.method, tc.path, w.Body.String())
	}

	t.Run("can't change their own role or manage the team", func(t *testing.T) {
		admin := roleID(t, f.db, "admin")
		w := f.do(t, tech, A, http.MethodPut, "/tenants/"+A.String()+"/members/"+tech.String()+"/role",
			map[string]interface{}{"role_id": admin})
		assert.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
		w = f.do(t, tech, A, http.MethodPost, "/tenants/"+A.String()+"/invitations",
			map[string]interface{}{"email": "x@example.com", "role_id": admin})
		assert.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
		w = f.do(t, tech, A, http.MethodGet, "/tenants/"+A.String()+"/members", nil)
		assert.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	})
}

func TestPostgresTechnicianCrossTenant(t *testing.T) {
	f := setupTechFixture(t)

	// Not a member of tenant B: the tenant middleware refuses it outright.
	w := f.do(t, f.techA, f.tenantB, http.MethodGet, "/jobs", nil)
	assert.Equal(t, http.StatusForbidden, w.Code, w.Body.String())

	// Tenant B's job doesn't exist from tenant A.
	w = f.do(t, f.techA, f.tenantA, http.MethodGet, "/jobs/"+f.jobB.ID.String(), nil)
	assert.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	// Tenant B's owner sees only B's job, unaffected by A's technicians.
	w = f.do(t, f.ownerB, f.tenantB, http.MethodGet, "/jobs", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var got jobList
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Len(t, got.Jobs, 1)
	assert.Equal(t, f.jobB.ID, got.Jobs[0].ID)
}

func TestPostgresOfficeStaffStillSeeEveryJobAndMoney(t *testing.T) {
	f := setupTechFixture(t)
	for _, user := range []uuid.UUID{f.ownerA, f.officeA} {
		w := f.do(t, user, f.tenantA, http.MethodGet, "/jobs", nil)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var got jobList
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
		require.Len(t, got.Jobs, 2)
		for _, j := range got.Jobs {
			assert.InDelta(t, 1234.5, j.EstimatedCost, 0.001)
		}
		w = f.do(t, user, f.tenantA, http.MethodGet, "/jobs/"+f.otherJob.ID.String(), nil)
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
		w = f.do(t, user, f.tenantA, http.MethodGet, "/jobs/"+f.otherJob.ID.String()+"/approvals", nil)
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	}
}

package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/javaknight1/servicepro/backend/config"
)

func setupTestAccessLogger(t *testing.T) (*AccessLogger, func()) {
	// Create temp directory for logs
	tmpDir := t.TempDir()

	cfg := &config.AccessLoggingConfig{
		Enabled:               true,
		LogSuccessfulRequests: true,
		LogFailedRequests:     true,
		LogDeniedAccess:       true,
		LogSlowRequests:       true,
		SlowRequestThreshold:  100 * time.Millisecond,
		FilterSensitiveData:   true,
		SensitiveHeaders:      []string{"Authorization", "Cookie"},
		SensitiveParams:       []string{"password", "token"},
		LogRequestBody:        true,
		LogResponseBody:       true,
		LogHeaders:            true,
		LogQueryParams:        true,
		MaxBodyLogSize:        1024,
		EnableMetrics:         true,
		MetricsPrefix:         "test",
		TrackResponseTime:     true,
		TrackRequestSize:      true,
		TrackResponseSize:     true,
		LogFormat:             "json",
		IncludeUserInfo:       true,
		IncludeIPInfo:         true,
		LogToFile:             true,
		LogFilePath:           filepath.Join(tmpDir, "access.log"),
		LogToStdout:           false,
		LogRotation: config.LogRotationConfig{
			Enabled:    true,
			MaxSize:    10,
			MaxBackups: 3,
			MaxAge:     7,
			Compress:   false,
		},
		EnableAuditTrail:  true,
		AuditTrailPath:    filepath.Join(tmpDir, "audit.log"),
		AuditSensitiveOps: []string{"delete", "update_permissions"},
	}

	logger, err := NewAccessLogger(cfg)
	if err != nil {
		t.Fatalf("Failed to create access logger: %v", err)
	}

	cleanup := func() {
		// Cleanup is handled by t.TempDir()
	}

	return logger, cleanup
}

func TestAccessLogging_SuccessfulRequest(t *testing.T) {
	logger, cleanup := setupTestAccessLogger(t)
	defer cleanup()

	router := setupTestRouter()
	router.Use(logger.AccessLogging())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify metrics were updated
	metrics := logger.GetMetrics()
	assert.Equal(t, int64(1), metrics.TotalRequests)
	assert.Equal(t, int64(1), metrics.SuccessfulRequests)
	assert.Equal(t, int64(0), metrics.FailedRequests)
}

func TestAccessLogging_FailedRequest(t *testing.T) {
	logger, cleanup := setupTestAccessLogger(t)
	defer cleanup()

	router := setupTestRouter()
	router.Use(logger.AccessLogging())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Verify metrics were updated
	metrics := logger.GetMetrics()
	assert.Equal(t, int64(1), metrics.TotalRequests)
	assert.Equal(t, int64(0), metrics.SuccessfulRequests)
	assert.Equal(t, int64(1), metrics.FailedRequests)
}

func TestAccessLogging_DeniedRequest(t *testing.T) {
	logger, cleanup := setupTestAccessLogger(t)
	defer cleanup()

	router := setupTestRouter()
	router.Use(logger.AccessLogging())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	// Verify metrics were updated
	metrics := logger.GetMetrics()
	assert.Equal(t, int64(1), metrics.TotalRequests)
	assert.Equal(t, int64(1), metrics.DeniedRequests)
}

func TestAccessLogging_SlowRequest(t *testing.T) {
	logger, cleanup := setupTestAccessLogger(t)
	defer cleanup()

	router := setupTestRouter()
	router.Use(logger.AccessLogging())
	router.GET("/test", func(c *gin.Context) {
		// Simulate slow request
		time.Sleep(150 * time.Millisecond)
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify slow request was tracked
	metrics := logger.GetMetrics()
	assert.Equal(t, int64(1), metrics.SlowRequests)
}

func TestAccessLogging_WithUserContext(t *testing.T) {
	logger, cleanup := setupTestAccessLogger(t)
	defer cleanup()

	userID := uuid.New()
	userEmail := "test@example.com"

	router := setupTestRouter()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("user_email", userEmail)
		c.Next()
	})
	router.Use(logger.AccessLogging())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Read log file to verify user info was logged
	logContent, err := os.ReadFile(logger.config.LogFilePath)
	assert.NoError(t, err)
	assert.Contains(t, string(logContent), userID.String())
	assert.Contains(t, string(logContent), userEmail)
}

func TestAccessLogging_RequestBody(t *testing.T) {
	logger, cleanup := setupTestAccessLogger(t)
	defer cleanup()

	router := setupTestRouter()
	router.Use(logger.AccessLogging())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	body := []byte(`{"username":"testuser","email":"test@example.com"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Read log file to verify request body was logged
	logContent, err := os.ReadFile(logger.config.LogFilePath)
	assert.NoError(t, err)
	assert.Contains(t, string(logContent), "testuser")
}

func TestAccessLogging_SensitiveDataFiltering(t *testing.T) {
	logger, cleanup := setupTestAccessLogger(t)
	defer cleanup()

	router := setupTestRouter()
	router.Use(logger.AccessLogging())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	body := []byte(`{"username":"testuser","password":"secret123"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret-token")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Read log file to verify sensitive data was filtered
	logContent, err := os.ReadFile(logger.config.LogFilePath)
	assert.NoError(t, err)

	// Password should be filtered
	assert.Contains(t, string(logContent), "[FILTERED]")
}

func TestAccessLogging_QueryParams(t *testing.T) {
	logger, cleanup := setupTestAccessLogger(t)
	defer cleanup()

	router := setupTestRouter()
	router.Use(logger.AccessLogging())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test?page=1&limit=10", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Read log file to verify query params were logged
	logContent, err := os.ReadFile(logger.config.LogFilePath)
	assert.NoError(t, err)
	assert.Contains(t, string(logContent), "page=1")
	assert.Contains(t, string(logContent), "limit=10")
}

func TestAccessLogging_Metrics(t *testing.T) {
	logger, cleanup := setupTestAccessLogger(t)
	defer cleanup()

	router := setupTestRouter()
	router.Use(logger.AccessLogging())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
	router.POST("/api/users", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"message": "created"})
	})

	// Make multiple requests
	requests := []struct {
		method string
		path   string
		status int
	}{
		{"GET", "/test", http.StatusOK},
		{"GET", "/test", http.StatusOK},
		{"POST", "/api/users", http.StatusCreated},
	}

	for _, req := range requests {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest(req.method, req.path, nil)
		router.ServeHTTP(w, r)
	}

	// Verify metrics
	metrics := logger.GetMetrics()
	assert.Equal(t, int64(3), metrics.TotalRequests)
	assert.Equal(t, int64(3), metrics.SuccessfulRequests)
	assert.Equal(t, int64(2), metrics.RequestsByMethod["GET"])
	assert.Equal(t, int64(1), metrics.RequestsByMethod["POST"])
	assert.Equal(t, int64(2), metrics.RequestsByPath["/test"])
	assert.Equal(t, int64(1), metrics.RequestsByPath["/api/users"])
}

func TestAccessLogging_ResetMetrics(t *testing.T) {
	logger, cleanup := setupTestAccessLogger(t)
	defer cleanup()

	router := setupTestRouter()
	router.Use(logger.AccessLogging())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	// Make a request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	// Verify metrics
	metrics := logger.GetMetrics()
	assert.Equal(t, int64(1), metrics.TotalRequests)

	// Reset metrics
	logger.ResetMetrics()

	// Verify metrics were reset
	metrics = logger.GetMetrics()
	assert.Equal(t, int64(0), metrics.TotalRequests)
	assert.Equal(t, int64(0), metrics.SuccessfulRequests)
}

func TestAccessLogging_FormatMetrics(t *testing.T) {
	logger, cleanup := setupTestAccessLogger(t)
	defer cleanup()

	router := setupTestRouter()
	router.Use(logger.AccessLogging())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	// Make a request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	// Get formatted metrics
	formatted := logger.FormatMetrics()
	assert.Contains(t, formatted, "Total Requests: 1")
	assert.Contains(t, formatted, "Successful: 1")
	assert.Contains(t, formatted, "GET: 1")
}

func TestAuditLog(t *testing.T) {
	logger, cleanup := setupTestAccessLogger(t)
	defer cleanup()

	router := setupTestRouter()
	userID := uuid.New()
	userEmail := "test@example.com"

	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("user_email", userEmail)
		c.Set("request_id", uuid.New().String())
		c.Next()
	})
	router.DELETE("/users/:id", func(c *gin.Context) {
		logger.AuditLog(c, "delete", "users", c.Param("id"), "success", "User deleted successfully")
		c.JSON(http.StatusOK, gin.H{"message": "deleted"})
	})

	w := httptest.NewRecorder()
	resourceID := uuid.New().String()
	req, _ := http.NewRequest("DELETE", "/users/"+resourceID, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Read audit log file
	auditContent, err := os.ReadFile(logger.config.AuditTrailPath)
	assert.NoError(t, err)

	// Parse JSON audit entry
	var auditEntry AuditLogEntry
	err = json.Unmarshal(auditContent, &auditEntry)
	assert.NoError(t, err)

	assert.Equal(t, userID.String(), auditEntry.UserID)
	assert.Equal(t, userEmail, auditEntry.UserEmail)
	assert.Equal(t, "delete", auditEntry.Action)
	assert.Equal(t, "users", auditEntry.Resource)
	assert.Equal(t, resourceID, auditEntry.ResourceID)
	assert.Equal(t, "success", auditEntry.Result)
}

func TestAccessLogging_JSONFormat(t *testing.T) {
	logger, cleanup := setupTestAccessLogger(t)
	defer cleanup()

	router := setupTestRouter()
	router.Use(logger.AccessLogging())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Read log file and verify it's valid JSON
	logContent, err := os.ReadFile(logger.config.LogFilePath)
	assert.NoError(t, err)

	var logEntry AccessLogEntry
	err = json.Unmarshal(logContent, &logEntry)
	assert.NoError(t, err)

	assert.Equal(t, "GET", logEntry.Method)
	assert.Equal(t, "/test", logEntry.Path)
	assert.Equal(t, 200, logEntry.StatusCode)
}

func TestAccessLogging_TextFormat(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.AccessLoggingConfig{
		Enabled:               true,
		LogSuccessfulRequests: true,
		LogFormat:             "text",
		IncludeUserInfo:       true,
		IncludeIPInfo:         true,
		LogToFile:             true,
		LogFilePath:           filepath.Join(tmpDir, "access.log"),
		LogToStdout:           false,
		EnableMetrics:         true,
		TrackResponseTime:     true,
		LogRotation: config.LogRotationConfig{
			Enabled: false,
		},
	}

	logger, err := NewAccessLogger(cfg)
	assert.NoError(t, err)

	router := setupTestRouter()
	router.Use(logger.AccessLogging())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Read log file and verify it's text format
	logContent, err := os.ReadFile(logger.config.LogFilePath)
	assert.NoError(t, err)

	logStr := string(logContent)
	assert.Contains(t, logStr, "GET")
	assert.Contains(t, logStr, "/test")
	assert.Contains(t, logStr, "200")
}

func TestAccessLogging_CombinedFormat(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.AccessLoggingConfig{
		Enabled:               true,
		LogSuccessfulRequests: true,
		LogFormat:             "combined",
		IncludeUserInfo:       true,
		IncludeIPInfo:         true,
		LogToFile:             true,
		LogFilePath:           filepath.Join(tmpDir, "access.log"),
		LogToStdout:           false,
		EnableMetrics:         true,
		TrackResponseTime:     true,
		TrackResponseSize:     true,
		LogRotation: config.LogRotationConfig{
			Enabled: false,
		},
	}

	logger, err := NewAccessLogger(cfg)
	assert.NoError(t, err)

	router := setupTestRouter()
	router.Use(logger.AccessLogging())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Read log file and verify it's Apache combined format
	logContent, err := os.ReadFile(logger.config.LogFilePath)
	assert.NoError(t, err)

	logStr := string(logContent)
	// Combined format should contain: IP - UserID [Date] "Method Path Protocol" Status Size "Referer" "UserAgent" Time
	assert.Contains(t, logStr, "GET /test HTTP/1.1")
	assert.Contains(t, logStr, "200")
}

func TestResponseWriter_Capture(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	buf := &bytes.Buffer{}
	rw := &responseWriter{
		ResponseWriter: c.Writer,
		body:           buf,
		statusCode:     200,
	}

	data := []byte("test response")
	n, err := rw.Write(data)

	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, len(data), rw.size)
	assert.Equal(t, string(data), buf.String())
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	rw := &responseWriter{
		ResponseWriter: c.Writer,
		statusCode:     200,
	}

	rw.WriteHeader(404)
	assert.Equal(t, 404, rw.statusCode)
}

func BenchmarkAccessLogging(b *testing.B) {
	tmpDir := b.TempDir()

	cfg := &config.AccessLoggingConfig{
		Enabled:               true,
		LogSuccessfulRequests: true,
		LogFormat:             "json",
		LogToFile:             true,
		LogFilePath:           filepath.Join(tmpDir, "access.log"),
		LogToStdout:           false,
		EnableMetrics:         true,
		TrackResponseTime:     true,
		LogRotation: config.LogRotationConfig{
			Enabled: false,
		},
	}

	logger, _ := NewAccessLogger(cfg)

	router := setupTestRouter()
	router.Use(logger.AccessLogging())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)
	}
}

func BenchmarkMetricsCollection(b *testing.B) {
	cfg := &config.AccessLoggingConfig{
		Enabled:               true,
		LogSuccessfulRequests: false, // Disable logging to focus on metrics
		EnableMetrics:         true,
		TrackResponseTime:     true,
		TrackRequestSize:      true,
		TrackResponseSize:     true,
		LogToFile:             false,
		LogToStdout:           false,
	}

	logger, _ := NewAccessLogger(cfg)

	router := setupTestRouter()
	router.Use(logger.AccessLogging())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)
	}
}

package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// ARAgingHandler handles A/R aging report endpoints
type ARAgingHandler struct {
	service services.ARAgingService
}

// NewARAgingHandler creates a new A/R aging handler
func NewARAgingHandler(service services.ARAgingService) *ARAgingHandler {
	return &ARAgingHandler{
		service: service,
	}
}

// GetARAgingReport handles GET /api/v1/reports/ar-aging
// @Summary Get A/R aging report
// @Description Generates an accounts receivable aging report showing outstanding invoices by age buckets
// @Tags reports
// @Accept json
// @Produce json
// @Param as_of_date query string false "As-of date for aging calculation (RFC3339 format)"
// @Param customer_id query string false "Filter by customer ID"
// @Param min_amount query number false "Minimum amount due filter"
// @Param include_paid query boolean false "Include paid invoices" default(false)
// @Param include_draft query boolean false "Include draft invoices" default(false)
// @Success 200 {object} models.ARAgingReportResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/reports/ar-aging [get]
func (h *ARAgingHandler) GetARAgingReport(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Tenant context required",
		})
		return
	}

	var query models.ARAgingQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid query parameters: " + err.Error(),
		})
		return
	}

	// Set cache control headers
	c.Header("Cache-Control", "private, max-age=300")

	report, err := h.service.GenerateReport(c.Request.Context(), tenantID, &query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to generate A/R aging report: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, report)
}

// GetBucketSummary handles GET /api/v1/reports/ar-aging/buckets
// @Summary Get A/R aging bucket summary
// @Description Returns summary of outstanding receivables grouped by aging buckets
// @Tags reports
// @Accept json
// @Produce json
// @Param as_of_date query string false "As-of date for aging calculation (RFC3339 format)"
// @Param customer_id query string false "Filter by customer ID"
// @Success 200 {array} models.ARAgingBucketSummary
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/reports/ar-aging/buckets [get]
func (h *ARAgingHandler) GetBucketSummary(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Tenant context required",
		})
		return
	}

	var query models.ARAgingQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid query parameters: " + err.Error(),
		})
		return
	}

	c.Header("Cache-Control", "private, max-age=300")

	buckets, err := h.service.GetBucketSummary(c.Request.Context(), tenantID, &query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get bucket summary: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": buckets,
	})
}

// GetInvoicesByBucket handles GET /api/v1/reports/ar-aging/buckets/:bucket/invoices
// @Summary Get invoices for a specific aging bucket
// @Description Returns list of invoices in a specific aging bucket for drill-down
// @Tags reports
// @Accept json
// @Produce json
// @Param bucket path string true "Aging bucket (current, 1-30, 31-60, 61-90, 90+)"
// @Param as_of_date query string false "As-of date for aging calculation (RFC3339 format)"
// @Param customer_id query string false "Filter by customer ID"
// @Param min_amount query number false "Minimum amount due filter"
// @Success 200 {array} models.ARAgingInvoice
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/reports/ar-aging/buckets/{bucket}/invoices [get]
func (h *ARAgingHandler) GetInvoicesByBucket(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Tenant context required",
		})
		return
	}

	bucketParam := c.Param("bucket")
	bucket := models.AgingBucket(bucketParam)

	// Validate bucket
	validBuckets := map[models.AgingBucket]bool{
		models.AgingBucketCurrent: true,
		models.AgingBucket1To30:   true,
		models.AgingBucket31To60:  true,
		models.AgingBucket61To90:  true,
		models.AgingBucket90Plus:  true,
	}
	if !validBuckets[bucket] {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid bucket: must be one of current, 1-30, 31-60, 61-90, 90+",
		})
		return
	}

	var query models.ARAgingQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid query parameters: " + err.Error(),
		})
		return
	}

	c.Header("Cache-Control", "private, max-age=300")

	invoices, err := h.service.GetInvoicesByBucket(c.Request.Context(), tenantID, bucket, &query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get invoices: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   invoices,
		"bucket": bucket,
		"count":  len(invoices),
	})
}

// GetCustomerSummary handles GET /api/v1/reports/ar-aging/customers
// @Summary Get A/R aging by customer
// @Description Returns A/R aging summary grouped by customer
// @Tags reports
// @Accept json
// @Produce json
// @Param as_of_date query string false "As-of date for aging calculation (RFC3339 format)"
// @Param min_amount query number false "Minimum amount due filter"
// @Success 200 {array} models.ARAgingCustomerSummary
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/reports/ar-aging/customers [get]
func (h *ARAgingHandler) GetCustomerSummary(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Tenant context required",
		})
		return
	}

	var query models.ARAgingQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid query parameters: " + err.Error(),
		})
		return
	}

	c.Header("Cache-Control", "private, max-age=300")

	customers, err := h.service.GetCustomerSummary(c.Request.Context(), tenantID, &query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get customer summary: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  customers,
		"count": len(customers),
	})
}

// ExportARAgingReport handles POST /api/v1/reports/ar-aging/export
// @Summary Export A/R aging report
// @Description Exports the A/R aging report in the requested format
// @Tags reports
// @Accept json
// @Produce json
// @Produce text/csv
// @Param request body models.ARAgingExportRequest true "Export request"
// @Success 200 {file} file "Exported file"
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/reports/ar-aging/export [post]
func (h *ARAgingHandler) ExportARAgingReport(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Tenant context required",
		})
		return
	}

	var req models.ARAgingExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// Generate the report data
	report, err := h.service.GenerateReport(c.Request.Context(), tenantID, &req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to generate report: " + err.Error(),
		})
		return
	}

	// Get invoices for CSV export
	var allInvoices []models.ARAgingInvoice
	for _, bucket := range report.Buckets {
		invoices, err := h.service.GetInvoicesByBucket(c.Request.Context(), tenantID, bucket.Bucket, &req.Query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get invoices: " + err.Error(),
			})
			return
		}
		allInvoices = append(allInvoices, invoices...)
	}

	// Handle CSV export
	if req.Format == models.ExportFormatCSV {
		filename := req.Filename
		if filename == "" {
			filename = "ar_aging_report_" + time.Now().Format("2006-01-02") + ".csv"
		}

		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename="+filename)

		// Write CSV header
		c.Writer.WriteString("Invoice Number,Customer,Company,Issue Date,Due Date,Total Amount,Amount Paid,Amount Due,Days Overdue,Bucket,Status\n")

		// Write CSV rows
		for _, inv := range allInvoices {
			companyName := ""
			if inv.CompanyName != nil {
				companyName = *inv.CompanyName
			}
			c.Writer.WriteString(
				inv.InvoiceNumber + "," +
					inv.CustomerName + "," +
					companyName + "," +
					inv.IssueDate.Format("2006-01-02") + "," +
					inv.DueDate.Format("2006-01-02") + "," +
					inv.TotalAmount.String() + "," +
					inv.AmountPaid.String() + "," +
					inv.AmountDue.String() + "," +
					string(rune(inv.DaysOverdue+'0')) + "," +
					models.GetBucketDisplayName(inv.Bucket) + "," +
					string(inv.Status) + "\n",
			)
		}
		return
	}

	// Default: return JSON
	report.Invoices = allInvoices
	c.JSON(http.StatusOK, report)
}

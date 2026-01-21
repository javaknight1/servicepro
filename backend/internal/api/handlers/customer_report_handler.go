package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// CustomerReportHandler handles customer report endpoints
type CustomerReportHandler struct {
	reportService services.CustomerReportService
}

// NewCustomerReportHandler creates a new customer report handler
func NewCustomerReportHandler(reportService services.CustomerReportService) *CustomerReportHandler {
	return &CustomerReportHandler{
		reportService: reportService,
	}
}

// GetCustomerReport handles GET /api/v1/reports/customers
// @Summary Get customer report
// @Description Generates a comprehensive customer report with segmentation, metrics, and trends
// @Tags reports
// @Accept json
// @Produce json
// @Param start_date query string false "Start date (RFC3339 format)"
// @Param end_date query string false "End date (RFC3339 format)"
// @Param segment query string false "Filter by segment (high_value, regular, occasional, at_risk, churned, new)"
// @Param customer_type query string false "Filter by type (residential, commercial)"
// @Param status query string false "Filter by status (active, inactive, prospect)"
// @Param state query string false "Filter by state"
// @Param sort_by query string false "Sort field" default(created_at)
// @Param sort_order query string false "Sort order (asc, desc)" default(desc)
// @Success 200 {object} models.CustomerReportResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/reports/customers [get]
func (h *CustomerReportHandler) GetCustomerReport(c *gin.Context) {
	var query models.CustomerReportQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid query parameters: " + err.Error(),
		})
		return
	}

	// Set cache control headers
	c.Header("Cache-Control", "private, max-age=300")

	startTime := time.Now()
	report, err := h.reportService.GenerateReport(c.Request.Context(), &query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to generate customer report: " + err.Error(),
		})
		return
	}

	report.Metadata.ExecutionTime = time.Since(startTime).Milliseconds()

	c.JSON(http.StatusOK, report)
}

// GetSegmentBreakdown handles GET /api/v1/reports/customers/segments
// @Summary Get customer segment breakdown
// @Description Returns customer breakdown by segment with metrics
// @Tags reports
// @Accept json
// @Produce json
// @Param customer_type query string false "Filter by type (residential, commercial)"
// @Param status query string false "Filter by status (active, inactive, prospect)"
// @Success 200 {array} models.SegmentBreakdownData
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/reports/customers/segments [get]
func (h *CustomerReportHandler) GetSegmentBreakdown(c *gin.Context) {
	var query models.CustomerReportQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid query parameters: " + err.Error(),
		})
		return
	}

	c.Header("Cache-Control", "private, max-age=300")

	segments, err := h.reportService.GetSegmentBreakdown(c.Request.Context(), &query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get segment breakdown: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": segments,
	})
}

// GetTypeBreakdown handles GET /api/v1/reports/customers/types
// @Summary Get customer type breakdown
// @Description Returns customer breakdown by type and status
// @Tags reports
// @Accept json
// @Produce json
// @Success 200 {array} models.TypeBreakdownData
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/reports/customers/types [get]
func (h *CustomerReportHandler) GetTypeBreakdown(c *gin.Context) {
	var query models.CustomerReportQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid query parameters: " + err.Error(),
		})
		return
	}

	c.Header("Cache-Control", "private, max-age=300")

	types, err := h.reportService.GetTypeBreakdown(c.Request.Context(), &query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get type breakdown: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": types,
	})
}

// GetGeographyBreakdown handles GET /api/v1/reports/customers/geography
// @Summary Get customer geographic distribution
// @Description Returns customer distribution by state and city
// @Tags reports
// @Accept json
// @Produce json
// @Success 200 {array} models.GeographyBreakdownData
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/reports/customers/geography [get]
func (h *CustomerReportHandler) GetGeographyBreakdown(c *gin.Context) {
	var query models.CustomerReportQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid query parameters: " + err.Error(),
		})
		return
	}

	c.Header("Cache-Control", "private, max-age=300")

	geography, err := h.reportService.GetGeographyBreakdown(c.Request.Context(), &query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get geography breakdown: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": geography,
	})
}

// GetAcquisitionTrend handles GET /api/v1/reports/customers/acquisition
// @Summary Get customer acquisition trend
// @Description Returns customer acquisition trend over time
// @Tags reports
// @Accept json
// @Produce json
// @Param start_date query string false "Start date (RFC3339 format)"
// @Param end_date query string false "End date (RFC3339 format)"
// @Success 200 {array} models.AcquisitionTrendData
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/reports/customers/acquisition [get]
func (h *CustomerReportHandler) GetAcquisitionTrend(c *gin.Context) {
	var query models.CustomerReportQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid query parameters: " + err.Error(),
		})
		return
	}

	c.Header("Cache-Control", "private, max-age=300")

	trend, err := h.reportService.GetAcquisitionTrend(c.Request.Context(), &query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get acquisition trend: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": trend,
	})
}

// GetTopCustomers handles GET /api/v1/reports/customers/top
// @Summary Get top customers by revenue
// @Description Returns top customers ranked by total revenue
// @Tags reports
// @Accept json
// @Produce json
// @Param limit query int false "Number of customers to return" default(10)
// @Param segment query string false "Filter by segment"
// @Param customer_type query string false "Filter by type"
// @Success 200 {array} models.CustomerMetricsData
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/reports/customers/top [get]
func (h *CustomerReportHandler) GetTopCustomers(c *gin.Context) {
	var query models.CustomerReportQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid query parameters: " + err.Error(),
		})
		return
	}

	limit := 10
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	c.Header("Cache-Control", "private, max-age=300")

	customers, err := h.reportService.GetTopCustomers(c.Request.Context(), &query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get top customers: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": customers,
	})
}

// GetAtRiskCustomers handles GET /api/v1/reports/customers/at-risk
// @Summary Get at-risk customers
// @Description Returns customers identified as at-risk of churning
// @Tags reports
// @Accept json
// @Produce json
// @Param limit query int false "Number of customers to return" default(10)
// @Success 200 {array} models.CustomerMetricsData
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/reports/customers/at-risk [get]
func (h *CustomerReportHandler) GetAtRiskCustomers(c *gin.Context) {
	limit := 10
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	c.Header("Cache-Control", "private, max-age=300")

	customers, err := h.reportService.GetAtRiskCustomers(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get at-risk customers: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": customers,
	})
}

// GetCohortData handles GET /api/v1/reports/customers/cohorts
// @Summary Get customer cohort retention data
// @Description Returns customer retention data by acquisition cohort
// @Tags reports
// @Accept json
// @Produce json
// @Param months query int false "Number of months to include" default(12)
// @Success 200 {array} models.CohortData
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/reports/customers/cohorts [get]
func (h *CustomerReportHandler) GetCohortData(c *gin.Context) {
	months := 12
	if m := c.Query("months"); m != "" {
		if parsed, err := strconv.Atoi(m); err == nil && parsed > 0 {
			months = parsed
		}
	}

	c.Header("Cache-Control", "private, max-age=300")

	cohorts, err := h.reportService.GetCohortData(c.Request.Context(), months)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get cohort data: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": cohorts,
	})
}

// RefreshMetricsCache handles POST /api/v1/reports/customers/cache/refresh
// @Summary Refresh customer metrics cache
// @Description Manually refreshes the customer metrics cache
// @Tags reports
// @Accept json
// @Produce json
// @Param customer_id query string false "Customer ID to refresh (omit for all)"
// @Success 200 {object} map[string]string
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/reports/customers/cache/refresh [post]
func (h *CustomerReportHandler) RefreshMetricsCache(c *gin.Context) {
	var customerID *uuid.UUID

	if idStr := c.Query("customer_id"); idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_id",
				Message: "Invalid customer ID format",
			})
			return
		}
		customerID = &id
	}

	if err := h.reportService.RefreshMetricsCache(c.Request.Context(), customerID); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "cache_refresh_failed",
			Message: "Failed to refresh metrics cache: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Metrics cache refreshed successfully",
	})
}

// ExportReport handles POST /api/v1/reports/customers/export
// @Summary Export customer report
// @Description Exports customer report in specified format (csv, json)
// @Tags reports
// @Accept json
// @Produce json
// @Param request body models.CustomerReportExportRequest true "Export request"
// @Success 200 {object} models.ReportExportResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/reports/customers/export [post]
func (h *CustomerReportHandler) ExportReport(c *gin.Context) {
	var req models.CustomerReportExportRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid export request: " + err.Error(),
		})
		return
	}

	// Validate format
	validFormats := map[models.ExportFormat]bool{
		models.ExportFormatCSV:  true,
		models.ExportFormatJSON: true,
	}

	if !validFormats[req.Format] {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_format",
			Message: "Invalid export format. Supported formats: csv, json",
		})
		return
	}

	export, err := h.reportService.ExportReport(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "export_failed",
			Message: "Failed to export report: " + err.Error(),
		})
		return
	}

	// For CSV format, return as downloadable file
	if req.Format == models.ExportFormatCSV {
		if csvData, ok := export.Data.(string); ok {
			c.Header("Content-Disposition", "attachment; filename="+export.FileName)
			c.Header("Content-Type", "text/csv")
			c.String(http.StatusOK, csvData)
			return
		}
	}

	c.JSON(http.StatusOK, export)
}

// GetChartData handles GET /api/v1/reports/customers/chart-data
// @Summary Get chart-formatted customer data
// @Description Returns customer data formatted for Chart.js consumption
// @Tags reports
// @Accept json
// @Produce json
// @Param chart_type query string false "Chart type (segments, types, geography, acquisition)" default(segments)
// @Success 200 {object} models.CustomerChartData
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/reports/customers/chart-data [get]
func (h *CustomerReportHandler) GetChartData(c *gin.Context) {
	var query models.CustomerReportQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid query parameters: " + err.Error(),
		})
		return
	}

	chartType := c.DefaultQuery("chart_type", "segments")

	c.Header("Cache-Control", "private, max-age=300")

	var chartData models.CustomerChartData

	switch chartType {
	case "segments":
		segments, err := h.reportService.GetSegmentBreakdown(c.Request.Context(), &query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get chart data: " + err.Error(),
			})
			return
		}
		chartData = formatSegmentChartData(segments)

	case "types":
		types, err := h.reportService.GetTypeBreakdown(c.Request.Context(), &query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get chart data: " + err.Error(),
			})
			return
		}
		chartData = formatTypeChartData(types)

	case "geography":
		geo, err := h.reportService.GetGeographyBreakdown(c.Request.Context(), &query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get chart data: " + err.Error(),
			})
			return
		}
		chartData = formatGeographyChartData(geo)

	case "acquisition":
		trend, err := h.reportService.GetAcquisitionTrend(c.Request.Context(), &query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get chart data: " + err.Error(),
			})
			return
		}
		chartData = formatAcquisitionChartData(trend)

	default:
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_chart_type",
			Message: "Invalid chart type. Supported: segments, types, geography, acquisition",
		})
		return
	}

	c.JSON(http.StatusOK, chartData)
}

// Helper functions for chart data formatting

func formatSegmentChartData(segments []models.SegmentBreakdownData) models.CustomerChartData {
	labels := make([]string, len(segments))
	data := make([]float64, len(segments))
	colors := make([]string, len(segments))

	for i, seg := range segments {
		labels[i] = seg.DisplayName
		data[i] = float64(seg.Count)
		colors[i] = seg.Color
	}

	return models.CustomerChartData{
		Labels: labels,
		Datasets: []models.CustomerChartDataset{
			{
				Label:           "Customers by Segment",
				Data:            data,
				BackgroundColor: colors,
			},
		},
		Options: map[string]any{
			"responsive": true,
			"plugins": map[string]any{
				"legend": map[string]any{
					"position": "right",
				},
				"title": map[string]any{
					"display": true,
					"text":    "Customer Segmentation",
				},
			},
		},
	}
}

func formatTypeChartData(types []models.TypeBreakdownData) models.CustomerChartData {
	labels := make([]string, len(types))
	data := make([]float64, len(types))

	for i, t := range types {
		labels[i] = string(t.CustomerType) + " - " + string(t.Status)
		data[i] = float64(t.Count)
	}

	return models.CustomerChartData{
		Labels: labels,
		Datasets: []models.CustomerChartDataset{
			{
				Label: "Customers by Type",
				Data:  data,
				BackgroundColor: []string{
					"#3B82F6", "#22C55E", "#F59E0B", "#EF4444", "#8B5CF6", "#EC4899",
				},
			},
		},
		Options: map[string]any{
			"responsive": true,
			"plugins": map[string]any{
				"title": map[string]any{
					"display": true,
					"text":    "Customer Type Distribution",
				},
			},
		},
	}
}

func formatGeographyChartData(geo []models.GeographyBreakdownData) models.CustomerChartData {
	// Limit to top 10 states
	limit := 10
	if len(geo) < limit {
		limit = len(geo)
	}

	labels := make([]string, limit)
	data := make([]float64, limit)

	for i := 0; i < limit; i++ {
		labels[i] = geo[i].State
		data[i] = float64(geo[i].CustomerCount)
	}

	return models.CustomerChartData{
		Labels: labels,
		Datasets: []models.CustomerChartDataset{
			{
				Label:       "Customers by State",
				Data:        data,
				BorderColor: "#3B82F6",
				Fill:        false,
			},
		},
		Options: map[string]any{
			"responsive": true,
			"plugins": map[string]any{
				"title": map[string]any{
					"display": true,
					"text":    "Geographic Distribution",
				},
			},
		},
	}
}

func formatAcquisitionChartData(trend []models.AcquisitionTrendData) models.CustomerChartData {
	labels := make([]string, len(trend))
	newData := make([]float64, len(trend))
	activeData := make([]float64, len(trend))

	for i, t := range trend {
		labels[i] = t.Period.Format("Jan 2006")
		newData[i] = float64(t.NewCustomers)
		activeData[i] = float64(t.ActiveCustomers)
	}

	return models.CustomerChartData{
		Labels: labels,
		Datasets: []models.CustomerChartDataset{
			{
				Label:       "New Customers",
				Data:        newData,
				BorderColor: "#3B82F6",
				Fill:        false,
			},
			{
				Label:       "Active Customers",
				Data:        activeData,
				BorderColor: "#22C55E",
				Fill:        false,
			},
		},
		Options: map[string]any{
			"responsive": true,
			"plugins": map[string]any{
				"title": map[string]any{
					"display": true,
					"text":    "Customer Acquisition Trend",
				},
			},
		},
	}
}

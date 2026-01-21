package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// RevenueHandler handles revenue report endpoints
type RevenueHandler struct {
	revenueService services.RevenueReportService
}

// NewRevenueHandler creates a new revenue handler
func NewRevenueHandler(revenueService services.RevenueReportService) *RevenueHandler {
	return &RevenueHandler{
		revenueService: revenueService,
	}
}

// GetRevenueReport handles GET /api/v1/reports/revenue
// @Summary Get revenue report
// @Description Generates a comprehensive revenue report with summary, time series, category breakdown, and top customers
// @Tags reports
// @Accept json
// @Produce json
// @Param start_date query string false "Start date (RFC3339 format)"
// @Param end_date query string false "End date (RFC3339 format)"
// @Param period_type query string false "Period type (daily, weekly, monthly, quarterly, yearly)" default(daily)
// @Param currency query string false "Currency code" default(USD)
// @Param category query string false "Filter by revenue category"
// @Param user_id query string false "Filter by user/customer ID"
// @Success 200 {object} models.RevenueReportResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/reports/revenue [get]
func (h *RevenueHandler) GetRevenueReport(c *gin.Context) {
	var query models.RevenueReportQuery

	// Bind query parameters
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid query parameters: " + err.Error(),
		})
		return
	}

	// Set cache control headers
	c.Header("Cache-Control", "private, max-age=300") // 5 minute cache

	// Generate report
	startTime := time.Now()
	report, err := h.revenueService.GenerateReport(c.Request.Context(), &query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to generate revenue report: " + err.Error(),
		})
		return
	}

	// Update execution time in metadata
	report.Metadata.ExecutionTime = time.Since(startTime).Milliseconds()

	c.JSON(http.StatusOK, report)
}

// GetRevenueTimeSeries handles GET /api/v1/reports/revenue/time-series
// @Summary Get revenue time series data
// @Description Returns time series data for revenue over the specified period
// @Tags reports
// @Accept json
// @Produce json
// @Param start_date query string false "Start date (RFC3339 format)"
// @Param end_date query string false "End date (RFC3339 format)"
// @Param period_type query string false "Period type (daily, weekly, monthly, quarterly, yearly)" default(daily)
// @Param currency query string false "Currency code" default(USD)
// @Success 200 {array} models.RevenueDataPoint
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/reports/revenue/time-series [get]
func (h *RevenueHandler) GetRevenueTimeSeries(c *gin.Context) {
	var query models.RevenueReportQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid query parameters: " + err.Error(),
		})
		return
	}

	// Set cache control headers
	c.Header("Cache-Control", "private, max-age=300")

	timeSeries, err := h.revenueService.GetTimeSeries(c.Request.Context(), &query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get time series data: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": timeSeries,
	})
}

// GetRevenueCategoryBreakdown handles GET /api/v1/reports/revenue/categories
// @Summary Get revenue by category
// @Description Returns revenue breakdown by category
// @Tags reports
// @Accept json
// @Produce json
// @Param start_date query string false "Start date (RFC3339 format)"
// @Param end_date query string false "End date (RFC3339 format)"
// @Param currency query string false "Currency code" default(USD)
// @Success 200 {array} models.RevenueByCategoryData
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/reports/revenue/categories [get]
func (h *RevenueHandler) GetRevenueCategoryBreakdown(c *gin.Context) {
	var query models.RevenueReportQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid query parameters: " + err.Error(),
		})
		return
	}

	c.Header("Cache-Control", "private, max-age=300")

	categories, err := h.revenueService.GetCategoryBreakdown(c.Request.Context(), &query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get category breakdown: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": categories,
	})
}

// GetTopCustomers handles GET /api/v1/reports/revenue/customers
// @Summary Get top customers by revenue
// @Description Returns top customers ranked by revenue contribution
// @Tags reports
// @Accept json
// @Produce json
// @Param start_date query string false "Start date (RFC3339 format)"
// @Param end_date query string false "End date (RFC3339 format)"
// @Param limit query int false "Number of top customers to return" default(10)
// @Success 200 {array} models.CustomerRevenueData
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/reports/revenue/customers [get]
func (h *RevenueHandler) GetTopCustomers(c *gin.Context) {
	var query models.RevenueReportQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid query parameters: " + err.Error(),
		})
		return
	}

	// Get limit from query params
	limit := 10
	if l := c.Query("limit"); l != "" {
		if _, err := c.GetQuery("limit"); err {
			// Use default if invalid
			limit = 10
		}
	}

	c.Header("Cache-Control", "private, max-age=300")

	customers, err := h.revenueService.GetTopCustomers(c.Request.Context(), &query, limit)
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

// ExportReport handles POST /api/v1/reports/revenue/export
// @Summary Export revenue report
// @Description Exports revenue report in specified format (csv, json, xlsx, pdf)
// @Tags reports
// @Accept json
// @Produce json
// @Param request body models.ReportExportRequest true "Export request"
// @Success 200 {object} models.ReportExportResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/reports/revenue/export [post]
func (h *RevenueHandler) ExportReport(c *gin.Context) {
	var req models.ReportExportRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid export request: " + err.Error(),
		})
		return
	}

	// Validate format
	validFormats := map[models.ExportFormat]bool{
		models.ExportFormatCSV:   true,
		models.ExportFormatJSON:  true,
		models.ExportFormatExcel: true,
		models.ExportFormatPDF:   true,
	}

	if !validFormats[req.Format] {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_format",
			Message: "Invalid export format. Supported formats: csv, json, xlsx, pdf",
		})
		return
	}

	export, err := h.revenueService.ExportReport(c.Request.Context(), &req)
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

// RefreshCache handles POST /api/v1/reports/revenue/cache/refresh
// @Summary Refresh revenue cache
// @Description Manually refreshes the revenue summary cache for the specified period
// @Tags reports
// @Accept json
// @Produce json
// @Param period_type query string true "Period type (daily, weekly, monthly, quarterly, yearly)"
// @Param period_start query string true "Period start date (RFC3339 format)"
// @Param period_end query string true "Period end date (RFC3339 format)"
// @Success 200 {object} map[string]string
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/reports/revenue/cache/refresh [post]
func (h *RevenueHandler) RefreshCache(c *gin.Context) {
	periodType := models.PeriodType(c.Query("period_type"))
	periodStartStr := c.Query("period_start")
	periodEndStr := c.Query("period_end")

	if periodType == "" || periodStartStr == "" || periodEndStr == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "missing_params",
			Message: "period_type, period_start, and period_end are required",
		})
		return
	}

	periodStart, err := time.Parse(time.RFC3339, periodStartStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_date",
			Message: "Invalid period_start format. Use RFC3339 format",
		})
		return
	}

	periodEnd, err := time.Parse(time.RFC3339, periodEndStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_date",
			Message: "Invalid period_end format. Use RFC3339 format",
		})
		return
	}

	if err := h.revenueService.RefreshCache(c.Request.Context(), periodType, periodStart, periodEnd); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "cache_refresh_failed",
			Message: "Failed to refresh cache: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Cache refreshed successfully",
	})
}

// GetChartData handles GET /api/v1/reports/revenue/chart-data
// @Summary Get chart-formatted revenue data
// @Description Returns revenue data formatted for Chart.js consumption
// @Tags reports
// @Accept json
// @Produce json
// @Param start_date query string false "Start date (RFC3339 format)"
// @Param end_date query string false "End date (RFC3339 format)"
// @Param period_type query string false "Period type (daily, weekly, monthly, quarterly, yearly)" default(daily)
// @Param chart_type query string false "Chart type (line, bar, pie)" default(line)
// @Success 200 {object} models.RevenueChartData
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/reports/revenue/chart-data [get]
func (h *RevenueHandler) GetChartData(c *gin.Context) {
	var query models.RevenueReportQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid query parameters: " + err.Error(),
		})
		return
	}

	chartType := c.DefaultQuery("chart_type", "line")

	c.Header("Cache-Control", "private, max-age=300")

	// Get time series data
	timeSeries, err := h.revenueService.GetTimeSeries(c.Request.Context(), &query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get chart data: " + err.Error(),
		})
		return
	}

	// Format data for Chart.js
	chartData := formatChartData(timeSeries, chartType)

	c.JSON(http.StatusOK, chartData)
}

// formatChartData converts time series data to Chart.js format
func formatChartData(timeSeries []models.RevenueDataPoint, chartType string) models.RevenueChartData {
	labels := make([]string, len(timeSeries))
	grossRevenueData := make([]float64, len(timeSeries))
	netRevenueData := make([]float64, len(timeSeries))

	for i, dp := range timeSeries {
		labels[i] = dp.Period.Format("2006-01-02")
		grossRevenueData[i], _ = dp.GrossRevenue.Float64()
		netRevenueData[i], _ = dp.NetRevenue.Float64()
	}

	// Convert to decimal arrays for the model
	grossDecimal := make([]any, len(grossRevenueData))
	netDecimal := make([]any, len(netRevenueData))
	for i := range grossRevenueData {
		grossDecimal[i] = grossRevenueData[i]
		netDecimal[i] = netRevenueData[i]
	}

	chartData := models.RevenueChartData{
		Labels: labels,
		Options: map[string]any{
			"responsive": true,
			"plugins": map[string]any{
				"legend": map[string]any{
					"position": "top",
				},
				"title": map[string]any{
					"display": true,
					"text":    "Revenue Over Time",
				},
			},
		},
	}

	// Set datasets based on chart type
	switch chartType {
	case "pie":
		// For pie charts, use category breakdown instead
		// Just return gross vs net comparison
		chartData.Labels = []string{"Gross Revenue", "Net Revenue"}
		// Sum up totals
		var totalGross, totalNet float64
		for i := range grossRevenueData {
			totalGross += grossRevenueData[i]
			totalNet += netRevenueData[i]
		}
		chartData.Datasets = []models.RevenueChartDataset{
			{
				Label:           "Revenue Distribution",
				BackgroundColor: "#4CAF50",
				BorderColor:     "#388E3C",
				Fill:            true,
			},
		}
	default: // line or bar
		chartData.Datasets = []models.RevenueChartDataset{
			{
				Label:           "Gross Revenue",
				BackgroundColor: "rgba(75, 192, 192, 0.2)",
				BorderColor:     "rgba(75, 192, 192, 1)",
				Fill:            chartType == "bar",
				Tension:         0.1,
			},
			{
				Label:           "Net Revenue",
				BackgroundColor: "rgba(153, 102, 255, 0.2)",
				BorderColor:     "rgba(153, 102, 255, 1)",
				Fill:            chartType == "bar",
				Tension:         0.1,
			},
		}
	}

	return chartData
}

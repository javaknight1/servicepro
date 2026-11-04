package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services/quickbooks"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// syncNowBudget bounds "Sync now" (the request must finish well inside the
// 60 s Cloud Run timeout); the rest waits for the hourly job.
const syncNowBudget = 25 * time.Second

// QuickBooksHandler serves Settings → Integrations → QuickBooks (T192).
type QuickBooksHandler struct {
	svc         *quickbooks.Service
	frontendURL string
}

// NewQuickBooksHandler creates the handler. frontendURL is where the OAuth
// callback sends the browser back to.
func NewQuickBooksHandler(svc *quickbooks.Service, frontendURL string) *QuickBooksHandler {
	return &QuickBooksHandler{svc: svc, frontendURL: strings.TrimRight(frontendURL, "/")}
}

func (h *QuickBooksHandler) tenantID(c *gin.Context) (uuid.UUID, bool) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		apierror.Write(c, http.StatusUnauthorized, "unauthorized", "User not authenticated")
		return uuid.Nil, false
	}
	return tenantID, true
}

// Status returns the tenant's QuickBooks connection, counts and errors.
// @Summary		Get QuickBooks status
// @Description	Whether QuickBooks is available (configured=false: the server has no Intuit app yet, show "Coming soon"), the connection (connected, reauth_required, disconnected; company, environment, since when invoices sync), the mapping settings, synced counts, records that failed (with their error) and recent activity. Requires integrations.manage (owners and admins).
// @Tags			Integrations
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Tenant ID"
// @Success		200	{object}	models.QuickBooksStatusResponse
// @Failure		401	{object}	models.ErrorResponse
// @Failure		403	{object}	models.ErrorResponse
// @Failure		500	{object}	models.ErrorResponse
// @Router			/tenants/{id}/integrations/quickbooks [get]
func (h *QuickBooksHandler) Status(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	st, err := h.svc.Status(c.Request.Context(), tenantID)
	if err != nil {
		apierror.Respond(c, err, "Failed to load QuickBooks status")
		return
	}
	c.JSON(http.StatusOK, st)
}

// Connect starts the Intuit OAuth flow.
// @Summary		Connect QuickBooks
// @Description	Returns the Intuit consent URL to open in the browser. The link is single use and expires after 10 minutes; Intuit sends the browser back to /integrations/quickbooks/callback, which returns it to Settings → Integrations. Connecting a different company than before starts the sync over from today. Requires integrations.manage.
// @Tags			Integrations
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Tenant ID"
// @Success		200	{object}	models.QuickBooksConnectResponse
// @Failure		401	{object}	models.ErrorResponse
// @Failure		403	{object}	models.ErrorResponse
// @Failure		503	{object}	models.ErrorResponse	"QuickBooks not configured on the server"
// @Router			/tenants/{id}/integrations/quickbooks/connect [post]
func (h *QuickBooksHandler) Connect(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	userID, err := middleware.GetUserID(c)
	if err != nil {
		apierror.Write(c, http.StatusUnauthorized, "unauthorized", "User not authenticated")
		return
	}
	u, err := h.svc.StartConnect(c.Request.Context(), tenantID, userID)
	if err != nil {
		apierror.Respond(c, err, "Failed to start QuickBooks connect")
		return
	}
	c.JSON(http.StatusOK, models.QuickBooksConnectResponse{URL: u})
}

// Callback receives Intuit's redirect after consent.
// @Summary		QuickBooks OAuth callback
// @Description	Intuit redirects the browser here (the redirect URI registered on the Intuit app) with code, state and realmId, or with error when the user cancels. The signed, single-use state identifies the tenant and the user who started the connect; no session is needed. Always answers with a redirect to Settings → Integrations (?quickbooks=connected, or ?quickbooks=error&reason=...).
// @Tags			Integrations
// @Param			code	query	string	false	"Authorization code"
// @Param			state	query	string	true	"State from the connect URL"
// @Param			realmId	query	string	false	"QuickBooks company ID"
// @Param			error	query	string	false	"Set when the user declined"
// @Success		302
// @Router			/integrations/quickbooks/callback [get]
func (h *QuickBooksHandler) Callback(c *gin.Context) {
	ctx := c.Request.Context()
	if e := c.Query("error"); e != "" {
		reason := "failed"
		if e == "access_denied" {
			reason = "cancelled"
		}
		h.redirect(c, reason)
		return
	}
	_, err := h.svc.CompleteConnect(ctx, c.Query("code"), c.Query("state"), c.Query("realmId"))
	if err != nil {
		reason := "failed"
		var de *apperrors.Error
		if errors.As(err, &de) {
			switch {
			case errors.Is(err, quickbooks.ErrInvalidState):
				reason = "expired"
			case errors.Is(err, quickbooks.ErrNotConfigured):
				reason = "unavailable"
			}
		} else {
			logging.Error(ctx, "[QUICKBOOKS] Callback failed", map[string]any{"error": err.Error()})
		}
		h.redirect(c, reason)
		return
	}
	h.redirect(c, "")
}

func (h *QuickBooksHandler) redirect(c *gin.Context, reason string) {
	q := url.Values{}
	if reason == "" {
		q.Set("quickbooks", "connected")
	} else {
		q.Set("quickbooks", "error")
		q.Set("reason", reason)
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer") // the URL carried a code
	c.Redirect(http.StatusFound, h.frontendURL+"/settings/integrations?"+q.Encode())
}

// Disconnect revokes the QuickBooks connection.
// @Summary		Disconnect QuickBooks
// @Description	Revokes the tokens at Intuit and forgets them. Nothing is deleted from QuickBooks; reconnecting the same company carries on where the sync stopped. Requires integrations.manage.
// @Tags			Integrations
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Tenant ID"
// @Success		200	{object}	models.QuickBooksStatusResponse
// @Failure		401	{object}	models.ErrorResponse
// @Failure		403	{object}	models.ErrorResponse
// @Failure		409	{object}	models.ErrorResponse	"Not connected"
// @Failure		503	{object}	models.ErrorResponse
// @Router			/tenants/{id}/integrations/quickbooks/disconnect [post]
func (h *QuickBooksHandler) Disconnect(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	if err := h.svc.Disconnect(c.Request.Context(), tenantID); err != nil {
		apierror.Respond(c, err, "Failed to disconnect QuickBooks")
		return
	}
	h.Status(c)
}

// Options lists the company's accounts and tax codes for the mapping.
// @Summary		List QuickBooks mapping options
// @Description	Income accounts, tax codes and deposit accounts (bank and other current asset) of the connected QuickBooks company. Requires integrations.manage.
// @Tags			Integrations
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Tenant ID"
// @Success		200	{object}	models.QuickBooksOptionsResponse
// @Failure		401	{object}	models.ErrorResponse
// @Failure		403	{object}	models.ErrorResponse
// @Failure		409	{object}	models.ErrorResponse	"Not connected, or needs reconnecting"
// @Failure		502	{object}	models.ErrorResponse	"QuickBooks didn't answer"
// @Router			/tenants/{id}/integrations/quickbooks/options [get]
func (h *QuickBooksHandler) Options(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	opts, err := h.svc.Options(c.Request.Context(), tenantID)
	if err != nil {
		apierror.Respond(c, err, "Failed to load QuickBooks options")
		return
	}
	c.JSON(http.StatusOK, opts)
}

// UpdateSettings saves the mapping settings.
// @Summary		Update QuickBooks mapping
// @Description	Sets the income account invoice lines post to (ServicePro finds or creates a service item "ServicePro - <account>" on it; without one, the company's "Services" item is used), an optional sales tax code (companies without automated sales tax) and an optional deposit account for payments (default: Undeposited Funds). Empty tax code / deposit account clear them. Records that failed are retried. Requires integrations.manage.
// @Tags			Integrations
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string								true	"Tenant ID"
// @Param			request	body		models.QuickBooksSettingsRequest	true	"Mapping"
// @Success		200		{object}	models.QuickBooksStatusResponse
// @Failure		400		{object}	models.ErrorResponse	"Not an account or tax code of the company"
// @Failure		401		{object}	models.ErrorResponse
// @Failure		403		{object}	models.ErrorResponse
// @Failure		409		{object}	models.ErrorResponse
// @Failure		502		{object}	models.ErrorResponse
// @Router			/tenants/{id}/integrations/quickbooks/settings [put]
func (h *QuickBooksHandler) UpdateSettings(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	var req models.QuickBooksSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}
	st, err := h.svc.UpdateSettings(c.Request.Context(), tenantID, req)
	if err != nil {
		apierror.Respond(c, err, "Failed to save QuickBooks settings")
		return
	}
	c.JSON(http.StatusOK, st)
}

// SyncNow pushes pending records right away.
// @Summary		Sync QuickBooks now
// @Description	Finds new and changed customers, invoices (issued since the connection) and payments and pushes them to QuickBooks for up to 25 seconds; anything left goes with the hourly sync. Safe to repeat: records already in QuickBooks are never created twice. Requires integrations.manage.
// @Tags			Integrations
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Tenant ID"
// @Success		200	{object}	models.QuickBooksSyncResponse
// @Failure		401	{object}	models.ErrorResponse
// @Failure		403	{object}	models.ErrorResponse
// @Failure		409	{object}	models.ErrorResponse	"Not connected, or needs reconnecting"
// @Failure		503	{object}	models.ErrorResponse
// @Router			/tenants/{id}/integrations/quickbooks/sync [post]
func (h *QuickBooksHandler) SyncNow(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	r, err := h.svc.SyncNow(c.Request.Context(), tenantID, syncNowBudget)
	if err != nil {
		apierror.Respond(c, err, "QuickBooks sync failed")
		return
	}
	c.JSON(http.StatusOK, models.QuickBooksSyncResponse{Enqueued: r.Enqueued, Synced: r.Synced, Failed: r.Failed, Skipped: r.Skipped, More: r.More})
}

// Retry re-queues one failed record.
// @Summary		Retry a QuickBooks record
// @Description	Puts a record that failed to sync back in the queue (it goes with the next sync, or "Sync now"). Requires integrations.manage.
// @Tags			Integrations
// @Produce		json
// @Security		BearerAuth
// @Param			id			path		string	true	"Tenant ID"
// @Param			record_id	path		string	true	"Sync record ID"
// @Success		200			{object}	models.QuickBooksStatusResponse
// @Failure		400			{object}	models.ErrorResponse
// @Failure		401			{object}	models.ErrorResponse
// @Failure		403			{object}	models.ErrorResponse
// @Failure		404			{object}	models.ErrorResponse
// @Router			/tenants/{id}/integrations/quickbooks/records/{record_id}/retry [post]
func (h *QuickBooksHandler) Retry(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	recordID, err := uuid.Parse(c.Param("record_id"))
	if err != nil {
		apierror.Write(c, http.StatusBadRequest, apperrors.CodeBadRequest, "Invalid record ID")
		return
	}
	if err := h.svc.Retry(c.Request.Context(), tenantID, recordID); err != nil {
		apierror.Respond(c, err, "Failed to retry record")
		return
	}
	h.Status(c)
}

// Limits for one QuickBooks sync pass: Cloud Run ends a request after 60 s.
const quickBooksJobBudget = 40 * time.Second

// QuickBooksSyncRunner runs one sync pass (quickbooks.Service.ProcessPending).
type QuickBooksSyncRunner func(ctx context.Context, budget time.Duration) (quickbooks.RunResult, error)

// QuickBooksSyncJobResponse is returned by POST /internal/jobs/quickbooks-sync.
type QuickBooksSyncJobResponse struct {
	Job        string `json:"job" example:"quickbooks-sync"`
	DurationMs int64  `json:"duration_ms" example:"812"`
	quickbooks.RunResult
}

// RunQuickBooksSync returns the QuickBooks sync job handler. It is mounted
// on the /internal/jobs group, behind InternalJobsHandler.Authorize.
// @Summary		Run QuickBooks sync job
// @Description	Internal job endpoint (auth: X-Internal-Jobs-Token header = INTERNAL_JOBS_SECRET; 404 when the secret is unset). For every tenant with QuickBooks connected (least recently synced first): finds new and changed customers, invoices and payments and pushes them to the tenant's own QuickBooks company, for up to 40 s per call (more=true: work left for the next call). Idempotent: one worker per tenant at a time (a lease on the connection), every record mapped once, creates carry an Intuit requestid that is reused on retry. Failed records back off exponentially and stay listed in Settings → Integrations. Mounted at the server root (/internal/jobs/quickbooks-sync), outside /api. A no-op when QuickBooks isn't configured.
// @Tags			Internal
// @Produce		json
// @Param			X-Internal-Jobs-Token	header		string	true	"Internal jobs secret"
// @Success		200						{object}	QuickBooksSyncJobResponse
// @Failure		401						{object}	models.ErrorResponse
// @Failure		404						{object}	models.ErrorResponse
// @Failure		500						{object}	models.ErrorResponse
// @Router			/internal/jobs/quickbooks-sync [post]
func RunQuickBooksSync(run QuickBooksSyncRunner) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		result, err := run(c.Request.Context(), quickBooksJobBudget)
		if err != nil {
			logging.Error(c.Request.Context(), "[INTERNAL-JOBS] quickbooks-sync failed", map[string]any{"error": err.Error()})
			apierror.WriteResponse(c, http.StatusInternalServerError, models.ErrorResponse{Error: "QuickBooks sync job failed"})
			return
		}
		c.JSON(http.StatusOK, QuickBooksSyncJobResponse{Job: "quickbooks-sync", DurationMs: time.Since(start).Milliseconds(), RunResult: result})
	}
}

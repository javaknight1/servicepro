package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
)

// T187: PO required, default PO, not-to-exceed limit + approvals.

func setCustomerRules(t *testing.T, f *isolationFixture, ctx context.Context, id uuid.UUID, rules map[string]any) {
	t.Helper()
	require.NoError(t, f.db.WithContext(ctx).Model(&models.Customer{}).Where("id = ?", id).Updates(rules).Error)
}

func draftInvoice(t *testing.T, svc *InvoiceService, f *isolationFixture, po string) *models.Invoice {
	t.Helper()
	now := time.Now()
	inv, err := svc.CreateInvoice(f.ctxA, &models.Invoice{
		CustomerID: f.custA.ID, IssueDate: now, DueDate: now.Add(24 * time.Hour), Lines: oneLine(100), PONumber: po,
	}, f.userA)
	require.NoError(t, err)
	return inv
}

func TestSendInvoice_PORequired(t *testing.T) {
	f := setupIsolation(t)
	svc := NewInvoiceService(f.db)
	setCustomerRules(t, f, f.ctxA, f.custA.ID, map[string]any{"po_required": true})

	inv := draftInvoice(t, svc, f, "")
	_, err := svc.SendInvoice(f.ctxA, inv.ID)
	require.ErrorIs(t, err, ErrPORequired)
	got, err := svc.GetInvoice(f.ctxA, inv.ID)
	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusDraft, got.Status, "still a draft")

	withPO := draftInvoice(t, svc, f, "7731-HV")
	sent, err := svc.SendInvoice(f.ctxA, withPO.ID)
	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusSent, sent.Status)

	// Customers without the rule are unaffected.
	setCustomerRules(t, f, f.ctxA, f.custA.ID, map[string]any{"po_required": false})
	noPO := draftInvoice(t, svc, f, "")
	_, err = svc.SendInvoice(f.ctxA, noPO.ID)
	require.NoError(t, err)
}

func TestCreateInvoice_UsesCustomerDefaultPO(t *testing.T) {
	f := setupIsolation(t)
	svc := NewInvoiceService(f.db)
	setCustomerRules(t, f, f.ctxA, f.custA.ID, map[string]any{"po_required": true, "default_po_number": "7731-HV"})

	inv := draftInvoice(t, svc, f, "")
	assert.Equal(t, "7731-HV", inv.PONumber)
	_, err := svc.SendInvoice(f.ctxA, inv.ID)
	require.NoError(t, err, "the standing PO satisfies the rule")

	own := draftInvoice(t, svc, f, "OTHER-1")
	assert.Equal(t, "OTHER-1", own.PONumber, "an explicit PO wins")
}

func setupNTE(t *testing.T) (*isolationFixture, *JobService, *JobApprovalService) {
	t.Helper()
	f := setupIsolation(t)
	createSQLiteTableForModel(t, f.db, &models.JobApproval{}, "")
	f.addMember(t, f.ctxA, f.userA, true)
	setCustomerRules(t, f, f.ctxA, f.custA.ID, map[string]any{
		"nte_limit": decimal.NewFromInt(500), "billing_approver_name": "Karen Doyle",
		"billing_approver_email": "karen.doyle@example.com",
	})
	return f, realJobService(f), NewJobApprovalService(f.db, repository.NewJobApprovalRepository(f.db))
}

func newExtraWork(t *testing.T, svc *JobService, f *isolationFixture, cost float64) uuid.UUID {
	t.Helper()
	job, err := svc.CreateJob(&models.CreateJobRequest{
		CustomerID: f.custA.ID, Title: "Carpet shampoo", JobType: models.JobTypeRepair, EstimatedCost: cost,
	}, f.userA)
	require.NoError(t, err)
	return job.ID
}

func TestSchedule_ExtraWorkOverNTENeedsApproval(t *testing.T) {
	f, jobs, approvals := setupNTE(t)
	jobID := newExtraWork(t, jobs, f, 750)

	_, err := jobs.TransitionStatus(jobID, models.JobStatusScheduled, "", nil, f.userA, models.UserRoleAdmin)
	require.ErrorIs(t, err, ErrApprovalRequired)

	// An approval for less than the estimate doesn't cover it.
	small := decimal.NewFromInt(600)
	_, err = approvals.Record(f.ctxA, jobID, &models.RecordJobApprovalRequest{Amount: &small}, f.userA)
	require.NoError(t, err)
	_, err = jobs.TransitionStatus(jobID, models.JobStatusScheduled, "", nil, f.userA, models.UserRoleAdmin)
	require.ErrorIs(t, err, ErrApprovalRequired)

	// Defaults: the customer's approver and the job's estimated cost.
	a, err := approvals.Record(f.ctxA, jobID, &models.RecordJobApprovalRequest{}, f.userA)
	require.NoError(t, err)
	assert.Equal(t, "karen.doyle@example.com", a.ApproverEmail)
	require.NotNil(t, a.ApproverName)
	assert.Equal(t, "Karen Doyle", *a.ApproverName)
	assert.True(t, decimal.NewFromInt(750).Equal(a.Amount), "amount %s", a.Amount)

	resp, err := jobs.TransitionStatus(jobID, models.JobStatusScheduled, "", nil, f.userA, models.UserRoleAdmin)
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusScheduled, resp.Status)

	list, err := approvals.List(f.ctxA, jobID)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestSchedule_WorkWithinNTEOrWithoutLimitNeedsNoApproval(t *testing.T) {
	f, jobs, _ := setupNTE(t)

	within := newExtraWork(t, jobs, f, 500) // equal to the limit is allowed
	_, err := jobs.TransitionStatus(within, models.JobStatusScheduled, "", nil, f.userA, models.UserRoleAdmin)
	require.NoError(t, err)

	setCustomerRules(t, f, f.ctxA, f.custA.ID, map[string]any{"nte_limit": nil})
	big := newExtraWork(t, jobs, f, 10000)
	_, err = jobs.TransitionStatus(big, models.JobStatusScheduled, "", nil, f.userA, models.UserRoleAdmin)
	require.NoError(t, err)
}

func TestJobApprovals_CrossTenant(t *testing.T) {
	f, jobs, approvals := setupNTE(t)
	jobID := newExtraWork(t, jobs, f, 750)
	email := "someone@b.test"

	_, err := approvals.Record(f.ctxB, jobID, &models.RecordJobApprovalRequest{ApproverEmail: &email}, f.userA)
	require.ErrorIs(t, err, ErrJobNotFound)
	_, err = approvals.List(f.ctxB, jobID)
	require.ErrorIs(t, err, ErrJobNotFound)

	// Still unapproved for tenant A.
	_, err = jobs.TransitionStatus(jobID, models.JobStatusScheduled, "", nil, f.userA, models.UserRoleAdmin)
	require.ErrorIs(t, err, ErrApprovalRequired)
}

func TestJobApprovals_NeedApproverEmail(t *testing.T) {
	f, jobs, approvals := setupNTE(t)
	setCustomerRules(t, f, f.ctxA, f.custA.ID, map[string]any{"billing_approver_email": nil})
	jobID := newExtraWork(t, jobs, f, 750)

	_, err := approvals.Record(f.ctxA, jobID, &models.RecordJobApprovalRequest{}, f.userA)
	require.ErrorIs(t, err, ErrApproverEmailRequired)
	neg := decimal.NewFromInt(-1)
	email := "pm@example.com"
	_, err = approvals.Record(f.ctxA, jobID, &models.RecordJobApprovalRequest{ApproverEmail: &email, Amount: &neg}, f.userA)
	require.ErrorIs(t, err, ErrInvalidApprovalAmount)
}

func TestBillingRulesRequest_Apply(t *testing.T) {
	var b models.BillingRules
	yes := true
	po, name, email := " 7731-HV ", "Karen", "karen@example.com"
	nte := decimal.RequireFromString("500.005")
	require.NoError(t, (&models.BillingRulesRequest{
		PORequired: &yes, DefaultPONumber: &po, BillingApproverName: &name, BillingApproverEmail: &email,
		NTELimit: &nte, ConsolidatedBilling: &yes,
	}).Apply(&b))
	assert.True(t, b.PORequired)
	assert.True(t, b.ConsolidatedBilling)
	require.NotNil(t, b.DefaultPONumber)
	assert.Equal(t, "7731-HV", *b.DefaultPONumber)
	require.NotNil(t, b.NTELimit)
	assert.Equal(t, "500.01", b.NTELimit.StringFixed(2))

	empty := ""
	require.NoError(t, (&models.BillingRulesRequest{DefaultPONumber: &empty, ClearNTELimit: true}).Apply(&b))
	assert.Nil(t, b.DefaultPONumber, "empty string clears")
	assert.Nil(t, b.NTELimit)

	bad := "not-an-email"
	negative := decimal.NewFromInt(-5)
	err := (&models.BillingRulesRequest{BillingApproverEmail: &bad, NTELimit: &negative}).Apply(&b)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid billing rules")
}

package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// Jobs follow their invoice: completed -> invoiced when invoiced, -> paid
// when the invoice is paid, back to completed if the invoice is cancelled
// (audit 7.M5: jobs never moved to invoiced or paid).
func TestInvoiceMovesJobThroughInvoicedAndPaid(t *testing.T) {
	f := setupIsolation(t)
	svc := NewInvoiceService(f.db)
	jobStatus := func(id uuid.UUID) models.JobStatus {
		var j models.Job
		require.NoError(t, f.db.WithContext(f.ctxA).First(&j, "id = ?", id).Error)
		return j.Status
	}

	job := seedJob(t, f, models.JobStatusCompleted)
	inv, err := svc.CreateInvoice(f.ctxA, invoiceFor(f, &job.ID, nil), f.userA)
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusInvoiced, jobStatus(job.ID))

	_, err = svc.SendInvoice(f.ctxA, inv.ID)
	require.NoError(t, err)
	_, err = svc.RecordPayment(f.ctxA, &models.InvoicePayment{InvoiceID: inv.ID, Amount: dec("40"), CreatedBy: &f.userA})
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusInvoiced, jobStatus(job.ID), "partial payment leaves the job invoiced")
	_, err = svc.RecordPayment(f.ctxA, &models.InvoicePayment{InvoiceID: inv.ID, Amount: dec("60"), CreatedBy: &f.userA})
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusPaid, jobStatus(job.ID))

	var history []models.JobStatusTransition
	require.NoError(t, f.db.Where("job_id = ?", job.ID).Find(&history).Error)
	assert.Len(t, history, 2)

	other := seedJob(t, f, models.JobStatusCompleted)
	inv2, err := svc.CreateInvoice(f.ctxA, invoiceFor(f, &other.ID, nil), f.userA)
	require.NoError(t, err)
	_, err = svc.CancelInvoice(f.ctxA, inv2.ID, "wrong job")
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusCompleted, jobStatus(other.ID))
}

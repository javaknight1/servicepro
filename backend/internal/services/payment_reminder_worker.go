package services

import (
	"context"
	"sync"
	"time"

	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// PaymentReminderWorker runs payment reminder processing on a schedule
type PaymentReminderWorker struct {
	service  *PaymentReminderService
	interval time.Duration
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewPaymentReminderWorker creates a new payment reminder worker
func NewPaymentReminderWorker(service *PaymentReminderService, interval time.Duration) *PaymentReminderWorker {
	return &PaymentReminderWorker{
		service:  service,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// Start begins the background worker
func (w *PaymentReminderWorker) Start(ctx context.Context) {
	w.wg.Add(1)
	go w.run(ctx)
	logging.Info(ctx, "[PAYMENT-REMINDER-WORKER] Started", map[string]any{"interval": w.interval.String()})
}

func (w *PaymentReminderWorker) run(ctx context.Context) {
	defer w.wg.Done()
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Run immediately on start
	w.service.ProcessAllTenants(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopChan:
			return
		case <-ticker.C:
			w.service.ProcessAllTenants(ctx)
		}
	}
}

// Stop gracefully stops the background worker
func (w *PaymentReminderWorker) Stop() {
	close(w.stopChan)
	w.wg.Wait()
}

package stripe

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v76"
)

type testLogger struct{}

func (l *testLogger) Debugf(format string, v ...interface{}) {}
func (l *testLogger) Infof(format string, v ...interface{})  {}
func (l *testLogger) Warnf(format string, v ...interface{})  {}
func (l *testLogger) Errorf(format string, v ...interface{}) {}

func getTestConfig() *Config {
	return &Config{
		SecretKey:         "sk_test_fake123",
		PublishableKey:    "pk_test_fake123",
		WebhookSecret:     "whsec_test_secret",
		Environment:       EnvironmentTest,
		MaxNetworkRetries: 3,
		ConnectTimeout:    30 * time.Second,
		Timeout:           80 * time.Second,
		WebhookTolerance:  300 * time.Second,
		Logger:            &testLogger{},
	}
}

func TestNewEventProcessor(t *testing.T) {
	logger := &testLogger{}
	processor := NewEventProcessor(logger)

	assert.NotNil(t, processor)
	assert.NotNil(t, processor.handlers)
	assert.NotNil(t, processor.stats)
	assert.Equal(t, logger, processor.logger)
}

func TestEventProcessor_RegisterHandler(t *testing.T) {
	processor := NewEventProcessor(&testLogger{})

	handler := func(ctx context.Context, event *stripe.Event) error {
		return nil
	}

	processor.RegisterHandler(EventPaymentIntentSucceeded, handler)

	assert.Len(t, processor.handlers[EventPaymentIntentSucceeded], 1)
}

func TestEventProcessor_ProcessEvent(t *testing.T) {
	t.Run("Process event with registered handler", func(t *testing.T) {
		processor := NewEventProcessor(&testLogger{})

		handlerCalled := false
		handler := func(ctx context.Context, event *stripe.Event) error {
			handlerCalled = true
			return nil
		}

		processor.RegisterHandler(EventPaymentIntentSucceeded, handler)

		event := &stripe.Event{
			ID:   "evt_test_123",
			Type: EventPaymentIntentSucceeded,
		}

		err := processor.ProcessEvent(context.Background(), event)
		assert.NoError(t, err)
		assert.True(t, handlerCalled)

		stats := processor.GetStatistics()
		assert.Equal(t, int64(1), stats.TotalProcessed)
		assert.Equal(t, int64(1), stats.SuccessfullyProcessed)
		assert.Equal(t, int64(0), stats.FailedProcessed)
	})

	t.Run("Process event without registered handler", func(t *testing.T) {
		processor := NewEventProcessor(&testLogger{})

		event := &stripe.Event{
			ID:   "evt_test_123",
			Type: "unknown.event",
		}

		err := processor.ProcessEvent(context.Background(), event)
		assert.NoError(t, err) // Should not error for missing handler

		stats := processor.GetStatistics()
		assert.Equal(t, int64(1), stats.TotalProcessed)
		assert.Equal(t, int64(1), stats.SuccessfullyProcessed)
	})

	t.Run("Process event with failing handler", func(t *testing.T) {
		processor := NewEventProcessor(&testLogger{})

		handler := func(ctx context.Context, event *stripe.Event) error {
			return assert.AnError
		}

		processor.RegisterHandler(EventPaymentIntentFailed, handler)

		event := &stripe.Event{
			ID:   "evt_test_123",
			Type: EventPaymentIntentFailed,
		}

		err := processor.ProcessEvent(context.Background(), event)
		assert.Error(t, err)

		stats := processor.GetStatistics()
		assert.Equal(t, int64(1), stats.TotalProcessed)
		assert.Equal(t, int64(0), stats.SuccessfullyProcessed)
		assert.Equal(t, int64(1), stats.FailedProcessed)
	})

	t.Run("Nil event", func(t *testing.T) {
		processor := NewEventProcessor(&testLogger{})

		err := processor.ProcessEvent(context.Background(), nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "event cannot be nil")
	})
}

func TestEventProcessor_RegisterDefaultHandlers(t *testing.T) {
	processor := NewEventProcessor(&testLogger{})
	processor.RegisterDefaultHandlers()

	// Check that default handlers are registered
	assert.NotEmpty(t, processor.handlers[EventPaymentIntentSucceeded])
	assert.NotEmpty(t, processor.handlers[EventPaymentIntentFailed])
	assert.NotEmpty(t, processor.handlers[EventChargeRefunded])
	assert.NotEmpty(t, processor.handlers[EventCustomerCreated])
}

func TestEventProcessor_GetStatistics(t *testing.T) {
	processor := NewEventProcessor(&testLogger{})

	handler := func(ctx context.Context, event *stripe.Event) error {
		return nil
	}

	processor.RegisterHandler(EventPaymentIntentSucceeded, handler)

	// Process multiple events
	for i := 0; i < 5; i++ {
		event := &stripe.Event{
			ID:   "evt_test_" + string(rune(i)),
			Type: EventPaymentIntentSucceeded,
		}
		processor.ProcessEvent(context.Background(), event)
	}

	stats := processor.GetStatistics()
	assert.Equal(t, int64(5), stats.TotalProcessed)
	assert.Equal(t, int64(5), stats.SuccessfullyProcessed)
	assert.Equal(t, int64(0), stats.FailedProcessed)
	assert.Equal(t, int64(5), stats.EventCounts[EventPaymentIntentSucceeded])
}

func TestEventProcessor_ResetStatistics(t *testing.T) {
	processor := NewEventProcessor(&testLogger{})

	handler := func(ctx context.Context, event *stripe.Event) error {
		return nil
	}

	processor.RegisterHandler(EventPaymentIntentSucceeded, handler)

	event := &stripe.Event{
		ID:   "evt_test_123",
		Type: EventPaymentIntentSucceeded,
	}
	processor.ProcessEvent(context.Background(), event)

	// Reset statistics
	processor.ResetStatistics()

	stats := processor.GetStatistics()
	assert.Equal(t, int64(0), stats.TotalProcessed)
	assert.Equal(t, int64(0), stats.SuccessfullyProcessed)
	assert.Equal(t, int64(0), stats.FailedProcessed)
	assert.Empty(t, stats.EventCounts)
}

func TestNewWebhookHandler(t *testing.T) {
	config := getTestConfig()
	processor := NewEventProcessor(config.Logger)
	handler := NewWebhookHandler(config, processor)

	assert.NotNil(t, handler)
	assert.Equal(t, config, handler.config)
	assert.Equal(t, processor, handler.eventProcessor)
	assert.NotNil(t, handler.processedEvents)
}

func TestWebhookHandler_HandleWebhook_InvalidSignature(t *testing.T) {
	config := getTestConfig()
	processor := NewEventProcessor(config.Logger)
	handler := NewWebhookHandler(config, processor)

	req := &WebhookRequest{
		Payload:   []byte(`{"type":"payment_intent.succeeded"}`),
		Signature: "invalid_signature",
	}

	response, err := handler.HandleWebhook(context.Background(), req)
	assert.Error(t, err)
	assert.NotNil(t, response)
	assert.False(t, response.Received)
	assert.Contains(t, response.Error, "signature verification failed")
}

func TestWebhookHandler_HandleWebhook_EmptyPayload(t *testing.T) {
	config := getTestConfig()
	processor := NewEventProcessor(config.Logger)
	handler := NewWebhookHandler(config, processor)

	req := &WebhookRequest{
		Payload:   []byte{},
		Signature: "test_signature",
	}

	response, err := handler.HandleWebhook(context.Background(), req)
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "payload is empty")
}

func TestWebhookHandler_HandleWebhook_MissingSignature(t *testing.T) {
	config := getTestConfig()
	processor := NewEventProcessor(config.Logger)
	handler := NewWebhookHandler(config, processor)

	req := &WebhookRequest{
		Payload:   []byte(`{"type":"payment_intent.succeeded"}`),
		Signature: "",
	}

	response, err := handler.HandleWebhook(context.Background(), req)
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "signature is missing")
}

func TestWebhookHandler_HandleHTTPWebhook(t *testing.T) {
	config := getTestConfig()
	processor := NewEventProcessor(config.Logger)
	handler := NewWebhookHandler(config, processor)

	t.Run("Missing signature header", func(t *testing.T) {
		body := bytes.NewBufferString(`{"type":"payment_intent.succeeded"}`)
		req := httptest.NewRequest(http.MethodPost, "/webhooks", body)
		w := httptest.NewRecorder()

		handler.HandleHTTPWebhook(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response WebhookResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.False(t, response.Received)
		assert.Contains(t, response.Error, "missing signature")
	})

	t.Run("Invalid signature", func(t *testing.T) {
		body := bytes.NewBufferString(`{"type":"payment_intent.succeeded"}`)
		req := httptest.NewRequest(http.MethodPost, "/webhooks", body)
		req.Header.Set("Stripe-Signature", "invalid_signature")
		w := httptest.NewRecorder()

		handler.HandleHTTPWebhook(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response WebhookResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.False(t, response.Received)
	})
}

func TestWebhookHandler_ProcessedEvents(t *testing.T) {
	config := getTestConfig()
	processor := NewEventProcessor(config.Logger)
	handler := NewWebhookHandler(config, processor)

	t.Run("Mark and check event processed", func(t *testing.T) {
		eventID := "evt_test_123"

		assert.False(t, handler.wasEventProcessed(eventID))

		handler.markEventProcessed(eventID)

		assert.True(t, handler.wasEventProcessed(eventID))
	})

	t.Run("Get processed events count", func(t *testing.T) {
		handler.ClearProcessedEvents()

		handler.markEventProcessed("evt_1")
		handler.markEventProcessed("evt_2")
		handler.markEventProcessed("evt_3")

		assert.Equal(t, 3, handler.GetProcessedEventsCount())
	})

	t.Run("Clear processed events", func(t *testing.T) {
		handler.markEventProcessed("evt_test")

		assert.True(t, handler.wasEventProcessed("evt_test"))

		handler.ClearProcessedEvents()

		assert.False(t, handler.wasEventProcessed("evt_test"))
		assert.Equal(t, 0, handler.GetProcessedEventsCount())
	})
}

func TestParsePaymentIntent(t *testing.T) {
	t.Run("Valid payment intent", func(t *testing.T) {
		piJSON := `{
			"id": "pi_test_123",
			"object": "payment_intent",
			"amount": 2000,
			"currency": "usd",
			"status": "succeeded",
			"description": "Test payment",
			"capture_method": "automatic",
			"confirmation_method": "automatic"
		}`

		event := &stripe.Event{
			Data: &stripe.EventData{
				Raw: json.RawMessage(piJSON),
			},
		}

		pi, err := ParsePaymentIntent(event)
		require.NoError(t, err)
		assert.Equal(t, "pi_test_123", pi.ID)
		assert.Equal(t, int64(2000), pi.Amount)
		assert.Equal(t, "usd", pi.Currency)
		assert.Equal(t, "succeeded", pi.Status)
		assert.Equal(t, "Test payment", pi.Description)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		event := &stripe.Event{
			Data: &stripe.EventData{
				Raw: json.RawMessage(`{invalid json}`),
			},
		}

		_, err := ParsePaymentIntent(event)
		assert.Error(t, err)
	})
}

func TestParseCharge(t *testing.T) {
	t.Run("Valid charge", func(t *testing.T) {
		chargeJSON := `{
			"id": "ch_test_123",
			"object": "charge",
			"amount": 2000,
			"currency": "usd",
			"status": "succeeded",
			"paid": true,
			"refunded": false
		}`

		event := &stripe.Event{
			Data: &stripe.EventData{
				Raw: json.RawMessage(chargeJSON),
			},
		}

		ch, err := ParseCharge(event)
		require.NoError(t, err)
		assert.Equal(t, "ch_test_123", ch.ID)
		assert.Equal(t, int64(2000), ch.Amount)
		assert.True(t, ch.Paid)
		assert.False(t, ch.Refunded)
	})
}

func TestParseCustomer(t *testing.T) {
	t.Run("Valid customer", func(t *testing.T) {
		customerJSON := `{
			"id": "cus_test_123",
			"object": "customer",
			"email": "test@example.com",
			"name": "Test Customer",
			"description": "Test description"
		}`

		event := &stripe.Event{
			Data: &stripe.EventData{
				Raw: json.RawMessage(customerJSON),
			},
		}

		cust, err := ParseCustomer(event)
		require.NoError(t, err)
		assert.Equal(t, "cus_test_123", cust.ID)
		assert.Equal(t, "test@example.com", cust.Email)
		assert.Equal(t, "Test Customer", cust.Name)
	})
}

func TestParseRefund(t *testing.T) {
	t.Run("Valid refund", func(t *testing.T) {
		refundJSON := `{
			"id": "re_test_123",
			"object": "refund",
			"amount": 1000,
			"currency": "usd",
			"status": "succeeded",
			"reason": "requested_by_customer"
		}`

		event := &stripe.Event{
			Data: &stripe.EventData{
				Raw: json.RawMessage(refundJSON),
			},
		}

		ref, err := ParseRefund(event)
		require.NoError(t, err)
		assert.Equal(t, "re_test_123", ref.ID)
		assert.Equal(t, int64(1000), ref.Amount)
		assert.Equal(t, "succeeded", ref.Status)
		assert.Equal(t, "requested_by_customer", ref.Reason)
	})
}

## Quote Status System

A comprehensive quote status management system with state machine pattern, WebSocket notifications, and history tracking.

## Features

- **State Machine**: Controlled status transitions with validation
- **WebSocket Notifications**: Real-time updates to connected clients
- **History Tracking**: Complete audit trail of all status changes
- **Event-Driven**: Event system for status changes
- **Validation**: Business logic validation for transitions
- **Auto-Expiration**: Automatic expiration of quotes past valid date

## Status States

```
┌─────────┐     send      ┌─────────┐     view      ┌─────────┐
│  DRAFT  │──────────────▶│  SENT   │──────────────▶│  VIEWED │
└─────────┘               └─────────┘               └─────────┘
                              │  │                       │  │
                              │  │                       │  │
                          accept│  │decline          accept│  │decline
                              │  │                       │  │
                              ▼  ▼                       ▼  ▼
                          ┌──────────┐              ┌──────────┐
                          │ ACCEPTED │              │ DECLINED │
                          └──────────┘              └──────────┘
                              │
                              │ expire
                              ▼
                          ┌──────────┐
                          │ EXPIRED  │
                          └──────────┘
```

### Status Definitions

| Status       | Description                   | Terminal | Who Can Set |
| ------------ | ----------------------------- | -------- | ----------- |
| **DRAFT**    | Quote is being created/edited | No       | User        |
| **SENT**     | Quote sent to customer        | No       | User        |
| **VIEWED**   | Customer viewed the quote     | No       | System      |
| **ACCEPTED** | Customer accepted the quote   | Yes\*    | Customer    |
| **DECLINED** | Customer declined the quote   | Yes      | Customer    |
| **EXPIRED**  | Quote passed valid_until date | Yes      | System      |

\*Accepted quotes can only transition to EXPIRED

## Allowed Transitions

### From DRAFT

- **→ SENT**: When quote is ready to send to customer
  - Requires: At least one line item, customer email

### From SENT

- **→ VIEWED**: When customer opens the quote
  - Triggered: Automatically by system
- **→ ACCEPTED**: When customer accepts
- **→ DECLINED**: When customer declines
- **→ DRAFT**: When editing is needed (back to draft)

### From VIEWED

- **→ ACCEPTED**: When customer accepts
- **→ DECLINED**: When customer declines

### From ACCEPTED

- **→ EXPIRED**: Only when quote expires

### Terminal States

- DECLINED: No further transitions
- EXPIRED: No further transitions

## Backend Implementation

### State Machine Service

```go
// Initialize the state machine
hub := services.NewWebSocketHub()
go hub.Run()

notificationSvc := services.NewWebSocketNotificationService(hub)
statusMachine := services.NewQuoteStatusMachine(db, notificationSvc)

// Send a quote
err := statusMachine.SendQuote(ctx, quoteID, userID)

// Mark as viewed
err := statusMachine.MarkViewed(ctx, quoteID)

// Accept quote
err := statusMachine.AcceptQuote(ctx, quoteID, customerID, "reason")

// Decline quote
err := statusMachine.DeclineQuote(ctx, quoteID, customerID, "reason")

// Check for allowed transitions
allowed := statusMachine.IsTransitionAllowed(
    models.QuoteStatusDraft,
    models.QuoteStatusSent,
)

// Get status history
history, err := statusMachine.GetStatusHistory(ctx, quoteID)
```

### Validation Rules

The state machine validates transitions before allowing them:

```go
// Example validation for sending a quote
func (sm *QuoteStatusMachine) ValidateTransition(
    ctx context.Context,
    transition *models.QuoteStatusTransition,
) error {
    // Check if transition is allowed
    if !sm.IsTransitionAllowed(transition.FromStatus, transition.ToStatus) {
        return ErrInvalidTransition
    }

    // Business logic validation
    if transition.ToStatus == models.QuoteStatusSent {
        // Must have line items
        if len(quote.Items) == 0 {
            return ValidationError{
                Field:   "items",
                Message: "cannot send quote without line items",
            }
        }

        // Must have customer email
        if quote.CustomerEmail == "" {
            return ValidationError{
                Field:   "customer_email",
                Message: "cannot send quote without customer email",
            }
        }
    }

    return nil
}
```

### WebSocket Notifications

Setup WebSocket endpoint:

```go
// In routes.go
wsHandler := handlers.NewWebSocketHandler(notificationService)
router.GET("/ws", middleware.Auth(), wsHandler.HandleWebSocket)
```

Notifications are automatically sent when status changes:

```go
// In state machine Transition method
notification := &models.QuoteStatusNotification{
    Type:      "status_change",
    QuoteID:   transition.QuoteID,
    Event:     transition.Event,
    Status:    transition.ToStatus,
    Timestamp: time.Now(),
    Data: map[string]interface{}{
        "from_status": transition.FromStatus,
        "reason":      transition.Reason,
    },
}

// Sent asynchronously to all subscribed clients
go notificationSvc.SendQuoteStatusNotification(ctx, notification)
```

### History Tracking

Every status change is recorded:

```go
type QuoteStatusHistory struct {
    ID            uuid.UUID
    QuoteID       uuid.UUID
    FromStatus    QuoteStatus
    ToStatus      QuoteStatus
    Event         QuoteStatusEvent
    ChangedByID   uuid.UUID
    ChangedByType string // "user", "customer", "system"
    Reason        string
    Metadata      map[string]interface{}
    CreatedAt     time.Time
}
```

Retrieve history:

```go
// Get all history for a quote
history, err := statusMachine.GetStatusHistory(ctx, quoteID)

// Get latest status change
latest, err := statusMachine.GetLatestStatusChange(ctx, quoteID)
```

### Auto-Expiration

Schedule a cron job to check for expired quotes:

```go
// Run periodically (e.g., every hour)
func checkExpiredQuotes() {
    err := statusMachine.CheckAndExpireQuotes(context.Background())
    if err != nil {
        log.Printf("Failed to expire quotes: %v", err)
    }
}

// Example cron setup
c := cron.New()
c.AddFunc("0 * * * *", checkExpiredQuotes) // Every hour
c.Start()
```

## Frontend Implementation

### WebSocket Hook

```tsx
import { useWebSocket, useQuoteStatusUpdates } from './hooks/useWebSocket';

// Basic WebSocket connection
const { ws, isConnected, subscribe, unsubscribe } = useWebSocket({
  onMessage: (message) => {
    console.log('Received:', message);
  },
});

// Subscribe to quote updates
useEffect(() => {
  if (quoteId && isConnected) {
    subscribe(quoteId);
    return () => unsubscribe(quoteId);
  }
}, [quoteId, isConnected]);

// Or use the convenient hook
const { lastMessage, isConnected } = useQuoteStatusUpdates(
  quoteId,
  (message) => {
    console.log('Status changed to:', message.status);
  }
);
```

### Status Badge Component

```tsx
import { QuoteStatusBadge } from './components/quotes';

function QuoteCard({ quote }) {
  return (
    <div>
      <QuoteStatusBadge status={quote.status} showIcon={true} />
    </div>
  );
}
```

### Status Timeline Component

```tsx
import { QuoteStatusTimeline } from './components/quotes';
import { useQuery } from '@tanstack/react-query';

function QuoteHistory({ quoteId }) {
  const { data: history } = useQuery({
    queryKey: ['quote-history', quoteId],
    queryFn: () =>
      fetch(`/api/v1/quotes/${quoteId}/history`).then((r) => r.json()),
  });

  return <QuoteStatusTimeline history={history || []} />;
}
```

### Status Actions Component

```tsx
import { QuoteStatusActions } from './components/quotes';

function QuoteView({ quote }) {
  return (
    <div>
      <QuoteStatusActions
        quoteId={quote.id}
        currentStatus={quote.status}
        onStatusChange={(newStatus) => {
          console.log('Status changed to:', newStatus);
        }}
      />
    </div>
  );
}
```

## API Endpoints

### Quote Status Endpoints

```
POST   /api/v1/quotes/:id/send      - Send quote to customer
POST   /api/v1/quotes/:id/accept    - Accept quote (customer)
POST   /api/v1/quotes/:id/decline   - Decline quote (customer)
GET    /api/v1/quotes/:id/history   - Get status history
GET    /api/v1/quotes/:id/view      - Mark quote as viewed
```

### WebSocket Endpoint

```
WS     /api/v1/ws                   - WebSocket connection
```

### WebSocket Messages

**Subscribe to quote:**

```json
{
  "action": "subscribe",
  "quote_id": "uuid"
}
```

**Unsubscribe from quote:**

```json
{
  "action": "unsubscribe",
  "quote_id": "uuid"
}
```

**Status change notification (received):**

```json
{
  "type": "status_change",
  "quote_id": "uuid",
  "event": "quote.accepted",
  "status": "accepted",
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    "from_status": "sent",
    "reason": "Looks good!"
  }
}
```

## Events

### Event Types

| Event            | Trigger           | Status Change          |
| ---------------- | ----------------- | ---------------------- |
| `quote.created`  | Quote created     | → DRAFT                |
| `quote.sent`     | User sends quote  | DRAFT → SENT           |
| `quote.viewed`   | Customer views    | SENT → VIEWED          |
| `quote.accepted` | Customer accepts  | SENT/VIEWED → ACCEPTED |
| `quote.declined` | Customer declines | SENT/VIEWED → DECLINED |
| `quote.expired`  | System expires    | ANY → EXPIRED          |
| `quote.updated`  | Quote modified    | No status change       |

## Database Migrations

Create the status history table:

```sql
CREATE TABLE quote_status_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quote_id UUID NOT NULL REFERENCES quotes(id) ON DELETE CASCADE,
    from_status VARCHAR(20),
    to_status VARCHAR(20) NOT NULL,
    event VARCHAR(50) NOT NULL,
    changed_by_id UUID,
    changed_by_type VARCHAR(20),
    reason TEXT,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_quote_status_history_quote_id ON quote_status_history(quote_id);
CREATE INDEX idx_quote_status_history_created_at ON quote_status_history(created_at DESC);
```

## Testing

### Backend Tests

```go
func TestStatusTransition(t *testing.T) {
    // Test valid transition
    err := statusMachine.SendQuote(ctx, quoteID, userID)
    assert.NoError(t, err)

    // Test invalid transition
    err = statusMachine.AcceptQuote(ctx, quoteID, customerID, "")
    assert.Error(t, err)
    assert.Equal(t, ErrInvalidTransition, err)
}

func TestValidation(t *testing.T) {
    // Test sending quote without items
    quote := &models.Quote{
        Items: []models.LineItem{},
    }
    err := statusMachine.SendQuote(ctx, quote.ID, userID)
    assert.Error(t, err)
}

func TestHistory(t *testing.T) {
    // Perform transitions
    statusMachine.SendQuote(ctx, quoteID, userID)
    statusMachine.MarkViewed(ctx, quoteID)

    // Check history
    history, err := statusMachine.GetStatusHistory(ctx, quoteID)
    assert.NoError(t, err)
    assert.Len(t, history, 2)
}
```

### Frontend Tests

```tsx
import { render, screen, waitFor } from '@testing-library/react';
import { QuoteStatusActions } from './QuoteStatusActions';

describe('QuoteStatusActions', () => {
  it('should show send button for draft quotes', () => {
    render(
      <QuoteStatusActions quoteId="123" currentStatus={QuoteStatus.DRAFT} />
    );
    expect(screen.getByText('Send to Customer')).toBeInTheDocument();
  });

  it('should show accept/decline for sent quotes', () => {
    render(
      <QuoteStatusActions quoteId="123" currentStatus={QuoteStatus.SENT} />
    );
    expect(screen.getByText('Accept Quote')).toBeInTheDocument();
    expect(screen.getByText('Decline Quote')).toBeInTheDocument();
  });
});
```

## Error Handling

### Backend Errors

```go
// Handle validation errors
if err := statusMachine.SendQuote(ctx, quoteID, userID); err != nil {
    switch {
    case errors.Is(err, services.ErrInvalidTransition):
        // Return 400 Bad Request
        return c.JSON(http.StatusBadRequest, gin.H{
            "error": "invalid_transition",
            "message": err.Error(),
        })
    case errors.Is(err, services.ErrQuoteExpired):
        // Return 409 Conflict
        return c.JSON(http.StatusConflict, gin.H{
            "error": "quote_expired",
            "message": "Quote has expired",
        })
    default:
        // Return 500 Internal Server Error
        return c.JSON(http.StatusInternalServerError, gin.H{
            "error": "server_error",
            "message": "Failed to update quote status",
        })
    }
}
```

### Frontend Error Handling

```tsx
const sendMutation = useMutation({
  mutationFn: () => quoteService.sendQuote(quoteId),
  onError: (error: any) => {
    if (error.response?.data?.error === 'invalid_transition') {
      toast.error('Cannot send this quote at this time');
    } else if (error.response?.data?.error === 'quote_expired') {
      toast.error('This quote has expired');
    } else {
      toast.error('Failed to send quote');
    }
  },
});
```

## Best Practices

1. **Always use the state machine** for status transitions

   - Don't update status directly in the database
   - Use the state machine methods to ensure validation

2. **Subscribe to WebSocket updates** for real-time UX

   - Subscribe when viewing a quote
   - Unsubscribe when leaving the page

3. **Check allowed transitions** before showing actions

   - Use `IsTransitionAllowed()` to show/hide buttons
   - Provide clear feedback when actions aren't available

4. **Track history for all changes**

   - Include `reason` when declining
   - Add metadata for important context

5. **Handle expiration gracefully**
   - Run expiration check regularly
   - Show clear messaging to users

## Performance Considerations

- **WebSocket connections**: Limit to 1000 concurrent connections per server
- **History table**: Partition by created_at for large datasets
- **Notifications**: Use goroutines for non-blocking sends
- **Cache**: Consider caching allowed transitions map

## Security

- **Authentication**: All WebSocket connections must be authenticated
- **Authorization**: Validate user permissions before transitions
- **Validation**: Always validate on backend, never trust client
- **Audit trail**: History table provides complete audit log

## Monitoring

Track these metrics:

- Status transition counts by type
- WebSocket connection count
- Average time in each status
- Failed transition attempts
- Quote expiration rate

## License

MIT

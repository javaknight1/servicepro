# Analytics

Event tracking and metrics aggregation for ServicePro.

## Overview

The analytics system provides:

- Custom event tracking (backend and frontend)
- Google Analytics 4 integration
- Metrics aggregation and dashboards
- Privacy-compliant data collection

## Event Categories

| Category    | Description                          |
| ----------- | ------------------------------------ |
| user        | Authentication and profile events    |
| page        | Page views and navigation            |
| feature     | Feature usage and interactions       |
| business    | Customer, job, quote, invoice events |
| performance | API latency and performance          |
| error       | Error tracking                       |
| api         | API request tracking                 |

## Backend Tracking

```go
// Initialize tracker
tracker := analytics.NewTracker(eventStore, config)

// Track event
event := analytics.NewEvent("feature_used", "feature").
    WithUser(userID).
    WithProperty("feature_name", "dashboard")

tracker.Track(ctx, event)
```

## Frontend Tracking

```typescript
import { analytics } from '../services/analytics';

// Initialize
analytics.init({
  enabled: true,
  ga4MeasurementId: 'G-XXXXXXXXXX',
});

// Track events
analytics.track('button_click', 'feature', { button: 'submit' });
analytics.pageView('/dashboard', 'Dashboard');
analytics.trackLogin('email', 'user-123');
```

## React Integration

```tsx
import { AnalyticsProvider, useAnalytics } from '../contexts/AnalyticsContext';

// Wrap app
<AnalyticsProvider config={config} trackPageViews={true}>
  <App />
</AnalyticsProvider>;

// Use hooks
const { track, trackClick } = useAnalytics();
const { trackCustomerCreated } = useBusinessTracking();
```

## API Endpoints

| Method | Endpoint                   | Description              |
| ------ | -------------------------- | ------------------------ |
| POST   | `/analytics/events`        | Track single event       |
| POST   | `/analytics/events/batch`  | Track multiple events    |
| POST   | `/analytics/events/query`  | Query events             |
| POST   | `/analytics/metrics/query` | Query aggregated metrics |

## Pre-aggregated Metrics

- `unique_users` - Active users by period
- `event_count` - Event counts by type
- `api_latency_avg` - Average API response time
- `business_event_count` - Business metrics

## Data Retention

| Data Type      | Retention |
| -------------- | --------- |
| Raw Events     | 30 days   |
| Hourly Metrics | 90 days   |
| Daily Metrics  | 365 days  |

## Configuration

```bash
# Backend
ANALYTICS_ENABLED=true
ANALYTICS_BATCH_SIZE=100
ANALYTICS_FLUSH_INTERVAL=10s

# Frontend
VITE_ANALYTICS_ENABLED=true
VITE_GA4_MEASUREMENT_ID=G-XXXXXXXXXX
```

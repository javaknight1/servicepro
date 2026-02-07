# Application Observability Guide

A comprehensive guide to application metrics, monitoring, and observability for ServicePro.

---

## Table of Contents

- [What is Observability?](#what-is-observability)
- [The Three Pillars](#the-three-pillars-of-observability)
- [Application Metrics Deep Dive](#application-metrics-deep-dive)
  - [Metric Types](#metric-types)
  - [What to Measure](#what-to-measure)
  - [Naming Conventions](#naming-conventions)
- [The Prometheus + Grafana Stack](#the-prometheus--grafana-stack)
  - [How It Works](#how-it-works)
  - [ServicePro's Current Setup](#servicepros-current-setup)
  - [Running Prometheus Locally](#running-prometheus-locally)
  - [Running Grafana Locally](#running-grafana-locally)
- [Implementation Options](#implementation-options)
- [Alerting](#alerting)
- [ServicePro Observability Roadmap](#servicepro-observability-roadmap)
- [Key Concepts Glossary](#key-concepts-glossary)

---

## What is Observability?

Observability answers the question: **"Is my application healthy right now, and if not, why?"**

Think of it like a car's dashboard gauges — engine temperature, oil pressure, RPM. The driver and mechanic look at these to keep the car running. You don't need them when everything is fine, but when something goes wrong, they tell you exactly where to look.

Without observability, you find out about problems when a user complains. With observability, you find out before the user even notices.

---

## The Three Pillars of Observability

The industry recognizes three complementary signals that together give full visibility into a running system:

| Pillar      | What It Is                                              | What It Answers                          | Tool Examples                      | ServicePro Status  |
| ----------- | ------------------------------------------------------- | ---------------------------------------- | ---------------------------------- | ------------------ |
| **Metrics** | Numeric time-series data (counters, gauges, histograms) | "How many? How fast? How much?"          | Prometheus, Datadog, CloudWatch    | Implemented (T005) |
| **Logs**    | Discrete text events with context                       | "What happened? What was the context?"   | Loki, BetterStack, CloudWatch Logs | Implemented (T006) |
| **Traces**  | Request flow across services/components                 | "Where did this request spend its time?" | Jaeger, Zipkin, Sentry Tracing     | Planned (T024)     |

Each pillar has different strengths:

- **Metrics** tell you _something_ is wrong (error rate spiked to 5%)
- **Logs** tell you _what_ went wrong ("failed to connect to database: timeout")
- **Traces** tell you _where_ it went wrong (the database call in the invoice service took 3 seconds)

You need all three for full observability, but **metrics are the foundation** — they're cheap to collect, fast to query, and power your alerting.

---

## Application Metrics Deep Dive

### Metric Types

There are four fundamental metric types. Understanding these is critical for effective monitoring.

#### 1. Counter

A value that only goes **up**. Resets to zero on application restart.

```
# Total HTTP requests served
http_requests_total{method="GET", path="/api/customers", status="200"} 15432

# Total errors
http_errors_total{type="internal_server_error"} 23

# Total invoices created
invoices_created_total{tenant="acme-hvac"} 847
```

**Use for:** Total requests, total errors, total emails sent, total jobs created — anything you want to count over time.

**How to read it:** A counter's raw value isn't very useful. You care about the **rate** — "how fast is it going up?" A Prometheus query like `rate(http_requests_total[5m])` gives you "requests per second over the last 5 minutes."

#### 2. Gauge

A value that goes **up and down**. Represents a current snapshot.

```
# Current active database connections
db_connections_open 12

# Current memory usage
memory_usage_bytes 471859200

# Jobs currently in "in_progress" status
jobs_in_progress{tenant="acme-hvac"} 7
```

**Use for:** Active connections, queue depth, memory usage, disk space, temperature — anything that fluctuates.

#### 3. Histogram

Tracks the **distribution** of observed values across configurable buckets. This is the most powerful metric type.

```
# How long do HTTP requests take?
http_request_duration_seconds_bucket{le="0.005"} 8000   # 8000 requests completed in < 5ms
http_request_duration_seconds_bucket{le="0.01"}  8500   # 8500 requests completed in < 10ms
http_request_duration_seconds_bucket{le="0.025"} 9200   # 9200 requests completed in < 25ms
http_request_duration_seconds_bucket{le="0.05"}  9500   # 9500 requests completed in < 50ms
http_request_duration_seconds_bucket{le="0.1"}   9800   # 9800 requests completed in < 100ms
http_request_duration_seconds_bucket{le="0.25"}  9950   # 9950 requests completed in < 250ms
http_request_duration_seconds_bucket{le="0.5"}   9990   # 9990 requests completed in < 500ms
http_request_duration_seconds_bucket{le="1.0"}   9998   # 9998 requests completed in < 1s
http_request_duration_seconds_bucket{le="+Inf"}  10000  # 10000 total requests
http_request_duration_seconds_sum 125.4                  # Total seconds spent
http_request_duration_seconds_count 10000                # Total observations
```

**Use for:** Request latency, query duration, response sizes, queue wait times — anything where you need percentiles (p50, p95, p99).

**Why percentiles matter:** An average of 50ms might hide the fact that 1% of users wait 5 seconds. The p99 (99th percentile) shows you the worst experience — "99% of requests complete in under X ms."

#### 4. Summary

Similar to histogram but computes percentiles on the client side (in your app). Less commonly used because:

- Cannot be aggregated across multiple instances
- Percentile targets must be defined at instrumentation time, not query time

**Generally prefer histograms over summaries.**

### What to Measure

The industry-standard approach is the **RED method** for request-driven services and the **USE method** for infrastructure:

#### RED Method (for services)

| Signal       | What                         | Metric Example                                            |
| ------------ | ---------------------------- | --------------------------------------------------------- |
| **R**ate     | Requests per second          | `rate(http_requests_total[5m])`                           |
| **E**rrors   | Failed requests per second   | `rate(http_errors_total[5m])`                             |
| **D**uration | Time per request (histogram) | `histogram_quantile(0.95, http_request_duration_seconds)` |

#### USE Method (for infrastructure)

| Signal          | What                  | Metric Example                             |
| --------------- | --------------------- | ------------------------------------------ |
| **U**tilization | % of resource in use  | `db_connections_open / db_connections_max` |
| **S**aturation  | Queue depth / backlog | `db_connections_wait_count`                |
| **E**rrors      | Error events          | `db_connection_errors_total`               |

#### What ServicePro Should Measure

**Request layer (already captured by access logging middleware):**

- `http_request_duration_seconds` — histogram by method, path, status
- `http_requests_total` — counter by method, path, status
- `http_response_size_bytes` — histogram

**Database layer (T017 — query performance monitoring):**

- `db_query_duration_seconds` — histogram by operation (SELECT/INSERT/UPDATE/DELETE), table
- `db_query_errors_total` — counter by operation, table
- `db_connections_open` — gauge
- `db_connections_idle` — gauge
- `db_connections_max_open` — gauge
- `db_connections_wait_count` — counter
- `db_connections_wait_duration_seconds` — counter

**Application layer:**

- `go_goroutines` — gauge (built-in with Prometheus Go client)
- `go_memstats_*` — built-in memory metrics
- `process_cpu_seconds_total` — built-in

**Business layer (future — T025/T026):**

- `jobs_created_total` — counter by tenant
- `invoices_total_amount` — counter (total dollar amount invoiced)
- `quotes_converted_total` — counter

### Naming Conventions

Follow Prometheus naming best practices:

```
<namespace>_<subsystem>_<metric_name>_<unit>

Examples:
  servicepro_http_request_duration_seconds
  servicepro_db_query_duration_seconds
  servicepro_db_connections_open
  servicepro_email_send_duration_seconds
  servicepro_jobs_created_total
```

Rules:

- Use `snake_case`
- Include the unit as a suffix: `_seconds`, `_bytes`, `_total`
- Counters must end in `_total`
- Use base units: seconds (not milliseconds), bytes (not megabytes)
- Use a namespace prefix to avoid collisions (`servicepro_`)

---

## The Prometheus + Grafana Stack

### How It Works

This is the industry-standard open-source monitoring stack:

```
┌─────────────────┐     scrapes every 15s     ┌─────────────────┐
│  ServicePro     │ ◄──────────────────────── │   Prometheus    │
│  Backend        │    GET /metrics            │   Server        │
│                 │    (text format)           │   (time-series  │
│  Exposes:       │                            │    database)    │
│  /metrics       │                            └────────┬────────┘
└─────────────────┘                                     │
                                                        │ queries (PromQL)
                                                        │
                                              ┌─────────▼────────┐
                                              │     Grafana      │
                                              │  (dashboards,    │
                                              │   visualizations)│
                                              └─────────┬────────┘
                                                        │
                                                        │ fires alerts
                                                        │
                                              ┌─────────▼────────┐
                                              │  Alertmanager    │
                                              │  (Slack, email,  │
                                              │   PagerDuty)     │
                                              └──────────────────┘
```

**Step by step:**

1. Your application collects metrics in memory (counters, histograms, etc.)
2. These are exposed at `GET /metrics` in Prometheus text exposition format
3. Prometheus server scrapes that endpoint every N seconds (default 15s)
4. Prometheus stores the data as time-series in its local database (default 15-day retention)
5. Grafana connects to Prometheus as a data source and builds dashboards
6. Alertmanager evaluates rules against Prometheus data and sends notifications

### ServicePro's Current Setup

ServicePro already has the application-side instrumentation in place:

**Prometheus metrics client:** `backend/pkg/clients/metrics/`

```
pkg/clients/metrics/
├── interface.go          # Metrics client interface (Counter, Gauge, Histogram, Summary)
├── factory.go            # Factory for creating metrics clients
├── prometheus/
│   └── prometheus.go     # Prometheus implementation
└── noop/
    └── noop.go           # No-op implementation (when metrics disabled)
```

**Configuration:** `backend/config/config.go`

```go
type PrometheusConfig struct {
    Enabled      bool          // PROMETHEUS_ENABLED (default: false)
    MetricsPath  string        // PROMETHEUS_METRICS_PATH (default: /metrics)
    Namespace    string        // PROMETHEUS_NAMESPACE
    Subsystem    string        // PROMETHEUS_SUBSYSTEM
    PushGateway  string        // PROMETHEUS_PUSH_GATEWAY (for batch jobs)
    PushInterval time.Duration // PROMETHEUS_PUSH_INTERVAL
}
```

**Endpoint registration:** `backend/internal/api/routes/routes.go`

The `/metrics` endpoint is registered when `PROMETHEUS_ENABLED=true` and serves the standard Prometheus text format.

### Running Prometheus Locally

Create a `prometheus.yml` configuration file:

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'servicepro-backend'
    metrics_path: '/metrics'
    static_configs:
      - targets: ['host.docker.internal:8080'] # or localhost:8080 if not using Docker
```

Run with Docker:

```bash
docker run -d \
  --name prometheus \
  -p 9090:9090 \
  -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml \
  prom/prometheus
```

Access at `http://localhost:9090`. Use the "Graph" tab to query metrics with PromQL.

### Running Grafana Locally

```bash
docker run -d \
  --name grafana \
  -p 3001:3000 \
  grafana/grafana
```

Access at `http://localhost:3001` (default login: admin/admin).

**Setup steps:**

1. Go to Configuration > Data Sources > Add Prometheus
2. Set URL to `http://host.docker.internal:9090` (or `http://prometheus:9090` if on same Docker network)
3. Click "Save & Test"
4. Go to Dashboards > Import > Create your first dashboard

### Common PromQL Queries

```promql
# Request rate (requests per second over last 5 minutes)
rate(http_requests_total[5m])

# Error rate as percentage
rate(http_errors_total[5m]) / rate(http_requests_total[5m]) * 100

# 95th percentile request latency
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# 99th percentile database query time
histogram_quantile(0.99, rate(db_query_duration_seconds_bucket[5m]))

# Average request duration
rate(http_request_duration_seconds_sum[5m]) / rate(http_request_duration_seconds_count[5m])

# Database connection pool utilization
db_connections_open / db_connections_max_open

# Top 5 slowest endpoints (by p95)
topk(5, histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])))
```

---

## Implementation Options

### Comparison

| Option                                 | Type               | Cost                                             | Setup Effort    | Best For                                  |
| -------------------------------------- | ------------------ | ------------------------------------------------ | --------------- | ----------------------------------------- |
| **Prometheus + Grafana (self-hosted)** | Open source        | Free (infra costs only)                          | Medium          | Full control, already have infrastructure |
| **Grafana Cloud**                      | Managed SaaS       | Free tier: 10k metrics, 50GB logs. Then ~$50+/mo | Low             | Want Prometheus without running servers   |
| **Datadog**                            | Fully managed SaaS | $15-23/host/mo                                   | Low             | All-in-one, teams with budget             |
| **New Relic**                          | Fully managed SaaS | Free tier (100GB/mo). Then $25+/mo               | Low             | Similar to Datadog, generous free tier    |
| **AWS CloudWatch**                     | AWS-native         | Pay per metric (~$0.30/metric/mo)                | Low (if on AWS) | Already deep in AWS ecosystem             |

### Recommendation for ServicePro

ServicePro already has the Prometheus client library integrated in the backend. The two practical paths are:

**Option A: Grafana Cloud (recommended for getting started)**

- Sign up at grafana.com (free tier is generous)
- Get a Prometheus remote-write endpoint
- Configure ServicePro to push metrics (or set up Grafana Agent to scrape)
- Dashboards, alerting, and log aggregation included
- No servers to manage

**Option B: Self-hosted Prometheus + Grafana (recommended for production)**

- Add `prometheus` and `grafana` services to `docker-compose.yml`
- Full control, no external dependency
- Requires maintaining the services

---

## Alerting

Metrics are most valuable when they trigger alerts proactively.

### What to Alert On

**High severity (page someone immediately):**

- Error rate > 5% for 5 minutes
- p95 latency > 2 seconds for 5 minutes
- Application is down (no metrics scraped for 2 minutes)
- Database connection pool exhausted

**Medium severity (notify during business hours):**

- Error rate > 1% for 15 minutes
- p95 latency > 500ms for 15 minutes
- Database connection pool > 80% utilized
- Slow queries (p99 > 1 second) sustained for 10 minutes

**Low severity (review weekly):**

- Disk usage > 70%
- Memory usage trending upward over 7 days
- Increasing query latency trend

### Example Prometheus Alerting Rules

```yaml
# alerts.yml
groups:
  - name: servicepro
    rules:
      - alert: HighErrorRate
        expr: rate(http_errors_total[5m]) / rate(http_requests_total[5m]) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: 'Error rate is above 5%'
          description: '{{ $value | humanizePercentage }} of requests are failing'

      - alert: HighLatency
        expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 2
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: '95th percentile latency is above 2 seconds'

      - alert: SlowDatabaseQueries
        expr: histogram_quantile(0.99, rate(db_query_duration_seconds_bucket[5m])) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: 'Database query p99 is above 1 second'

      - alert: DatabaseConnectionPoolHigh
        expr: db_connections_open / db_connections_max_open > 0.8
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: 'Database connection pool is {{ $value | humanizePercentage }} utilized'
```

---

## ServicePro Observability Roadmap

| Task                                | Status  | What It Adds                                                           |
| ----------------------------------- | ------- | ---------------------------------------------------------------------- |
| T005 - Prometheus /metrics endpoint | Done    | Foundation — metrics exposed from the app                              |
| T006 - Structured JSON logging      | Done    | Structured logs with context (request ID, user ID, tenant ID)          |
| T017 - Query performance monitoring | Pending | Database query histograms, slow query logging, connection pool metrics |
| T024 - Full Sentry integration      | Pending | Error tracking + distributed tracing (the third pillar)                |
| T012 - Bundle size monitoring in CI | Pending | Frontend build-time metrics in CI pipeline                             |

### What "done" looks like

When the full observability stack is in place, you can answer questions like:

- "What's our average API response time?" — Grafana dashboard, request histogram
- "Which database query is slowest?" — T017 metrics + slow query logs
- "Why was the app slow yesterday at 3pm?" — Grafana time range + correlated logs
- "Which endpoint has the most errors?" — Error rate by path
- "Are we about to run out of database connections?" — Connection pool gauge + alert
- "Did last night's deploy make things faster or slower?" — Before/after comparison in Grafana

---

## Key Concepts Glossary

| Term                | Definition                                                                                                                                    |
| ------------------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| **Scraping**        | Prometheus pulling metrics from your app's `/metrics` endpoint at regular intervals                                                           |
| **Time series**     | A sequence of (timestamp, value) pairs identified by a metric name and label set                                                              |
| **Label**           | A key-value pair that adds dimensions to a metric (e.g., `method="GET"`, `status="200"`)                                                      |
| **Cardinality**     | The number of unique label combinations for a metric. High cardinality (thousands of unique values) causes performance problems in Prometheus |
| **PromQL**          | Prometheus Query Language — used to query metrics in Grafana dashboards and alerting rules                                                    |
| **SLI**             | Service Level Indicator — a metric that measures service quality (e.g., request latency p95)                                                  |
| **SLO**             | Service Level Objective — a target for an SLI (e.g., "p95 latency < 200ms")                                                                   |
| **SLA**             | Service Level Agreement — a contractual commitment to meet SLOs                                                                               |
| **RED method**      | Rate, Errors, Duration — the three key signals for monitoring request-driven services                                                         |
| **USE method**      | Utilization, Saturation, Errors — the three key signals for monitoring infrastructure resources                                               |
| **Golden signals**  | Google's version: latency, traffic, errors, saturation (from the SRE book)                                                                    |
| **Instrumentation** | Adding metric collection code to your application                                                                                             |
| **Exposition**      | Serving metrics in a format Prometheus can scrape                                                                                             |
| **Remote write**    | Pushing metrics from your app or agent to a remote Prometheus-compatible store                                                                |

---

## Further Reading

- [Prometheus documentation](https://prometheus.io/docs/)
- [Grafana documentation](https://grafana.com/docs/)
- [Google SRE Book - Monitoring](https://sre.google/sre-book/monitoring-distributed-systems/)
- [RED Method (Tom Wilkie)](https://www.weave.works/blog/the-red-method-key-metrics-for-microservices-architecture/)
- [USE Method (Brendan Gregg)](https://www.brendangregg.com/usemethod.html)

# ServicePro API image (Cloud Run; also usable with docker compose).
#
# Build from the backend/ directory:
#   docker build -t servicepro-api \
#     --build-arg GIT_SHA=$(git rev-parse --short HEAD) \
#     --build-arg APP_VERSION=$(cat ../VERSION) \
#     backend
#
# The image contains two static binaries:
#   /app/server   the API (default CMD)
#   /app/migrate  golang-migrate runner: docker run ... /app/migrate up|version
# SQL migrations and templates are embedded in the binaries; nothing else is
# copied from the source tree.

# ---- Build stage -------------------------------------------------------------
# Keep in sync with the `toolchain` directive in go.mod: the binary must be
# built with a Go release that has the current stdlib security fixes
# (govulncheck reports stdlib CVEs against the build toolchain). Official
# images set GOTOOLCHAIN=local, so the image tag is what counts.
FROM golang:1.27-alpine AS builder

WORKDIR /src

# Dependencies first so they cache independently of source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# backend/docs (Swagger) is committed and kept current by `make docs`; CI's
# openapi-drift job fails if it is stale, so the image builds from it as-is.

ENV CGO_ENABLED=0 GOOS=linux
RUN go build -trimpath -ldflags="-s -w" -o /out/server ./cmd \
 && go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

# ---- Runtime stage -----------------------------------------------------------
FROM alpine:3.22

# ca-certificates: outbound TLS (Supabase, Stripe, Resend, Sentry, PostHog).
# tzdata: tenant business-hours / timezone handling.
# busybox wget (in the base image) is used by HEALTHCHECK.
RUN apk add --no-cache ca-certificates tzdata \
 && addgroup -S -g 10001 app \
 && adduser -S -D -H -u 10001 -G app app

WORKDIR /app
COPY --from=builder /out/server /out/migrate ./

# Release identity, passed by the release pipeline. Declared in the final
# stage so a new SHA does not invalidate the cached `go build` layers.
# Read at runtime by GET /version, /health and Sentry release tagging.
ARG GIT_SHA=unknown
ARG APP_VERSION=dev
ENV GIT_SHA=${GIT_SHA} \
    APP_VERSION=${APP_VERSION} \
    ENV=production \
    GIN_MODE=release \
    PORT=8080

USER app:app

EXPOSE 8080

# Cloud Run ignores Docker HEALTHCHECK (it uses its own probes: /health/ready
# as startup probe, /health/live as liveness); this serves compose/docker run.
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -q -O /dev/null "http://127.0.0.1:${PORT}/health/live" || exit 1

CMD ["/app/server"]

package mock

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// T126: outside development the logger writes one JSON object per line to
// stdout (Cloud Run -> Cloud Logging) instead of picking a remote provider
// that was never linked into the binary.
func TestNewClient_ProductionWritesJSONToStdout(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.Env = "production"

	client, err := logging.NewClient(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	client.Error(context.Background(), "boom", map[string]any{"invoice_id": "inv_1"})
	os.Stdout = orig
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	line := strings.TrimSpace(buf.String())
	var got map[string]any
	if err := json.Unmarshal([]byte(line), &got); err != nil {
		t.Fatalf("stdout line is not JSON: %q (%v)", line, err)
	}
	if got["severity"] != "ERROR" || got["message"] != "boom" {
		t.Fatalf("unexpected entry: %v", got)
	}
	if fields, _ := got["fields"].(map[string]any); fields["invoice_id"] != "inv_1" {
		t.Fatalf("fields missing: %v", got)
	}
}

// T223: a request ID on the context (set by the RequestID middleware) is
// written on every log line so it can be matched to an error response.
func TestStdoutJSON_IncludesRequestID(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.Env = "production"
	client, err := logging.NewClient(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	client.Info(logging.WithRequestID(context.Background(), "req-123-abc"), "hello", nil)
	os.Stdout = orig
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	var got map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &got); err != nil {
		t.Fatalf("stdout line is not JSON: %q (%v)", buf.String(), err)
	}
	if got["request_id"] != "req-123-abc" {
		t.Fatalf("request_id missing from log line: %v", got)
	}
}

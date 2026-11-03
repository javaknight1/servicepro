package s3

import (
	"context"
	"strings"
	"testing"
	"time"
)

func newTestClient(t *testing.T, presignEndpoint string) *Client {
	t.Helper()
	c, err := NewClient(context.Background(), &Config{
		Endpoint: "http://minio:9000", Bucket: "servicepro-uploads", Region: "us-east-1",
		AccessKeyID: "test", SecretAccessKey: "test", UsePathStyle: true, DisableSSL: true,
		PresignEndpoint: presignEndpoint,
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// T160: presigned links were signed for the API's endpoint and then had their
// host rewritten to PublicURL, which breaks the SigV4 signature
// (SignatureDoesNotMatch). They are now signed for the browser-facing host.
func TestGetPresignedURL_SignedForPresignEndpoint(t *testing.T) {
	url, err := newTestClient(t, "http://localhost:9000").GetPresignedURL(context.Background(), "pdfs/inv.pdf", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(url, "http://localhost:9000/servicepro-uploads/pdfs/inv.pdf?") || !strings.Contains(url, "X-Amz-SignedHeaders=host") {
		t.Fatalf("url = %s", url)
	}
}

func TestGetPresignedURL_DefaultsToEndpoint(t *testing.T) {
	url, err := newTestClient(t, "").GetPresignedURL(context.Background(), "pdfs/inv.pdf", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(url, "http://minio:9000/servicepro-uploads/pdfs/inv.pdf?") {
		t.Fatalf("url = %s", url)
	}
}

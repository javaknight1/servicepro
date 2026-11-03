package smtp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/mail"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/pkg/clients/email"
)

// fakeSMTP accepts one message and returns its DATA section.
func fakeSMTP(t *testing.T) (port int, received <-chan []byte) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	out := make(chan []byte, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
		r := bufio.NewReader(conn)
		reply := func(s string) { _, _ = io.WriteString(conn, s+"\r\n") }
		reply("220 fake ESMTP")
		var data bytes.Buffer
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			cmd := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
				reply("250 fake")
			case cmd == "DATA":
				reply("354 go ahead")
				for {
					l, err := r.ReadString('\n')
					if err != nil {
						return
					}
					if l == ".\r\n" {
						break
					}
					data.WriteString(strings.TrimPrefix(l, "."))
				}
				out <- data.Bytes()
				reply("250 queued")
			case cmd == "QUIT":
				reply("221 bye")
				return
			default:
				reply("250 ok")
			}
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port, out
}

// Regression (T271): the SMTP client (Mailpit locally) dropped the PDF
// attachment and the "Download PDF" link, so invoice PDFs couldn't be
// checked locally before production.
func TestSendInvoiceEmail_IncludesPDFAttachmentAndDownloadLink(t *testing.T) {
	port, received := fakeSMTP(t)
	client, err := NewClient(&Config{Host: "127.0.0.1", Port: port, FromEmail: "noreply@servicepro.local"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	pdf := []byte("%PDF-1.4 fake invoice " + strings.Repeat("x", 200))
	invoice := &models.Invoice{
		InvoiceNumber: "INV-000001",
		TotalAmount:   decimal.RequireFromString("1240.00"),
		DueDate:       time.Date(2026, 9, 19, 0, 0, 0, 0, time.UTC),
	}
	downloadURL := "http://localhost:9000/servicepro-uploads/invoices/x.pdf?X-Amz-Signature=abc&X-Amz-Date=1"
	err = client.SendInvoiceEmail(context.Background(), "jane@example.com", invoice, "",
		&email.Attachment{Filename: "Invoice-INV-000001.pdf", ContentType: "application/pdf", Content: pdf}, downloadURL)
	if err != nil {
		t.Fatalf("SendInvoiceEmail: %v", err)
	}

	var raw []byte
	select {
	case raw = <-received:
	case <-time.After(5 * time.Second):
		t.Fatal("no message received")
	}

	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("parse message: %v", err)
	}
	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/mixed" {
		t.Fatalf("Content-Type = %q (%v), want multipart/mixed", msg.Header.Get("Content-Type"), err)
	}

	mr := multipart.NewReader(msg.Body, params["boundary"])
	var htmlBody string
	var gotPDF []byte
	var gotName string
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("next part: %v", err)
		}
		body, _ := io.ReadAll(part) // still base64 for the PDF part; decoded below
		switch {
		case strings.HasPrefix(part.Header.Get("Content-Type"), "text/html"):
			htmlBody = string(body)
		case strings.HasPrefix(part.Header.Get("Content-Type"), "application/pdf"):
			gotName = part.FileName()
			gotPDF = decodeBase64(t, body)
		}
	}

	if !bytes.Equal(gotPDF, pdf) {
		t.Errorf("PDF attachment = %d bytes, want the %d-byte PDF", len(gotPDF), len(pdf))
	}
	if gotName != "Invoice-INV-000001.pdf" {
		t.Errorf("attachment filename = %q", gotName)
	}
	wantLink := `href="http://localhost:9000/servicepro-uploads/invoices/x.pdf?X-Amz-Signature=abc&amp;X-Amz-Date=1"`
	if !strings.Contains(htmlBody, wantLink) {
		t.Errorf("HTML body missing download link %s:\n%s", wantLink, htmlBody)
	}
}

func TestBuildMessage_NoAttachmentsStaysSinglePart(t *testing.T) {
	c, err := NewClient(&Config{Host: "localhost"})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := c.buildMessage(context.Background(), email.NewEmailMessage("a@example.com", "Hi", "<p>hi</p>"), "noreply@localhost")
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if ct := msg.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
}

func TestDownloadLinkHTML_EmptyWithoutURL(t *testing.T) {
	if got := downloadLinkHTML(""); got != "" {
		t.Errorf("downloadLinkHTML(\"\") = %q, want empty", got)
	}
}

func decodeBase64(t *testing.T, body []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	// The decoder skips the CRLF line breaks.
	if _, err := io.Copy(&out, base64.NewDecoder(base64.StdEncoding, bytes.NewReader(body))); err != nil {
		t.Fatalf("decode base64: %v", err)
	}
	return out.Bytes()
}

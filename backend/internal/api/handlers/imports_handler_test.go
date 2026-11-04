package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/imports"
	"github.com/javaknight1/servicepro/backend/internal/models"
)

// mappedUpload posts a multipart import form to path.
func mappedUpload(t *testing.T, path, filename string, data []byte, fields map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Preview and the checks before a job is created need no service.
	h := NewImportsHandler(nil)
	h.now = func() time.Time { return time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC) }
	r.POST("/imports/:entity/preview", h.PreviewImport)
	r.POST("/imports/:entity", h.StartImport)

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	for k, v := range fields {
		require.NoError(t, w.WriteField(k, v))
	}
	fw, err := w.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = fw.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	req := httptest.NewRequest(http.MethodPost, path, &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "imports", "testdata", name))
	require.NoError(t, err)
	return data
}

func TestPreviewImport_JobberClients(t *testing.T) {
	w := mappedUpload(t, "/imports/customers/preview", "clients.csv", fixture(t, "jobber_clients.csv"), map[string]string{"source": "jobber"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp models.ImportPreviewResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "jobber", resp.Source)
	assert.Empty(t, resp.Missing)
	assert.Equal(t, 5, resp.TotalRows)
	assert.Equal(t, 4, resp.ValidRows)
	assert.Equal(t, 1, resp.ErrorRows)
	require.Len(t, resp.Errors, 1)
	assert.Equal(t, 5, resp.Errors[0].Row)
	assert.Equal(t, "E-mails", resp.Mapping["email"])
	assert.Equal(t, "ap@riversidepm.example", resp.Rows[0].Values["email"], "rows are shown normalised")
	assert.NotEmpty(t, resp.Fields)
}

func TestPreviewImport_HousecallProInvoicesShowSkips(t *testing.T) {
	w := mappedUpload(t, "/imports/invoices/preview", "jobs.csv", fixture(t, "hcp_jobs.csv"), map[string]string{"source": "housecall_pro"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp models.ImportPreviewResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.ValidRows)
	assert.Equal(t, 3, resp.SkipRows)
	assert.Equal(t, "nothing left to pay", resp.Rows[1].Skip)
	_, internal := resp.Rows[1].Values[imports.SkipKey]
	assert.False(t, internal, "the internal skip marker isn't sent")
}

func TestPreviewImport_GenericMapping(t *testing.T) {
	data := fixture(t, "generic_customers.csv")
	w := mappedUpload(t, "/imports/customers/preview", "c.csv", data, nil)
	require.Equal(t, http.StatusOK, w.Code)
	var auto models.ImportPreviewResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &auto))
	assert.NotEmpty(t, auto.Missing, "unusual headers need a mapping")
	assert.Zero(t, auto.ErrorRows, "rows aren't validated until the mapping is complete")

	mapping, err := json.Marshal(map[string]string{
		"full_name": "Customer", "email": "Contact email", "phone": "Tel", "billing_street": "Addr",
		"billing_city": "Town", "billing_state": "ST", "billing_zip": "Postcode", "customer_type": "Kind",
	})
	require.NoError(t, err)
	w = mappedUpload(t, "/imports/customers/preview", "c.csv", data, map[string]string{"mapping": string(mapping)})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var mapped models.ImportPreviewResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &mapped))
	assert.Empty(t, mapped.Missing)
	assert.Equal(t, 2, mapped.ValidRows)
	assert.Equal(t, "Contact email", mapped.Mapping["email"])
}

func TestImportUpload_Rejections(t *testing.T) {
	csv := fixture(t, "generic_customers.csv")
	for name, tc := range map[string]struct {
		path   string
		file   []byte
		fields map[string]string
		code   int
		errKey string
	}{
		"unknown entity":       {"/imports/payments/preview", csv, nil, http.StatusNotFound, "not_found"},
		"unknown source":       {"/imports/customers/preview", csv, map[string]string{"source": "servicetitan"}, http.StatusBadRequest, "invalid_source"},
		"bad mapping JSON":     {"/imports/customers/preview", csv, map[string]string{"mapping": "{"}, http.StatusBadRequest, "invalid_mapping"},
		"mapping unknown col":  {"/imports/customers/preview", csv, map[string]string{"mapping": `{"email":"Nope"}`}, http.StatusBadRequest, "invalid_mapping"},
		"not a spreadsheet":    {"/imports/customers/preview", []byte("PK\x03\x04junk"), nil, http.StatusBadRequest, "invalid_file"},
		"no data rows":         {"/imports/customers/preview", []byte("email\n"), nil, http.StatusBadRequest, "invalid_file"},
		"start without fields": {"/imports/customers", csv, nil, http.StatusBadRequest, "missing_columns"},
	} {
		w := mappedUpload(t, tc.path, "f.csv", tc.file, tc.fields)
		assert.Equal(t, tc.code, w.Code, name+": "+w.Body.String())
		assert.Contains(t, w.Body.String(), `"error":"`+tc.errKey+`"`, name)
	}
}

// Regression (F-23): size and row caps on the mapped import endpoints.
func TestImportUpload_Caps(t *testing.T) {
	big := bytes.Repeat([]byte("x"), int(imports.MaxFileBytes)+256<<10)
	w := mappedUpload(t, "/imports/customers/preview", "big.csv", append([]byte("email\n"), big...), nil)
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)

	var b strings.Builder
	b.WriteString("email\n")
	for i := 0; i <= imports.MaxRows; i++ {
		fmt.Fprintf(&b, "u%d@x.io\n", i)
	}
	w = mappedUpload(t, "/imports/customers", "rows.csv", []byte(b.String()), nil)
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	assert.Contains(t, w.Body.String(), "50000 rows")
}

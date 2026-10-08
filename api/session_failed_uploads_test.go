package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Example payload from the KSeF 2.0 OpenAPI spec for GET /sessions/{ref}/invoices/failed.
const failedInvoicesDuplicateJSON = `{
  "continuationToken": "",
  "invoices": [
    {
      "ordinalNumber": 2,
      "referenceNumber": "20250626-EE-2F20AD2000-242386DF86-52",
      "invoiceHash": "mkht+3m5trnfxlTYhq3QFn74LkEO69MFNlsMAkCDSPA=",
      "invoicingDate": "2025-07-11T12:23:56.0154302+00:00",
      "status": {
        "code": 440,
        "description": "Duplikat faktury",
        "details": [
          "Duplikat faktury. Faktura o numerze KSeF: 5265877635-20250626-010080DD2B5E-26 została już prawidłowo przesłana do systemu w sesji: 20250626-SO-2F14610000-242991F8C9-B4"
        ],
        "extensions": {
          "originalSessionReferenceNumber": "20250626-SO-2F14610000-242991F8C9-B4",
          "originalKsefNumber": "5265877635-20250626-010080DD2B5E-26"
        }
      }
    }
  ]
}`

// failedUploadsSession returns an authenticated session "SESSION-REF" whose
// failed-invoices endpoint answers with body.
func failedUploadsSession(t *testing.T, body string) *UploadSession {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/sessions/SESSION-REF/invoices/failed", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	client := &Client{clientOpts: clientOpts{
		client:      resty.New(),
		url:         srv.URL,
		accessToken: &apiToken{Token: "test-token", ValidUntil: time.Now().Add(time.Hour).Format(time.RFC3339Nano)},
	}}
	return &UploadSession{ReferenceNumber: "SESSION-REF", Client: client}
}

func TestGetFailedUploadDataDecodesExtensions(t *testing.T) {
	session := failedUploadsSession(t, failedInvoicesDuplicateJSON)

	failed, err := session.GetFailedUploadData(context.Background())
	require.NoError(t, err)
	require.Len(t, failed, 1)

	status := failed[0].Status
	require.NotNil(t, status)
	assert.Equal(t, InvoiceStatusDuplicate, status.Code)
	assert.Equal(t, "Duplikat faktury", status.Description)
	assert.Equal(t, "20250626-SO-2F14610000-242991F8C9-B4", status.Extensions[ExtensionOriginalSessionReferenceNumber])
	assert.Equal(t, "5265877635-20250626-010080DD2B5E-26", status.Extensions[ExtensionOriginalKsefNumber])
}

func TestGetFailedUploadDataWithoutExtensions(t *testing.T) {
	cases := map[string]string{
		"absent": `{"invoices":[{"ordinalNumber":1,"status":{"code":440,"description":"Duplikat faktury","details":[]}}]}`,
		"null":   `{"invoices":[{"ordinalNumber":1,"status":{"code":440,"description":"Duplikat faktury","extensions":null}}]}`,
		"null values": `{"invoices":[{"ordinalNumber":1,"status":{"code":440,"description":"Duplikat faktury",
			"extensions":{"originalSessionReferenceNumber":null,"originalKsefNumber":null}}}]}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			session := failedUploadsSession(t, body)

			failed, err := session.GetFailedUploadData(context.Background())
			require.NoError(t, err)
			require.Len(t, failed, 1)
			assert.Equal(t, InvoiceStatusDuplicate, failed[0].Status.Code)
			assert.Empty(t, failed[0].Status.Extensions[ExtensionOriginalSessionReferenceNumber])
			assert.Empty(t, failed[0].Status.Extensions[ExtensionOriginalKsefNumber])
		})
	}
}

func TestGetFailedUploadDataRequiresReferenceNumber(t *testing.T) {
	client := &Client{clientOpts: clientOpts{client: resty.New(), url: "http://127.0.0.1:1"}}
	session := &UploadSession{Client: client}

	_, err := session.GetFailedUploadData(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing reference number")
}

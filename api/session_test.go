package api_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"testing"
	"time"

	ksef_api "github.com/invopop/gobl.pl.ksef/api"
	"github.com/invopop/gobl.pl.ksef/test"
	"github.com/invopop/xmldsig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateSession(t *testing.T) {
	t.Run("creates session", func(t *testing.T) {
		cert, err := xmldsig.LoadCertificate("./test/cert-20260102-131809.pfx", "")
		require.NoError(t, err)

		client := ksef_api.NewClient(
			&ksef_api.ContextIdentifier{Nip: "8126178616"},
			cert,
		)

		ctx := context.Background()
		err = client.Authenticate(ctx)
		require.NoError(t, err)

		uploadSession, err := client.CreateSession(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, uploadSession.ReferenceNumber)
		assert.NotEmpty(t, uploadSession.ValidUntil)
		assert.Len(t, uploadSession.SymmetricKey, 32)
		assert.Len(t, uploadSession.InitializationVector, 16)

		err = uploadSession.FinishUpload(ctx)
		assert.NoError(t, err)
	})
}

func TestUploadInvoice(t *testing.T) {
	t.Run("uploads invoice during session", func(t *testing.T) {
		ctxIdentifier := &ksef_api.ContextIdentifier{Nip: "8126178616"}
		cert, err := xmldsig.LoadCertificate("./test/cert-20260102-131809.pfx", "")
		require.NoError(t, err)

		client := ksef_api.NewClient(
			ctxIdentifier,
			cert,
		)

		ctx := context.Background()
		err = client.Authenticate(ctx)
		require.NoError(t, err)

		uploadSession, err := client.CreateSession(ctx)
		require.NoError(t, err)

		doc, err := test.BuildFAVATFrom("invoice-standard.json")
		require.NoError(t, err)

		// Update seller NIP to match the authenticated context
		doc.Seller.NIP = ctxIdentifier.Nip

		// Generate unique identifier for the invoice.
		// Without it, uploading will result in error because of a duplicate.
		now := time.Now().UTC()
		doc.Inv.IssueDate = now.Format("2006-01-02")             // current date
		doc.Inv.SequentialNumber = fmt.Sprintf("%d", now.Unix()) // Unix timestamp in seconds

		invoiceBytes, err := doc.Bytes()
		require.NoError(t, err)

		err = uploadSession.UploadInvoice(ctx, invoiceBytes)
		require.NoError(t, err)

		err = uploadSession.FinishUpload(ctx)
		assert.NoError(t, err)

		_, err = uploadSession.PollStatus(ctx)
		assert.NoError(t, err)

		uploadedInvoices, err := uploadSession.ListUploadedInvoices(ctx)
		assert.NoError(t, err)
		assert.Len(t, uploadedInvoices, 1)

		// For debugging - we should not get any failed uploads, but if an upload fails, we should get more information about what exactly went wrong
		failedUploads, err := uploadSession.GetFailedUploadData(ctx)
		assert.NoError(t, err)
		for _, inv := range failedUploads {
			fmt.Printf("Failed invoice %s (ordinal %d): %+v\n", inv.ReferenceNumber, inv.OrdinalNumber, inv.Status)
		}

		hashBytes, err := base64.StdEncoding.DecodeString(uploadedInvoices[0].InvoiceHash)
		require.NoError(t, err)

		// Attach required stamps to envelope
		qrURL, err := ksef_api.GenerateQrCodeURL(
			ksef_api.EnvironmentTest,
			ctxIdentifier.Nip,
			uploadedInvoices[0].InvoicingDate,
			hashBytes,
		)
		require.NoError(t, err)

		// Check if the URL is correctly formed
		// IMPORTANT: when the URL contains invalid parameters (e.g. NIP is different), the response is still 200,
		// but the website content says that "no invoice found".
		// To check if the URL is actually valid, we need to check the returned HTML, and this is very fragile
		resp, err := http.Get(qrURL)
		require.NoError(t, err)
		defer func() {
			assert.NoError(t, resp.Body.Close())
		}()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

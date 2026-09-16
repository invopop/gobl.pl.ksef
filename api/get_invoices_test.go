package api_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	ksef_api "github.com/invopop/gobl.pl.ksef/api"
	"github.com/invopop/xmldsig"
	"github.com/stretchr/testify/require"
)

func TestListInvoices(t *testing.T) {
	t.Run("lists uploaded invoices in the last 14 days", func(t *testing.T) {
		cert, err := xmldsig.LoadCertificate("./test/cert-20260102-131809.pfx", "")
		require.NoError(t, err)

		client := ksef_api.NewClient(
			&ksef_api.ContextIdentifier{Nip: "8126178616"},
			cert,
			ksef_api.WithDebugClient(),
		)

		ctx := context.Background()
		require.NoError(t, client.Authenticate(ctx))

		today := time.Now().UTC()
		to := today
		params := ksef_api.ListInvoicesParams{
			SubjectType: ksef_api.InvoiceSubjectTypeSupplier,
			From:        today.AddDate(0, 0, -14),
			To:          &to,
		}

		_, err = client.ListInvoices(ctx, params)
		require.NoError(t, err)
	})
}

func TestGetInvoice(t *testing.T) {
	t.Run("fetches invoice by ksef number", func(t *testing.T) {
		cert, err := xmldsig.LoadCertificate("./test/cert-20260102-131809.pfx", "")
		require.NoError(t, err)

		client := ksef_api.NewClient(
			&ksef_api.ContextIdentifier{Nip: "8126178616"},
			cert,
			ksef_api.WithDebugClient(),
		)

		ctx := context.Background()
		require.NoError(t, client.Authenticate(ctx))

		ksefNumber := "8126178616-20260117-010020CE337D-CD"
		xmlData, err := client.GetInvoice(ctx, ksefNumber)
		require.NoError(t, err)

		fmt.Printf("Invoice XML for %s:\n%s\n", ksefNumber, string(xmlData))
	})
}

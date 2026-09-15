package api_test

import (
	"context"
	"testing"

	ksef_api "github.com/invopop/gobl.pl.ksef/api"
	"github.com/invopop/xmldsig"
	"github.com/stretchr/testify/assert"
)

func TestAuthentication(t *testing.T) {
	t.Run("should authenticate to API", func(t *testing.T) {
		cert, err := xmldsig.LoadCertificate("./test/cert-20260102-131809.pfx", "")
		assert.NoError(t, err)

		client := ksef_api.NewClient(
			&ksef_api.ContextIdentifier{Nip: "8126178616"},
			cert,
			ksef_api.WithDebugClient(),
		)
		ctx := context.Background()
		err = client.Authenticate(ctx)
		assert.NoError(t, err)
	})
}

// Package api_test provides tools for testing the api
// TODO: this file will be temporarily unused, for now the tests will be querying the real API

package api_test

import (
	"context"

	ksef_api "github.com/invopop/gobl.pl.ksef/api"
	"github.com/invopop/xmldsig"
	//"github.com/jarcoal/httpmock"
)

// Client creates authorized client for testing
func Client() (*ksef_api.Client, error) {
	// mockClient := resty.New()

	// httpmock.ActivateNonDefault(mockClient.GetClient())

	// reqT, err := time.Parse("2006-01-02T15:04:05.000Z", "2024-01-26T16:18:51.701Z")
	// if err != nil {
	// 	return nil, err
	// }

	// httpmock.RegisterResponder("POST", "https://ksef-test.mf.gov.pl/api/online/Session/AuthorisationChallenge",
	// 	httpmock.NewJsonResponderOrPanic(200, &ksef_api.AuthorizationChallengeResponse{Timestamp: reqT, Challenge: "20240126-CR-077CAFEC31-83ACAC25E4-64"}))

	// sessionToken := "exampleSessionToken"
	// httpmock.RegisterResponder("POST", "https://ksef-test.mf.gov.pl/api/online/Session/InitToken",
	// 	httpmock.NewJsonResponderOrPanic(200, &ksef_api.InitSessionTokenResponse{ReferenceNumber: "ExampleReferenceNumber", SessionToken: &ksef_api.SessionToken{Token: sessionToken}}))

	cert, err := xmldsig.LoadCertificate("api/test/cert-20260102-131809.pfx", "")
	if err != nil {
		return nil, err
	}

	client := ksef_api.NewClient(
		&ksef_api.ContextIdentifier{Nip: "8126178616"},
		cert,
	)

	ctx := context.Background()
	err = client.Authenticate(ctx)
	if err != nil {
		return nil, err
	}

	return client, nil
}

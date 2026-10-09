package api

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCSR(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	data := &CertificateEnrollmentData{
		CommonName:             "Firma Example Cert",
		CountryName:            "PL",
		OrganizationName:       "Firma Example Sp. z o.o.",
		SerialNumber:           "ABC123456",
		Surname:                "Kowalski",
		GivenName:              "Jan",
		UniqueIdentifier:       "123e4567-e89b-12d3-a456-426614174000",
		OrganizationIdentifier: "1234567890",
	}

	csrBase64, err := data.GenerateCSR(key)
	require.NoError(t, err)

	csrDER, err := base64.StdEncoding.DecodeString(csrBase64)
	require.NoError(t, err)

	csr, err := x509.ParseCertificateRequest(csrDER)
	require.NoError(t, err)
	require.NoError(t, csr.CheckSignature())

	assert.Equal(t, data.CommonName, csr.Subject.CommonName)
	assert.Equal(t, []string{data.CountryName}, csr.Subject.Country)
	assert.Equal(t, []string{data.OrganizationName}, csr.Subject.Organization)
	assert.Equal(t, data.SerialNumber, csr.Subject.SerialNumber)

	attrValue := func(oid string) string {
		for _, attr := range csr.Subject.Names {
			if attr.Type.String() == oid {
				if str, ok := attr.Value.(string); ok {
					return str
				}
			}
		}
		return ""
	}

	assert.Equal(t, data.Surname, attrValue(oidSurname.String()))
	assert.Equal(t, data.GivenName, attrValue(oidGivenName.String()))
	assert.Equal(t, data.UniqueIdentifier, attrValue(oidUniqueIdentifier.String()))
	assert.Equal(t, data.OrganizationIdentifier, attrValue(oidOrganizationIdentifier.String()))
}

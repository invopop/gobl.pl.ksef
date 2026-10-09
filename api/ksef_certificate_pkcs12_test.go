package api

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"software.sslmate.com/src/go-pkcs12"
)

func TestBuildPKCS12Certificate(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(123456789),
		Subject:               pkix.Name{CommonName: "Test Cert"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	entry := &CertificateRetrieveEntry{
		Certificate:             base64.StdEncoding.EncodeToString(certDER),
		CertificateName:         "Test",
		CertificateSerialNumber: "123456",
		CertificateType:         "Authentication",
	}

	password := "secret"
	pfx, err := BuildPKCS12Certificate(entry, key, password)
	require.NoError(t, err)
	require.NotEmpty(t, pfx)

	decodedKey, cert, _, err := pkcs12.DecodeChain(pfx, password)
	require.NoError(t, err)
	require.NotNil(t, cert)

	decodedECDSAKey, ok := decodedKey.(*ecdsa.PrivateKey)
	require.True(t, ok)
	require.Equal(t, key.D, decodedECDSAKey.D)
}

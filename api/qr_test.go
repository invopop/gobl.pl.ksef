package api

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateQrCodeURL(t *testing.T) {
	invoiceHash := []byte("hash1234567890") // Example hash

	t.Run("production", func(t *testing.T) {
		date, _ := time.Parse("2006-01-02", "2023-10-25")
		url, err := GenerateQrCodeURL(EnvironmentProduction, "1234567890", date, invoiceHash)
		assert.NoError(t, err)
		assert.Equal(t, "https://qr.ksef.mf.gov.pl/invoice/1234567890/25-10-2023/aGFzaDEyMzQ1Njc4OTA", url)
	})

	t.Run("test", func(t *testing.T) {
		date, _ := time.Parse("2006-01-02", "2023-10-25")
		url, err := GenerateQrCodeURL(EnvironmentTest, "1234567890", date, invoiceHash)
		assert.NoError(t, err)
		assert.Equal(t, "https://qr-test.ksef.mf.gov.pl/invoice/1234567890/25-10-2023/aGFzaDEyMzQ1Njc4OTA", url)
	})
}

func TestGenerateCertificateQrCodeURL(t *testing.T) {
	invoiceHash := []byte{0x00, 0x01, 0x02, 0x03}
	// Base64 RawURL of 00010203 -> AAECAw

	t.Run("test environment", func(t *testing.T) {
		url, err := GenerateUnsignedCertificateQrCodeURL(
			EnvironmentTest,
			"1111111111", // contextNip
			"2222222222", // sellerNip
			"CERT123",    // certificateId
			invoiceHash,
		)
		assert.NoError(t, err)
		assert.Equal(t, "qr-test.ksef.mf.gov.pl/certificate/Nip/1111111111/2222222222/CERT123/AAECAw", url)
	})

	t.Run("production environment", func(t *testing.T) {
		url, err := GenerateUnsignedCertificateQrCodeURL(
			EnvironmentProduction,
			"1111111111",
			"2222222222",
			"CERT123",
			invoiceHash,
		)
		assert.NoError(t, err)
		assert.Equal(t, "qr.ksef.mf.gov.pl/certificate/Nip/1111111111/2222222222/CERT123/AAECAw", url)
	})

	t.Run("empty args", func(t *testing.T) {
		_, err := GenerateUnsignedCertificateQrCodeURL(EnvironmentTest, "", "222", "C", invoiceHash)
		assert.Error(t, err)
		_, err = GenerateUnsignedCertificateQrCodeURL(EnvironmentTest, "111", "", "C", invoiceHash)
		assert.Error(t, err)
		_, err = GenerateUnsignedCertificateQrCodeURL(EnvironmentTest, "111", "222", "", invoiceHash)
		assert.Error(t, err)
	})
}

func TestGenerateSignedCertificateQrCodeURL_ECDSA(t *testing.T) {
	unsignedURL := "qr.ksef.mf.gov.pl/certificate/Nip/1111111111/2222222222/CERT123/AAECAw"
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	signedURL, err := GenerateSignedCertificateQrCodeURL(unsignedURL, key)
	require.NoError(t, err)

	prefix := "https://" + unsignedURL + "/"
	if !assert.True(t, strings.HasPrefix(signedURL, prefix)) {
		return
	}

	sigEncoded := strings.TrimPrefix(signedURL, prefix)
	signature, err := base64.RawURLEncoding.DecodeString(sigEncoded)
	require.NoError(t, err)

	size := (key.Params().BitSize + 7) / 8
	require.Equal(t, 2*size, len(signature))

	r := new(big.Int).SetBytes(signature[:size])
	s := new(big.Int).SetBytes(signature[size:])
	hash := sha256.Sum256([]byte(unsignedURL))
	assert.True(t, ecdsa.Verify(&key.PublicKey, hash[:], r, s))
}

func TestGenerateSignedCertificateQrCodeURL_RSA(t *testing.T) {
	unsignedURL := "qr.ksef.mf.gov.pl/certificate/Nip/3333333333/4444444444/CERT999/BBEFAA"
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	signedURL, err := GenerateSignedCertificateQrCodeURL(unsignedURL, key)
	require.NoError(t, err)

	prefix := "https://" + unsignedURL + "/"
	if !assert.True(t, strings.HasPrefix(signedURL, prefix)) {
		return
	}

	sigEncoded := strings.TrimPrefix(signedURL, prefix)
	signature, err := base64.RawURLEncoding.DecodeString(sigEncoded)
	require.NoError(t, err)

	hash := sha256.Sum256([]byte(unsignedURL))
	assert.NoError(t, rsa.VerifyPSS(&key.PublicKey, crypto.SHA256, hash[:], signature, &rsa.PSSOptions{SaltLength: 32, Hash: crypto.SHA256}))
}

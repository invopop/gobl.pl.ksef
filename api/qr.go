package api

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"
)

const (
	EnvironmentProductionQrUrl = "qr.ksef.mf.gov.pl"
	EnvironmentDemoQrUrl       = "qr-demo.ksef.mf.gov.pl"
	EnvironmentTestQrUrl       = "qr-test.ksef.mf.gov.pl"
)

// GenerateQrCodeURL builds the verification URL for an invoice, both in online and offline mode.
func GenerateQrCodeURL(environment Environment, nip string, invoicingDate time.Time, invoiceHash []byte) (string, error) {
	var baseUrl string
	switch environment {
	case EnvironmentProduction:
		baseUrl = EnvironmentProductionQrUrl
	case EnvironmentDemo:
		baseUrl = EnvironmentDemoQrUrl
	case EnvironmentTest:
		baseUrl = EnvironmentTestQrUrl
	default:
		return "", fmt.Errorf("invalid environment: %s", environment)
	}

	if nip == "" {
		return "", fmt.Errorf("nip is empty")
	}

	base64URLHash := base64.RawURLEncoding.EncodeToString(invoiceHash)

	return fmt.Sprintf("https://%s/invoice/%s/%s/%s",
		baseUrl,
		nip,
		invoicingDate.Format("02-01-2006"),
		base64URLHash,
	), nil
}

// GenerateUnsignedCertificateQrCodeURL builds the unsigned certificate verification URL for an offline invoice.
// Then, the URL must be signed with GenerateSignedCertificateQrCodeURL function.
func GenerateUnsignedCertificateQrCodeURL(environment Environment, contextNip string, sellerNip string, certificateId string, invoiceHash []byte) (string, error) {
	var baseUrl string
	switch environment {
	case EnvironmentProduction:
		baseUrl = EnvironmentProductionQrUrl
	case EnvironmentDemo:
		baseUrl = EnvironmentDemoQrUrl
	case EnvironmentTest:
		baseUrl = EnvironmentTestQrUrl
	default:
		return "", fmt.Errorf("invalid environment: %s", environment)
	}

	if contextNip == "" {
		return "", fmt.Errorf("contextNip is empty")
	}
	if sellerNip == "" {
		return "", fmt.Errorf("sellerNip is empty")
	}
	if certificateId == "" {
		return "", fmt.Errorf("certificateId is empty")
	}

	base64InvoiceHash := base64.RawURLEncoding.EncodeToString(invoiceHash)

	return fmt.Sprintf("%s/certificate/Nip/%s/%s/%s/%s",
		baseUrl,
		contextNip,
		sellerNip,
		certificateId,
		base64InvoiceHash,
	), nil
}

// GenerateSignedCertificateQrCodeURL creates a signed URL to be shown as QR code, for certificate verification of offline invoices.
// Private key must come from a KSeF offline certificate (important!). Example how to obtain it:
// privateKey, _, _, err := pkcs12.DecodeChain(certificateData, certificatePassword)
func GenerateSignedCertificateQrCodeURL(unsignedUrl string, privateKey crypto.Signer) (string, error) {
	urlHash := sha256.Sum256([]byte(unsignedUrl)) // KSeF requires SHA-256

	var signature []byte
	var err error

	// KSeF requires signature in the following format:
	// - For ECDSA: IEEE P1363 format (r || s)
	// - For RSA: PSS signature
	switch key := privateKey.(type) {
	case *ecdsa.PrivateKey:
		r, s, err := ecdsa.Sign(rand.Reader, key, urlHash[:])
		if err != nil {
			return "", err
		}
		// IEEE P1363
		size := (key.Curve.Params().BitSize + 7) / 8 // 32 for P-256

		rb := r.FillBytes(make([]byte, size))
		sb := s.FillBytes(make([]byte, size))

		signature = append(rb, sb...)
	case *rsa.PrivateKey:
		signature, err = key.Sign(rand.Reader, urlHash[:], &rsa.PSSOptions{SaltLength: 32, Hash: crypto.SHA256})
		if err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("certificate private key must be ECDSA or RSA")
	}

	signatureBase64 := base64.RawURLEncoding.EncodeToString(signature)

	return fmt.Sprintf("https://%s/%s", unsignedUrl, signatureBase64), nil
}

package api

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"fmt"
)

var (
	oidSurname                = asn1.ObjectIdentifier{2, 5, 4, 4}
	oidGivenName              = asn1.ObjectIdentifier{2, 5, 4, 42}
	oidUniqueIdentifier       = asn1.ObjectIdentifier{2, 5, 4, 45}
	oidOrganizationIdentifier = asn1.ObjectIdentifier{2, 5, 4, 97}
)

// GenerateCSR builds a PKCS#10 certificate signing request encoded in Base64.
// The private key must be EC (secp256r1) and match the public key that will be embedded in the CSR.
// API documentation says that both RSA and EC keys are supported, but EC is recommended.
func (d *CertificateEnrollmentData) GenerateCSR(privateKey *ecdsa.PrivateKey) (string, error) {
	if d == nil {
		return "", fmt.Errorf("certificate enrollment data is nil")
	}
	if privateKey == nil {
		return "", fmt.Errorf("private key is nil")
	}
	if privateKey.Curve != elliptic.P256() {
		return "", fmt.Errorf("unsupported EC curve: expected P-256")
	}

	subject, err := d.asPkixName()
	if err != nil {
		return "", err
	}

	template := &x509.CertificateRequest{
		Subject:            subject,
		SignatureAlgorithm: x509.ECDSAWithSHA256,
	}

	csr, err := x509.CreateCertificateRequest(rand.Reader, template, privateKey)
	if err != nil {
		return "", fmt.Errorf("create certificate request: %w", err)
	}

	return base64.StdEncoding.EncodeToString(csr), nil
}

func (d *CertificateEnrollmentData) asPkixName() (pkix.Name, error) {
	if d.CommonName == "" {
		return pkix.Name{}, fmt.Errorf("commonName is empty")
	}
	if d.CountryName == "" {
		return pkix.Name{}, fmt.Errorf("countryName is empty")
	}

	name := pkix.Name{
		CommonName:   d.CommonName,
		SerialNumber: d.SerialNumber,
	}

	if d.CountryName != "" {
		name.Country = []string{d.CountryName}
	}
	if d.OrganizationName != "" {
		name.Organization = []string{d.OrganizationName}
	}

	name.ExtraNames = appendAttribute(name.ExtraNames, oidSurname, d.Surname)
	name.ExtraNames = appendAttribute(name.ExtraNames, oidGivenName, d.GivenName)
	name.ExtraNames = appendAttribute(name.ExtraNames, oidUniqueIdentifier, d.UniqueIdentifier)
	name.ExtraNames = appendAttribute(name.ExtraNames, oidOrganizationIdentifier, d.OrganizationIdentifier)

	return name, nil
}

func appendAttribute(attrs []pkix.AttributeTypeAndValue, oid asn1.ObjectIdentifier, value string) []pkix.AttributeTypeAndValue {
	if value == "" {
		return attrs
	}
	return append(attrs, pkix.AttributeTypeAndValue{
		Type:  oid,
		Value: value,
	})
}

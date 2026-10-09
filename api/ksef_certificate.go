package api

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"time"

	"software.sslmate.com/src/go-pkcs12"
)

// CertificateEnrollmentData stores the subject data required to prepare a certificate request.
type CertificateEnrollmentData struct {
	CommonName             string `json:"commonName"`
	Surname                string `json:"surname,omitempty"`
	SerialNumber           string `json:"serialNumber,omitempty"`
	CountryName            string `json:"countryName"`
	OrganizationName       string `json:"organizationName,omitempty"`
	GivenName              string `json:"givenName,omitempty"`
	UniqueIdentifier       string `json:"uniqueIdentifier,omitempty"`
	OrganizationIdentifier string `json:"organizationIdentifier,omitempty"`
}

// CertificateType identifies the target certificate flavor.
type CertificateType string

const (
	CertificateTypeAuthentication CertificateType = "Authentication"
	CertificateTypeOffline        CertificateType = "Offline"
)

var (
	ErrCertificateEnrollmentPollingCountExceeded = errors.New("certificate enrollment polling count exceeded")
	ErrCertificateEnrollmentFailed               = errors.New("certificate enrollment failed")
)

func (t CertificateType) isValid() bool {
	switch t {
	case CertificateTypeAuthentication, CertificateTypeOffline:
		return true
	default:
		return false
	}
}

type certificateEnrollmentRequest struct {
	CertificateName string          `json:"certificateName"`
	CertificateType CertificateType `json:"certificateType"`
	CSR             string          `json:"csr"`
	ValidFrom       *time.Time      `json:"validFrom,omitempty"`
}

type certificateRetrieveRequest struct {
	CertificateSerialNumbers []string `json:"certificateSerialNumbers"`
}

// CertificateEnrollmentResponse stores the response metadata returned after submitting a CSR.
type CertificateEnrollmentResponse struct {
	ReferenceNumber string `json:"referenceNumber"`
	Timestamp       string `json:"timestamp"`
}

// CertificateEnrollmentStatusResponse describes the status returned for an enrollment reference number.
type CertificateEnrollmentStatusResponse struct {
	RequestDate             time.Time                    `json:"requestDate"`
	Status                  *CertificateEnrollmentStatus `json:"status"`
	CertificateSerialNumber string                       `json:"certificateSerialNumber,omitempty"`
}

// CertificateEnrollmentStatus mirrors the StatusInfo structure returned by the API.
type CertificateEnrollmentStatus struct {
	Code        int      `json:"code"`
	Description string   `json:"description"`
	Details     []string `json:"details,omitempty"`
}

// CertificateRetrieveEntry represents a single certificate returned by /certificates/retrieve.
type CertificateRetrieveEntry struct {
	Certificate             string `json:"certificate"`
	CertificateName         string `json:"certificateName"`
	CertificateSerialNumber string `json:"certificateSerialNumber"`
	CertificateType         string `json:"certificateType"`
}

// CertificateRetrieveResponse mirrors the API response for POST /certificates/retrieve.
type CertificateRetrieveResponse struct {
	Certificates []CertificateRetrieveEntry `json:"certificates"`
}

// GetCertificateEnrollmentData returns the identification data used when building a new CSR.
func (c *Client) GetCertificateEnrollmentData(ctx context.Context) (*CertificateEnrollmentData, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	response := &CertificateEnrollmentData{}
	resp, err := c.client.R().
		SetContext(ctx).
		SetAuthToken(token).
		SetResult(response).
		Get(c.url + "/certificates/enrollments/data")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, newErrorResponse(resp)
	}

	return response, nil
}

// EnrollCertificate submits the generated CSR and returns reference metadata.
// validFrom is optional, if not provided, the certificate will be valid from the current date.
func (c *Client) EnrollCertificate(ctx context.Context, certificateName string, certificateType CertificateType, csr string, validFrom *time.Time) (*CertificateEnrollmentResponse, error) {
	if certificateName == "" {
		return nil, fmt.Errorf("certificateName is required")
	}
	if !certificateType.isValid() {
		return nil, fmt.Errorf("certificateType is invalid: %s", certificateType)
	}
	if csr == "" {
		return nil, fmt.Errorf("csr is required")
	}

	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	request := &certificateEnrollmentRequest{
		CertificateName: certificateName,
		CertificateType: certificateType,
		CSR:             csr,
		ValidFrom:       validFrom,
	}

	response := &CertificateEnrollmentResponse{}
	resp, err := c.client.R().
		SetContext(ctx).
		SetAuthToken(token).
		SetBody(request).
		SetResult(response).
		Post(c.url + "/certificates/enrollments")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, newErrorResponse(resp)
	}

	return response, nil
}

// RetrieveCertificate downloads metadata and content for the provided certificate serial number.
func (c *Client) RetrieveCertificate(ctx context.Context, certificateSerialNumber string) (*CertificateRetrieveEntry, error) {
	if certificateSerialNumber == "" {
		return nil, fmt.Errorf("certificate serial number is required")
	}

	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	request := &certificateRetrieveRequest{
		CertificateSerialNumbers: []string{certificateSerialNumber},
	}

	response := &CertificateRetrieveResponse{}
	resp, err := c.client.R().
		SetContext(ctx).
		SetAuthToken(token).
		SetBody(request).
		SetResult(response).
		Post(c.url + "/certificates/retrieve")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, newErrorResponse(resp)
	}
	if len(response.Certificates) != 1 {
		return nil, fmt.Errorf("expected exactly 1 certificate, got %d", len(response.Certificates))
	}

	return &response.Certificates[0], nil
}

// CreateKsefCertificate orchestrates the full flow of requesting a KSeF certificate using the provided key material.
// Returns the retrieved certificate response once the request succeeds.
func (c *Client) CreateKsefCertificate(ctx context.Context, certificateName string, certificateType CertificateType, privateKey *ecdsa.PrivateKey, validFrom *time.Time) (*CertificateRetrieveEntry, error) {
	if privateKey == nil {
		return nil, fmt.Errorf("private key is required")
	}

	enrollmentData, err := c.GetCertificateEnrollmentData(ctx)
	if err != nil {
		return nil, err
	}

	csr, err := enrollmentData.GenerateCSR(privateKey)
	if err != nil {
		return nil, err
	}

	enrollmentResp, err := c.EnrollCertificate(ctx, certificateName, certificateType, csr, validFrom)
	if err != nil {
		return nil, err
	}

	statusResp, err := c.PollCertificateEnrollmentStatus(ctx, enrollmentResp.ReferenceNumber)
	if err != nil {
		return nil, err
	}

	if statusResp.CertificateSerialNumber == "" {
		return nil, fmt.Errorf("certificate serial number missing in enrollment status response")
	}
	return c.RetrieveCertificate(ctx, statusResp.CertificateSerialNumber)
}

// BuildPKCS12Certificate is a helper that bundles the retrieved certificate and the private key into a PKCS#12 archive.
// Note: there are multiple possible encodings of PKCS#12 (LegacyRC2, Legacy, Modern). LegacyRC2 is insecure, and Modern may be not compatible with older systems.
func BuildPKCS12Certificate(entry *CertificateRetrieveEntry, privateKey *ecdsa.PrivateKey, password string) ([]byte, error) {
	if entry == nil {
		return nil, fmt.Errorf("certificate entry is nil")
	}
	if privateKey == nil {
		return nil, fmt.Errorf("private key is nil")
	}

	certBytes, err := base64.StdEncoding.DecodeString(entry.Certificate)
	if err != nil {
		return nil, fmt.Errorf("decode certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}

	pfx, err := pkcs12.Legacy.Encode(privateKey, cert, nil, password)
	if err != nil {
		return nil, fmt.Errorf("encode pkcs12: %w", err)
	}

	return pfx, nil
}

// RevokeCertificate submits a revocation request for the provided certificate serial number.
func (c *Client) RevokeCertificate(ctx context.Context, certificateSerialNumber string) error {
	if certificateSerialNumber == "" {
		return fmt.Errorf("certificateSerialNumber is required")
	}

	token, err := c.getAccessToken(ctx)
	if err != nil {
		return err
	}

	resp, err := c.client.R().
		SetContext(ctx).
		SetAuthToken(token).
		Post(c.url + "/certificates/" + url.PathEscape(certificateSerialNumber) + "/revoke")
	if err != nil {
		return err
	}
	if resp.IsError() {
		return newErrorResponse(resp)
	}

	return nil
}

// GetCertificateEnrollmentStatus returns processing status for the specified certificate request reference number.
func (c *Client) GetCertificateEnrollmentStatus(ctx context.Context, referenceNumber string) (*CertificateEnrollmentStatusResponse, error) {
	if referenceNumber == "" {
		return nil, fmt.Errorf("referenceNumber is required")
	}

	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	response := &CertificateEnrollmentStatusResponse{}
	resp, err := c.client.R().
		SetContext(ctx).
		SetAuthToken(token).
		SetResult(response).
		Get(c.url + "/certificates/enrollments/" + url.PathEscape(referenceNumber))
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, newErrorResponse(resp)
	}

	return response, nil
}

// PollCertificateEnrollmentStatus keeps querying enrollment status until it succeeds or fails.
func (c *Client) PollCertificateEnrollmentStatus(ctx context.Context, referenceNumber string) (*CertificateEnrollmentStatusResponse, error) {
	attempt := 0
	for {
		attempt++
		if attempt > 30 {
			return nil, ErrCertificateEnrollmentPollingCountExceeded
		}

		statusResp, err := c.GetCertificateEnrollmentStatus(ctx, referenceNumber)
		if err != nil {
			return nil, err
		}
		if statusResp == nil || statusResp.Status == nil {
			return nil, fmt.Errorf("certificate enrollment status response missing status")
		}

		switch statusResp.Status.Code {
		case 200:
			return statusResp, nil
		case 100:
			time.Sleep(2 * time.Second)
			continue
		default:
			return nil, fmt.Errorf("%w: %s", ErrCertificateEnrollmentFailed, statusResp.Status.Description)
		}
	}
}

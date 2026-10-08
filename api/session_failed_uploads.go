package api

import (
	"context"
	"fmt"
)

const (
	// InvoiceStatusDuplicate is the invoice status code KSeF reports when an
	// invoice with the same seller NIP, invoice type and number was already
	// accepted ("Duplikat faktury").
	InvoiceStatusDuplicate = 440

	// ExtensionOriginalSessionReferenceNumber is the extensions key carrying the
	// reference number of the session in which the original invoice was accepted.
	ExtensionOriginalSessionReferenceNumber = "originalSessionReferenceNumber"
	// ExtensionOriginalKsefNumber is the extensions key carrying the KSeF number
	// assigned to the original invoice.
	ExtensionOriginalKsefNumber = "originalKsefNumber"
)

// FailedUploadInvoiceStatus describes the status payload for invoices that failed to upload.
type FailedUploadInvoiceStatus struct {
	Code        int      `json:"code"`
	Description string   `json:"description"`
	Details     []string `json:"details"`
	// Extensions carries status-specific data. For InvoiceStatusDuplicate it
	// holds ExtensionOriginalSessionReferenceNumber and ExtensionOriginalKsefNumber.
	Extensions map[string]string `json:"extensions,omitempty"`
}

// FailedUploadInvoice contains a single failed invoice entry returned by the API.
type FailedUploadInvoice struct {
	OrdinalNumber   int                        `json:"ordinalNumber"`
	ReferenceNumber string                     `json:"referenceNumber"`
	InvoiceHash     string                     `json:"invoiceHash"`
	Status          *FailedUploadInvoiceStatus `json:"status"`
}

type failedUploadInvoicesResponse struct {
	ContinuationToken string                `json:"continuationToken"`
	Invoices          []FailedUploadInvoice `json:"invoices"`
}

// GetFailedUploadData lists invoices that failed during upload for the session, following continuation tokens if needed.
func (s *UploadSession) GetFailedUploadData(ctx context.Context) ([]FailedUploadInvoice, error) {
	if s == nil {
		return nil, fmt.Errorf("upload session is nil")
	}
	if s.ReferenceNumber == "" {
		return nil, fmt.Errorf("upload session missing reference number")
	}

	c, err := s.clientForRequests()
	if err != nil {
		return nil, err
	}

	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	var (
		allInvoices       []FailedUploadInvoice
		continuationToken string
	)

	for {
		response := &failedUploadInvoicesResponse{}

		req := c.client.R().
			SetContext(ctx).
			SetAuthToken(token).
			SetResult(response)
		if continuationToken != "" {
			req.SetHeader("x-continuation-token", continuationToken)
		}

		resp, err := req.Get(c.url + "/sessions/" + s.ReferenceNumber + "/invoices/failed")
		if err != nil {
			return nil, err
		}
		if resp.IsError() {
			return nil, newErrorResponse(resp)
		}

		allInvoices = append(allInvoices, response.Invoices...)

		if response.ContinuationToken == "" {
			break
		}
		continuationToken = response.ContinuationToken
	}

	return allInvoices, nil
}

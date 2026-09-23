package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
)

// ErrorResponse parses error responses
type ErrorResponse struct {
	Exception struct {
		ServiceCtx          string `json:"serviceCtx"`
		ServiceCode         string `json:"serviceCode"`
		ServiceName         string `json:"serviceName"`
		Timestamp           string `json:"timestamp"`
		ReferenceNumber     string `json:"referenceNumber"`
		ExceptionDetailList []struct {
			ExceptionCode        int    `json:"exceptionCode"`
			ExceptionDescription string `json:"exceptionDescription"`
		} `json:"exceptionDetailList"`
	} `json:"exception"`
	// Status is the shape KSeF uses for some responses, such as 429 Too Many Requests.
	Status struct {
		Code        int      `json:"code"`
		Description string   `json:"description"`
		Details     []string `json:"details"`
	} `json:"status"`
}

// Error implements the error interface
func (e ErrorResponse) Error() string {
	msgs := make([]string, 0, len(e.Exception.ExceptionDetailList)+len(e.Status.Details))
	for _, detail := range e.Exception.ExceptionDetailList {
		msgs = append(msgs, fmt.Sprintf("Code %d: %s", detail.ExceptionCode, detail.ExceptionDescription))
	}
	msgs = append(msgs, e.Status.Details...)

	return strings.Join(msgs, ", ")
}

// ServiceError wraps an HTTP error response from the KSeF API,
// preserving the status code and Retry-After header for callers to inspect.
type ServiceError struct {
	StatusCode int
	Status     string
	RetryAfter int32 // seconds, from Retry-After header (0 if not present)
	Err        error // underlying ErrorResponse or raw body message
}

// Error implements the error interface.
func (e *ServiceError) Error() string {
	msg := fmt.Sprintf("KSeF service error response (Status %s)", e.Status)
	if e.Err != nil {
		return fmt.Sprintf("%s: %s", msg, e.Err.Error())
	}
	return msg
}

// Unwrap returns the underlying error for use with errors.Is/As.
func (e *ServiceError) Unwrap() error {
	return e.Err
}

// IsTransient returns true if this is a transient error (429 rate limit or 5xx server error).
func (e *ServiceError) IsTransient() bool {
	return e.StatusCode == 429 || e.StatusCode >= 500
}

// IsTransient reports whether err is a transient KSeF API failure that should
// be retried rather than treated as a permanent failure. This covers both
// service-side conditions (429 rate limit, 5xx) carried by *ServiceError and
// transport-level conditions (request deadline exceeded, cancellation, other
// network timeouts) that never produce an HTTP response.
func IsTransient(err error) bool {
	if err == nil {
		return false
	}
	var se *ServiceError
	if errors.As(err, &se) {
		return se.IsTransient()
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return false
}

func newErrorResponse(resp *resty.Response) error {
	se := &ServiceError{
		StatusCode: resp.StatusCode(),
		Status:     resp.Status(),
	}

	// Parse Retry-After header (delta-seconds as per KSeF docs).
	if ra := resp.Header().Get("Retry-After"); ra != "" {
		if v, err := strconv.ParseInt(ra, 10, 32); err == nil && v > 0 {
			se.RetryAfter = int32(v)
		}
	}

	if resp.StatusCode() >= 500 {
		// 5xx errors don't include an ErrorResponse body
		return se
	}

	er := new(ErrorResponse)
	if err := json.Unmarshal(resp.Body(), er); err != nil {
		se.Err = fmt.Errorf("invalid JSON response (%w): %s", err, resp.Body())
		return se
	}

	se.Err = er
	return se
}


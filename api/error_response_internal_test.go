package api

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewErrorResponse(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		header     http.Header
		body       string
		wantMsg    string
		retryAfter int32
	}{
		{
			name:    "exception details",
			status:  http.StatusBadRequest,
			body:    `{"exception":{"exceptionDetailList":[{"exceptionCode":21405,"exceptionDescription":"Błąd walidacji danych wejściowych."}]}}`,
			wantMsg: "KSeF service error response (Status 400 Bad Request): Code 21405: Błąd walidacji danych wejściowych.",
		},
		{
			name:       "rate limited",
			status:     http.StatusTooManyRequests,
			header:     http.Header{"Retry-After": []string{"30"}},
			body:       `{"status":{"code":429,"description":"Too Many Requests","details":["Przekroczono limit 20 żądań na minutę. Spróbuj ponownie po 30 sekundach."]}}`,
			wantMsg:    "KSeF service error response (Status 429 Too Many Requests): Przekroczono limit 20 żądań na minutę. Spróbuj ponownie po 30 sekundach.",
			retryAfter: 30,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &resty.Response{RawResponse: &http.Response{
				StatusCode: tt.status,
				Status:     fmt.Sprintf("%d %s", tt.status, http.StatusText(tt.status)),
				Header:     tt.header,
			}}
			resp.SetBody([]byte(tt.body))

			err := newErrorResponse(resp)

			var se *ServiceError
			require.True(t, errors.As(err, &se))
			assert.Equal(t, tt.wantMsg, err.Error())
			assert.Equal(t, tt.retryAfter, se.RetryAfter)
		})
	}
}

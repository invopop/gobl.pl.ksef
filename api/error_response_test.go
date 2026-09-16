package api_test

import (
	"context"
	"errors"
	"net/url"
	"testing"

	ksef_api "github.com/invopop/gobl.pl.ksef/api"
	"github.com/stretchr/testify/assert"
)

func TestIsTransient(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"service error 500", &ksef_api.ServiceError{StatusCode: 500, Status: "500 Internal Server Error"}, true},
		{"service error 503", &ksef_api.ServiceError{StatusCode: 503, Status: "503 Service Unavailable"}, true},
		{"service error 429", &ksef_api.ServiceError{StatusCode: 429, Status: "429 Too Many Requests"}, true},
		{"service error 400", &ksef_api.ServiceError{StatusCode: 400, Status: "400 Bad Request"}, false},
		{"service error 404", &ksef_api.ServiceError{StatusCode: 404, Status: "404 Not Found"}, false},
		{"context deadline exceeded", context.DeadlineExceeded, true},
		{"context canceled", context.Canceled, true},
		{"url error wrapping deadline", &url.Error{Op: "Post", URL: "https://api.ksef.mf.gov.pl/v2/auth/token/refresh", Err: context.DeadlineExceeded}, true},
		{"generic error", errors.New("boom"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ksef_api.IsTransient(tt.err))
		})
	}
}

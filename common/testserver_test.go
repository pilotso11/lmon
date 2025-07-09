package common

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestStartTestServer(t *testing.T) {
	defer goleak.VerifyNone(t)

	// We need to cancel before the t.Context() is done to test for leaks.
	ctx, cancel := context.WithCancel(t.Context())

	type args struct {
		t       *testing.T
		uri     string
		code    int
		delay   int
		timeout int
	}
	tests := []struct {
		name        string
		args        args
		want        int
		wantErr     bool
		wantTimeout bool
	}{
		{"default", args{t: t, uri: "/health", code: http.StatusOK}, http.StatusOK, false, false},
		{"with delay", args{t: t, uri: "/", code: http.StatusOK, delay: 10, timeout: 20}, http.StatusOK, false, false},
		{"with timeout", args{t: t, uri: "/", code: http.StatusOK, delay: 20, timeout: 10}, http.StatusOK, false, true},
		{"with 404", args{t: t, uri: "/", code: http.StatusNotFound, delay: 0, timeout: 10}, http.StatusNotFound, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := StartTestServerWithContext(ctx, tt.args.t, tt.args.uri)
			s.RespCode.Store(int32(tt.args.code))
			s.DelayMs.Store(int32(tt.args.delay))
			to := time.Duration(tt.args.timeout) * time.Millisecond
			if to == 0 {
				to = 100 * time.Millisecond // Default timeout if not specified
			}

			if !tt.wantTimeout {
				assert.Eventually(t, func() bool {
					r, err := http.Get(s.URL)
					defer func(Body io.ReadCloser) {
						_ = Body.Close()
					}(r.Body)
					if tt.wantErr {
						assert.Error(t, err, "error expected")
					} else {
						assert.NoError(t, err, "error not expected")
						assert.Equal(t, tt.args.code, r.StatusCode, "status code")
					}
					return true
				}, to, 1*time.Millisecond, "Server did not respond within %v", to)
			} else {
				toCtx, cancel := context.WithTimeout(t.Context(), to)
				defer cancel()
				body := strings.NewReader("test body")
				req, _ := http.NewRequestWithContext(toCtx, "GET", s.URL, body)
				r, err := http.DefaultClient.Do(req)
				assert.ErrorIs(t, err, context.DeadlineExceeded, "expected timeout error")
				assert.Equal(t, "test body", s.ReqBody.Load(), "status code")
				if r != nil {
					defer func(Body io.ReadCloser) {
						_ = Body.Close()
					}(r.Body)
				} else {
					assert.Nil(t, r, "response should be nil on timeout")
				}
			}

		})
	}
	cancel()
}

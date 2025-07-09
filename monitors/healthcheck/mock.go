// mock.go provides a mock implementation of UsageProvider for testing healthcheck monitors.
package healthcheck

import (
	"context"
	"net/http"
	"net/url"

	"go.uber.org/atomic"
)

var _ UsageProvider = (*MockHealthcheckProvider)(nil)

// MockHealthcheckProvider is a mock implementation of UsageProvider for testing.
// It allows simulation of HTTP status codes and errors.
type MockHealthcheckProvider struct {
	Result *atomic.Int32 // HTTP status RespCode to return
	err    error         // Error to return from Check, if any
}

// Check returns a mocked http.Response based on the Result value, or an error if set.
func (m MockHealthcheckProvider) Check(_ context.Context, _ *url.URL, _ int) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &http.Response{
		StatusCode: int(m.Result.Load()),
		Status:     http.StatusText(int(m.Result.Load())),
	}, nil
}

// NewMockHealthcheckProvider creates a new MockHealthcheckProvider with the given HTTP status RespCode.
func NewMockHealthcheckProvider(code int) *MockHealthcheckProvider {
	return &MockHealthcheckProvider{Result: atomic.NewInt32(int32(code))}
}

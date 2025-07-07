package healthcheck

import (
	"context"
	"net/http"
	"net/url"

	"go.uber.org/atomic"
)

type MockHealthcheckProvider struct {
	Result *atomic.Int32
	err    error
}

func (m MockHealthcheckProvider) Check(_ context.Context, _ *url.URL, _ int) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &http.Response{
		StatusCode: int(m.Result.Load()),
		Status:     http.StatusText(int(m.Result.Load())),
	}, nil
}

func NewMockHealthcheckProvider(code int) *MockHealthcheckProvider {
	return &MockHealthcheckProvider{Result: atomic.NewInt32(int32(code))}
}

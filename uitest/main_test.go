package uitest

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"lmon/web"
)

// TestServerHealth tests that the server is healthy
func TestServerHealth(t *testing.T) {
	ctx, canncel := context.WithCancel(t.Context())
	defer canncel()

	s, _ := web.StartTestServer(ctx, t, "")
	s.Start(ctx)

	resp, body := web.GetTestRequest(ctx, t, s, "/healthz")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, body, "body returned")
	_ = resp.Body.Close()
}

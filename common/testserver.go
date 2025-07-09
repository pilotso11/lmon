package common

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
)

// TestServer is a helper for spinning up a local HTTP server for healthcheck integration tests.
type TestServer struct {
	Server   *http.Server
	RespCode atomic.Int32
	DelayMs  atomic.Int32
	URL      string
	ReqBody  atomic.String
	BodyType atomic.String // Content-Type of the request body, if applicable
}

// handler is the HTTP handler for the TestServer, simulating various status codes and delays.
func (ts *TestServer) handler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	ts.ReqBody.Store(string(body))
	ts.BodyType.Store(r.Header.Get("Content-Type"))

	delay := ts.DelayMs.Load()
	code := int(ts.RespCode.Load())
	if delay > 0 {
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
	if code == http.StatusOK {
		w.WriteHeader(code)
		_, _ = w.Write([]byte(http.StatusText(code)))
	} else {
		http.Error(w, http.StatusText(code), code)
	}

}

// StartTestServer spins up a test HTTP server for integration tests.
func StartTestServer(t *testing.T, uri string) *TestServer {
	return StartTestServerWithContext(t.Context(), t, uri)
}

// StartTestServerWithContext spins up a test HTTP server for  integration tests with a cancel context.
func StartTestServerWithContext(ctx context.Context, t *testing.T, uri string) *TestServer {
	ts := &TestServer{}
	ts.Server = &http.Server{}
	ts.RespCode.Store(int32(http.StatusOK))
	mux := http.NewServeMux()
	mux.HandleFunc(uri, ts.handler)
	ts.Server.Handler = mux

	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	assert.NoError(t, err)
	ts.URL = "http://" + ln.Addr().String() + uri

	// Run the server in a goroutine
	go func() {
		_ = ts.Server.Serve(ln)
	}()

	// Cleanup
	go func() {
		<-ctx.Done()
		_ = ts.Server.Close()
	}()

	return ts
}

package ping

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockPingProvider_Success(t *testing.T) {
	mock := NewMockPingProvider(42, nil)
	ms, err := mock.Ping(context.Background(), "127.0.0.1", 1000)
	assert.Nil(t, err, "Expected no error")
	assert.Equal(t, 42, ms, "response time")
}

func TestMockPingProvider_Error(t *testing.T) {
	mock := NewMockPingProvider(0, errors.New("simulated error"))
	ms, err := mock.Ping(context.Background(), "127.0.0.1", 1000)
	require.Error(t, err, "error is expected")
	assert.Equal(t, "simulated error", err.Error(), "error message")
	assert.Equal(t, 0, ms, "response time")
}

func TestMockPingProvider_ZeroResponse(t *testing.T) {
	mock := NewMockPingProvider(0, nil)
	ms, err := mock.Ping(context.Background(), "127.0.0.1", 1000)
	assert.Nil(t, err, "Expected no error")
	assert.Equal(t, 0, ms, "response time")
}

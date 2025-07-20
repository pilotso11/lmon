package ping

import (
	"context"
	"errors"
	"testing"
)

func TestMockPingProvider_Success(t *testing.T) {
	mock := NewMockPingProvider(42, nil)
	ms, err := mock.Ping(context.Background(), "127.0.0.1", 1000)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if ms != 42 {
		t.Errorf("Expected response time 42, got %d", ms)
	}
}

func TestMockPingProvider_Error(t *testing.T) {
	mock := NewMockPingProvider(0, errors.New("simulated error"))
	ms, err := mock.Ping(context.Background(), "127.0.0.1", 1000)
	if err == nil {
		t.Errorf("Expected error, got nil")
	} else if err.Error() != "simulated error" {
		t.Errorf("Unexpected error message: %v", err)
	}
	if ms != 0 {
		t.Errorf("Expected response time 0, got %d", ms)
	}
}

func TestMockPingProvider_ZeroResponse(t *testing.T) {
	mock := NewMockPingProvider(0, nil)
	ms, err := mock.Ping(context.Background(), "127.0.0.1", 1000)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if ms != 0 {
		t.Errorf("Expected response time 0, got %d", ms)
	}
}

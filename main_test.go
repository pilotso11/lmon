package main

import (
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"lmon/common"
)

func Test_Main(t *testing.T) {
	t.Setenv("LMON_WEB_PORT", "0")
	logWriter = &common.AtomicWriter{}

	go main()

	time.Sleep(200 * time.Millisecond) // Allow some time for the main function to start

	_ = syscall.Kill(syscall.Getpid(), syscall.SIGINT)

	time.Sleep(200 * time.Millisecond)

	logOutput := logWriter.String()
	assert.Contains(t, logOutput, "Starting lmon - Lightweight Monitoring Service")
	assert.Contains(t, logOutput, "Shutdown complete")
}

func Test_Main_WithCustomPort(t *testing.T) {
	t.Setenv("LMON_WEB_PORT", "0") // Use port 0 for testing
	t.Setenv("LMON_WEB_HOST", "127.0.0.1")
	logWriter = &common.AtomicWriter{}

	go main()

	time.Sleep(200 * time.Millisecond) // Allow some time for the main function to start

	_ = syscall.Kill(syscall.Getpid(), syscall.SIGTERM) // Test SIGTERM instead of SIGINT

	time.Sleep(200 * time.Millisecond)

	logOutput := logWriter.String()
	assert.Contains(t, logOutput, "Starting lmon - Lightweight Monitoring Service")
	assert.Contains(t, logOutput, "Shutdown complete")
}

func Test_Main_WithLogWriter(t *testing.T) {
	t.Setenv("LMON_WEB_PORT", "0")

	// Test the logWriter code path
	writer := &common.AtomicWriter{}
	logWriter = writer

	go main()

	time.Sleep(200 * time.Millisecond)

	_ = syscall.Kill(syscall.Getpid(), syscall.SIGINT)

	time.Sleep(200 * time.Millisecond)

	logOutput := writer.String()
	assert.Contains(t, logOutput, "Starting lmon - Lightweight Monitoring Service")
	assert.Contains(t, logOutput, "Shutdown complete")
}

func Test_Main_WithNilLogWriter(t *testing.T) {
	t.Setenv("LMON_WEB_PORT", "0")

	// Test the nil logWriter code path - should use os.Stdout
	logWriter = nil

	go main()

	time.Sleep(200 * time.Millisecond)

	_ = syscall.Kill(syscall.Getpid(), syscall.SIGINT)

	time.Sleep(200 * time.Millisecond)

	// Can't easily test stdout output, just ensure no panic occurred
}

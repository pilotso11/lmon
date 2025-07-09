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

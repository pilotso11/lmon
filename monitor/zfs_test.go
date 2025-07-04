package monitor

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIsZFSVolume tests the isZFSVolume function
func TestIsZFSVolume(t *testing.T) {
	// Skip test if ZFS is not installed
	_, err := exec.LookPath("zfs")
	if err != nil {
		t.Skip("Skipping test because ZFS is not installed")
	}
	// Save the original exec.Command function
	originalExecCommand := execCommand
	defer func() { execCommand = originalExecCommand }()

	tests := []struct {
		name     string
		path     string
		mockFunc func(string, ...string) *exec.Cmd
		expected bool
	}{
		{
			name: "Path is a ZFS volume",
			path: "/zfs/pool",
			mockFunc: func(command string, args ...string) *exec.Cmd {
				assert.Equal(t, "zfs", command)
				assert.Equal(t, []string{"list", "-H", "-o", "name", "/zfs/pool"}, args)
				return mockExecCommand("echo", "zpool/dataset")
			},
			expected: true,
		},
		{
			name: "Path is not a ZFS volume",
			path: "/not/zfs",
			mockFunc: func(command string, args ...string) *exec.Cmd {
				assert.Equal(t, "zfs", command)
				assert.Equal(t, []string{"list", "-H", "-o", "name", "/not/zfs"}, args)
				return mockExecCommand("echo", "")
			},
			expected: false,
		},
		{
			name: "Command fails",
			path: "/error/path",
			mockFunc: func(command string, args ...string) *exec.Cmd {
				assert.Equal(t, "zfs", command)
				assert.Equal(t, []string{"list", "-H", "-o", "name", "/error/path"}, args)
				return mockExecCommandError("exit 1")
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Mock the exec.Command function
			execCommand = tc.mockFunc

			// Call the function
			result := isZFSVolume(tc.path)

			// Verify the result
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestGetZFSPoolHealth tests the getZFSPoolHealth function
func TestGetZFSPoolHealth(t *testing.T) {
	// Skip test if ZFS is not installed
	_, err := exec.LookPath("zfs")
	if err != nil {
		t.Skip("Skipping test because ZFS is not installed")
	}
	// Save the original exec.Command function
	originalExecCommand := execCommand
	defer func() { execCommand = originalExecCommand }()

	tests := []struct {
		name           string
		path           string
		mockFuncs      []func(string, ...string) *exec.Cmd
		expectedPool   string
		expectedStatus string
		expectError    bool
	}{
		{
			name: "Valid ZFS pool with ONLINE status",
			path: "/zfs/pool",
			mockFuncs: []func(string, ...string) *exec.Cmd{
				// First call to get dataset name
				func(command string, args ...string) *exec.Cmd {
					assert.Equal(t, "zfs", command)
					assert.Equal(t, []string{"list", "-H", "-o", "name", "/zfs/pool"}, args)
					return mockExecCommand("echo", "zpool/dataset")
				},
				// Second call to get pool status
				func(command string, args ...string) *exec.Cmd {
					assert.Equal(t, "zpool", command)
					assert.Equal(t, []string{"status", "-H", "-p", "zpool"}, args)
					return mockExecCommand("echo", "zpool\t1234567890\tONLINE\t0\t0\t0")
				},
			},
			expectedPool:   "zpool",
			expectedStatus: "ONLINE",
			expectError:    false,
		},
		{
			name: "Valid ZFS pool with DEGRADED status",
			path: "/zfs/pool",
			mockFuncs: []func(string, ...string) *exec.Cmd{
				// First call to get dataset name
				func(command string, args ...string) *exec.Cmd {
					assert.Equal(t, "zfs", command)
					assert.Equal(t, []string{"list", "-H", "-o", "name", "/zfs/pool"}, args)
					return mockExecCommand("echo", "zpool/dataset")
				},
				// Second call to get pool status
				func(command string, args ...string) *exec.Cmd {
					assert.Equal(t, "zpool", command)
					assert.Equal(t, []string{"status", "-H", "-p", "zpool"}, args)
					return mockExecCommand("echo", "zpool\t1234567890\tDEGRADED\t0\t0\t0")
				},
			},
			expectedPool:   "zpool",
			expectedStatus: "DEGRADED",
			expectError:    false,
		},
		{
			name: "Error getting dataset name",
			path: "/error/path",
			mockFuncs: []func(string, ...string) *exec.Cmd{
				// First call to get dataset name fails
				func(command string, args ...string) *exec.Cmd {
					assert.Equal(t, "zfs", command)
					assert.Equal(t, []string{"list", "-H", "-o", "name", "/error/path"}, args)
					return mockExecCommandError("exit 1")
				},
			},
			expectedPool:   "",
			expectedStatus: "",
			expectError:    true,
		},
		{
			name: "Empty dataset name",
			path: "/empty/dataset",
			mockFuncs: []func(string, ...string) *exec.Cmd{
				// First call to get dataset name returns empty
				func(command string, args ...string) *exec.Cmd {
					assert.Equal(t, "zfs", command)
					assert.Equal(t, []string{"list", "-H", "-o", "name", "/empty/dataset"}, args)
					return mockExecCommand("echo", "")
				},
			},
			expectedPool:   "",
			expectedStatus: "",
			expectError:    true,
		},
		{
			name: "Error getting pool status",
			path: "/zfs/pool",
			mockFuncs: []func(string, ...string) *exec.Cmd{
				// First call to get dataset name
				func(command string, args ...string) *exec.Cmd {
					assert.Equal(t, "zfs", command)
					assert.Equal(t, []string{"list", "-H", "-o", "name", "/zfs/pool"}, args)
					return mockExecCommand("echo", "zpool/dataset")
				},
				// Second call to get pool status fails
				func(command string, args ...string) *exec.Cmd {
					assert.Equal(t, "zpool", command)
					assert.Equal(t, []string{"status", "-H", "-p", "zpool"}, args)
					return mockExecCommandError("exit 1")
				},
			},
			expectedPool:   "",
			expectedStatus: "",
			expectError:    true,
		},
		{
			name: "Invalid pool status output format",
			path: "/zfs/pool",
			mockFuncs: []func(string, ...string) *exec.Cmd{
				// First call to get dataset name
				func(command string, args ...string) *exec.Cmd {
					assert.Equal(t, "zfs", command)
					assert.Equal(t, []string{"list", "-H", "-o", "name", "/zfs/pool"}, args)
					return mockExecCommand("echo", "zpool/dataset")
				},
				// Second call to get pool status returns invalid format
				func(command string, args ...string) *exec.Cmd {
					assert.Equal(t, "zpool", command)
					assert.Equal(t, []string{"status", "-H", "-p", "zpool"}, args)
					return mockExecCommand("echo", "invalid format")
				},
			},
			expectedPool:   "",
			expectedStatus: "",
			expectError:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set up a counter for the mock functions
			callCount := 0
			execCommand = func(command string, args ...string) *exec.Cmd {
				if callCount < len(tc.mockFuncs) {
					cmd := tc.mockFuncs[callCount](command, args...)
					callCount++
					return cmd
				}
				t.Fatalf("Unexpected call to execCommand: %s %v", command, args)
				return nil
			}

			// Call the function
			result, err := getZFSPoolHealth(tc.path)

			// Verify the result
			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tc.expectedPool, result.Pool)
				assert.Equal(t, tc.expectedStatus, result.Status)
			}

			// Verify all mock functions were called
			assert.Equal(t, len(tc.mockFuncs), callCount, "Not all mock functions were called")
		})
	}
}

// Mock helpers

// Override the exec.Command function
var execCommand = exec.Command

// mockExecCommand returns a mock exec.Command that executes the echo command with the given output
func mockExecCommand(command string, output string) *exec.Cmd {
	return exec.Command(command, output)
}

// mockExecCommandError returns a mock exec.Command that fails
func mockExecCommandError(command string) *exec.Cmd {
	return exec.Command(command)
}

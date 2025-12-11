package docker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMonitor_Restart_WithWhitelist tests restart with allowed containers
func TestMonitor_Restart_WithWhitelist(t *testing.T) {
	tests := []struct {
		name              string
		containers        string
		allowedContainers []string
		expectRestart     []string
		expectError       bool
	}{
		{
			name:              "all containers allowed",
			containers:        "web-app, api-server, worker",
			allowedContainers: []string{"web-app", "api-server", "worker", "database"},
			expectRestart:     []string{"web-app", "api-server", "worker"},
			expectError:       false,
		},
		{
			name:              "some containers allowed",
			containers:        "web-app, api-server, worker",
			allowedContainers: []string{"web-app", "api-server"},
			expectRestart:     []string{"web-app", "api-server"},
			expectError:       false,
		},
		{
			name:              "no containers allowed",
			containers:        "web-app, api-server",
			allowedContainers: []string{"database", "cache"},
			expectRestart:     nil,
			expectError:       true,
		},
		{
			name:              "empty whitelist allows all",
			containers:        "web-app, api-server, worker",
			allowedContainers: nil,
			expectRestart:     []string{"web-app", "api-server", "worker"},
			expectError:       false,
		},
		{
			name:              "single container allowed",
			containers:        "web-app, api-server",
			allowedContainers: []string{"web-app"},
			expectRestart:     []string{"web-app"},
			expectError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockDockerProvider{}
			m, err := NewMonitor("test", tt.containers, 5, "box", 0, tt.allowedContainers, mock)
			require.NoError(t, err)

			err = m.Restart(context.Background())
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "no containers in the restart list are allowed")
			} else {
				assert.NoError(t, err)
				assert.ElementsMatch(t, tt.expectRestart, mock.RestartsRequested)
			}
		})
	}
}

// TestFilterAllowedContainers tests the filtering logic
func TestFilterAllowedContainers(t *testing.T) {
	tests := []struct {
		name        string
		containers  []string
		allowedList []string
		expected    []string
	}{
		{
			name:        "all allowed",
			containers:  []string{"web", "api", "db"},
			allowedList: []string{"web", "api", "db"},
			expected:    []string{"web", "api", "db"},
		},
		{
			name:        "some allowed",
			containers:  []string{"web", "api", "db"},
			allowedList: []string{"web", "api"},
			expected:    []string{"web", "api"},
		},
		{
			name:        "none allowed",
			containers:  []string{"web", "api", "db"},
			allowedList: []string{"cache", "queue"},
			expected:    []string{},
		},
		{
			name:        "empty whitelist allows all",
			containers:  []string{"web", "api", "db"},
			allowedList: []string{},
			expected:    []string{"web", "api", "db"},
		},
		{
			name:        "empty containers",
			containers:  []string{},
			allowedList: []string{"web", "api"},
			expected:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterAllowedContainers(tt.containers, tt.allowedList)
			if len(tt.expected) == 0 && len(result) == 0 {
				// Both are empty, consider them equal (nil vs empty slice)
				return
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestFindSkippedContainers tests finding containers that were skipped
func TestFindSkippedContainers(t *testing.T) {
	tests := []struct {
		name     string
		original []string
		filtered []string
		expected []string
	}{
		{
			name:     "some skipped",
			original: []string{"web", "api", "db"},
			filtered: []string{"web", "api"},
			expected: []string{"db"},
		},
		{
			name:     "none skipped",
			original: []string{"web", "api", "db"},
			filtered: []string{"web", "api", "db"},
			expected: []string{},
		},
		{
			name:     "all skipped",
			original: []string{"web", "api", "db"},
			filtered: []string{},
			expected: []string{"web", "api", "db"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findSkippedContainers(tt.original, tt.filtered)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

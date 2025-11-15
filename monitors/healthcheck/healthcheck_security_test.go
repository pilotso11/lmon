package healthcheck

import (
	"context"
	"testing"

	"lmon/monitors/docker"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHealthcheck_RestartContainers_WithWhitelist tests restart with allowed containers
func TestHealthcheck_RestartContainers_WithWhitelist(t *testing.T) {
	tests := []struct {
		name             string
		restartContainers string
		allowedContainers string
		expectRestart    []string
		expectError      bool
	}{
		{
			name:             "all containers allowed",
			restartContainers: "web-app, api-server, worker",
			allowedContainers: "web-app, api-server, worker, database",
			expectRestart:    []string{"web-app", "api-server", "worker"},
			expectError:      false,
		},
		{
			name:             "some containers allowed",
			restartContainers: "web-app, api-server, worker",
			allowedContainers: "web-app, api-server",
			expectRestart:    []string{"web-app", "api-server"},
			expectError:      false,
		},
		{
			name:             "no containers allowed",
			restartContainers: "web-app, api-server",
			allowedContainers: "database, cache",
			expectRestart:    nil,
			expectError:      true,
		},
		{
			name:             "empty whitelist allows all",
			restartContainers: "web-app, api-server, worker",
			allowedContainers: "",
			expectRestart:    []string{"web-app", "api-server", "worker"},
			expectError:      false,
		},
		{
			name:             "single container allowed",
			restartContainers: "web-app, api-server",
			allowedContainers: "web-app",
			expectRestart:    []string{"web-app"},
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDocker := &docker.MockDockerProvider{}
			h, err := NewHealthcheck("test", "http://localhost", 5, 0, "", tt.restartContainers, tt.allowedContainers, nil, mockDocker)
			require.NoError(t, err)

			err = h.RestartContainers(context.Background())
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "no containers in the restart list are allowed")
			} else {
				assert.NoError(t, err)
				assert.ElementsMatch(t, tt.expectRestart, mockDocker.RestartsRequested)
			}
		})
	}
}

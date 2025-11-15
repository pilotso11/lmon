package docker

import (
	"context"
	"fmt"
)

// MockDockerProvider is a mock implementation of Provider for testing
type MockDockerProvider struct {
	RestartCounts     map[string]int
	RestartError      error
	GetCountsError    error
	RestartsRequested []string
	CloseCalled       bool
	CloseError        error
}

// GetRestartCounts returns mock restart counts
func (m *MockDockerProvider) GetRestartCounts(ctx context.Context, containerNames []string) (map[string]int, error) {
	if m.GetCountsError != nil {
		return nil, m.GetCountsError
	}

	if m.RestartCounts == nil {
		// Default: return 0 for all requested containers
		result := make(map[string]int)
		for _, name := range containerNames {
			result[name] = 0
		}
		return result, nil
	}

	// Return configured counts for requested containers
	result := make(map[string]int)
	for _, name := range containerNames {
		if count, ok := m.RestartCounts[name]; ok {
			result[name] = count
		} else {
			return nil, fmt.Errorf("container not found: %s", name)
		}
	}

	return result, nil
}

// RestartContainers records restart requests
func (m *MockDockerProvider) RestartContainers(ctx context.Context, containerNames []string) error {
	if m.RestartError != nil {
		return m.RestartError
	}

	m.RestartsRequested = append(m.RestartsRequested, containerNames...)
	return nil
}

// Close implements io.Closer for testing
func (m *MockDockerProvider) Close() error {
	m.CloseCalled = true
	return m.CloseError
}

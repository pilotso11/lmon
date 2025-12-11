package docker

import (
	"context"
	"errors"
	"testing"

	"lmon/config"
	"lmon/monitors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMonitor(t *testing.T) {
	tests := []struct {
		name       string
		monName    string
		containers string
		threshold  int
		icon       string
		wantErr    bool
		wantCount  int
	}{
		{
			name:       "valid comma separated",
			monName:    "test",
			containers: "web-app, api-server, worker",
			threshold:  5,
			icon:       "box",
			wantErr:    false,
			wantCount:  3,
		},
		{
			name:       "valid space separated",
			monName:    "test",
			containers: "web-app api-server worker",
			threshold:  5,
			icon:       "box",
			wantErr:    false,
			wantCount:  3,
		},
		{
			name:       "valid mixed separators",
			monName:    "test",
			containers: "web-app, api-server worker",
			threshold:  5,
			icon:       "box",
			wantErr:    false,
			wantCount:  3,
		},
		{
			name:       "single container",
			monName:    "test",
			containers: "web-app",
			threshold:  5,
			icon:       "box",
			wantErr:    false,
			wantCount:  1,
		},
		{
			name:       "empty containers",
			monName:    "test",
			containers: "",
			threshold:  5,
			icon:       "box",
			wantErr:    true,
		},
		{
			name:       "default icon",
			monName:    "test",
			containers: "web-app",
			threshold:  5,
			icon:       "",
			wantErr:    false,
			wantCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockDockerProvider{}
			m, err := NewMonitor(tt.monName, tt.containers, tt.threshold, tt.icon, 0, nil, mock)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantCount, len(m.containers))
			assert.Equal(t, tt.threshold, m.threshold)

			if tt.icon == "" {
				assert.Equal(t, Icon, m.icon)
			} else {
				assert.Equal(t, tt.icon, m.icon)
			}
		})
	}
}

func TestMonitor_Name(t *testing.T) {
	mock := &MockDockerProvider{}
	m, err := NewMonitor("test", "web-app", 5, "box", 0, nil, mock)
	require.NoError(t, err)

	assert.Equal(t, "docker_test", m.Name())
}

func TestMonitor_Group(t *testing.T) {
	mock := &MockDockerProvider{}
	m, err := NewMonitor("test", "web-app", 5, "box", 0, nil, mock)
	require.NoError(t, err)

	assert.Equal(t, Group, m.Group())
}

func TestMonitor_DisplayName(t *testing.T) {
	tests := []struct {
		name       string
		containers string
		want       string
	}{
		{
			name:       "single container",
			containers: "web-app",
			want:       "web-app",
		},
		{
			name:       "multiple containers",
			containers: "web-app, api-server, worker",
			want:       "test (3 containers)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockDockerProvider{}
			m, err := NewMonitor("test", tt.containers, 5, "box", 0, nil, mock)
			require.NoError(t, err)

			assert.Equal(t, tt.want, m.DisplayName())
		})
	}
}

func TestMonitor_Check_Green(t *testing.T) {
	mock := &MockDockerProvider{
		RestartCounts: map[string]int{
			"web-app":    0,
			"api-server": 1,
			"worker":     2,
		},
	}

	m, err := NewMonitor("test", "web-app, api-server, worker", 10, "box", 0, nil, mock)
	require.NoError(t, err)

	result := m.Check(context.Background())

	assert.Equal(t, monitors.RAGGreen, result.Status)
	assert.Contains(t, result.Value, "Max restarts: 2")
	assert.Contains(t, result.Value2, "web-app: 0")
	assert.Contains(t, result.Value2, "api-server: 1")
	assert.Contains(t, result.Value2, "worker: 2")
}

func TestMonitor_Check_Amber(t *testing.T) {
	mock := &MockDockerProvider{
		RestartCounts: map[string]int{
			"web-app":    9,
			"api-server": 8,
			"worker":     7,
		},
	}

	m, err := NewMonitor("test", "web-app, api-server, worker", 10, "box", 0, nil, mock)
	require.NoError(t, err)

	result := m.Check(context.Background())

	assert.Equal(t, monitors.RAGAmber, result.Status)
	assert.Contains(t, result.Value, "Max restarts: 9")
}

func TestMonitor_Check_Red(t *testing.T) {
	mock := &MockDockerProvider{
		RestartCounts: map[string]int{
			"web-app":    15,
			"api-server": 8,
			"worker":     7,
		},
	}

	m, err := NewMonitor("test", "web-app, api-server, worker", 10, "box", 0, nil, mock)
	require.NoError(t, err)

	result := m.Check(context.Background())

	assert.Equal(t, monitors.RAGRed, result.Status)
	assert.Contains(t, result.Value, "Max restarts: 15")
}

func TestMonitor_Check_Error(t *testing.T) {
	mock := &MockDockerProvider{
		GetCountsError: errors.New("Docker daemon not available"),
	}

	m, err := NewMonitor("test", "web-app", 10, "box", 0, nil, mock)
	require.NoError(t, err)

	result := m.Check(context.Background())

	assert.Equal(t, monitors.RAGError, result.Status)
	assert.Contains(t, result.Value, "error")
	assert.Contains(t, result.Value, "Docker daemon not available")
}

func TestMonitor_Check_ThresholdEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		restartCount   int
		threshold      int
		expectedStatus monitors.RAG
	}{
		{
			name:           "exactly at threshold",
			restartCount:   10,
			threshold:      10,
			expectedStatus: monitors.RAGRed,
		},
		{
			name:           "just below threshold",
			restartCount:   9,
			threshold:      10,
			expectedStatus: monitors.RAGAmber, // 9 >= 10*0.9 (9)
		},
		{
			name:           "at amber boundary",
			restartCount:   9,
			threshold:      10,
			expectedStatus: monitors.RAGAmber,
		},
		{
			name:           "below amber boundary",
			restartCount:   8,
			threshold:      10,
			expectedStatus: monitors.RAGGreen, // 8 < 10*0.9 (9)
		},
		{
			name:           "zero restarts",
			restartCount:   0,
			threshold:      10,
			expectedStatus: monitors.RAGGreen,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockDockerProvider{
				RestartCounts: map[string]int{
					"web-app": tt.restartCount,
				},
			}

			m, err := NewMonitor("test", "web-app", tt.threshold, "box", 0, nil, mock)
			require.NoError(t, err)

			result := m.Check(context.Background())
			assert.Equal(t, tt.expectedStatus, result.Status)
		})
	}
}

func TestMonitor_Save(t *testing.T) {
	mock := &MockDockerProvider{}
	m, err := NewMonitor("test", "web-app, api-server", 5, "box", 0, nil, mock)
	require.NoError(t, err)

	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			Docker: make(map[string]config.DockerConfig),
		},
	}

	m.Save(cfg)

	saved, ok := cfg.Monitoring.Docker["test"]
	require.True(t, ok)
	assert.Equal(t, "web-app, api-server", saved.Containers)
	assert.Equal(t, 5, saved.Threshold)
	assert.Equal(t, "box", saved.Icon)
}

func TestMonitor_Restart(t *testing.T) {
	mock := &MockDockerProvider{}
	m, err := NewMonitor("test", "web-app, api-server, worker", 5, "box", 0, nil, mock)
	require.NoError(t, err)

	err = m.Restart(context.Background())
	require.NoError(t, err)

	assert.ElementsMatch(t, []string{"web-app", "api-server", "worker"}, mock.RestartsRequested)
}

func TestMonitor_Restart_Error(t *testing.T) {
	mock := &MockDockerProvider{
		RestartError: errors.New("permission denied"),
	}
	m, err := NewMonitor("test", "web-app", 5, "box", 0, nil, mock)
	require.NoError(t, err)

	err = m.Restart(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}

func TestParseContainerList(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "comma separated",
			input: "web-app,api-server,worker",
			want:  []string{"web-app", "api-server", "worker"},
		},
		{
			name:  "comma and space separated",
			input: "web-app, api-server, worker",
			want:  []string{"web-app", "api-server", "worker"},
		},
		{
			name:  "space separated",
			input: "web-app api-server worker",
			want:  []string{"web-app", "api-server", "worker"},
		},
		{
			name:  "mixed separators",
			input: "web-app, api-server worker",
			want:  []string{"web-app", "api-server", "worker"},
		},
		{
			name:  "extra whitespace",
			input: "  web-app  ,  api-server  ,  worker  ",
			want:  []string{"web-app", "api-server", "worker"},
		},
		{
			name:  "single container",
			input: "web-app",
			want:  []string{"web-app"},
		},
		{
			name:  "empty string",
			input: "",
			want:  []string{},
		},
		{
			name:  "only whitespace",
			input: "   ",
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseContainerList(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMonitor_Close(t *testing.T) {
	mock := &MockDockerProvider{}
	m, err := NewMonitor("test", "web-app", 5, "box", 0, nil, mock)
	require.NoError(t, err)

	err = m.Close()
	require.NoError(t, err)
	assert.True(t, mock.CloseCalled, "Close should be called on the provider")
}

func TestMonitor_Close_Error(t *testing.T) {
	mock := &MockDockerProvider{
		CloseError: errors.New("close failed"),
	}
	m, err := NewMonitor("test", "web-app", 5, "box", 0, nil, mock)
	require.NoError(t, err)

	err = m.Close()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "close failed")
	assert.True(t, mock.CloseCalled, "Close should be called even if it returns an error")
}

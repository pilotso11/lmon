package monitor

import (
	"errors"
	"testing"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"lmon/config"
)

// MockDiskUsageProvider is a mock implementation of DiskUsageProvider
type MockDiskUsageProvider struct {
	mock.Mock
}

// Usage is a mock implementation of the Usage method
func (m *MockDiskUsageProvider) Usage(path string) (*disk.UsageStat, error) {
	args := m.Called(path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*disk.UsageStat), args.Error(1)
}

func TestDiskMonitor_Check(t *testing.T) {
	// Test cases
	tests := []struct {
		name           string
		diskConfig     []config.DiskConfig
		setupMock      func(*MockDiskUsageProvider)
		expectedItems  int
		expectedStatus Status
		expectError    bool
	}{
		{
			name: "Single disk with OK status",
			diskConfig: []config.DiskConfig{
				{
					Path:      "/",
					Threshold: 80,
					Icon:      "storage",
				},
			},
			setupMock: func(m *MockDiskUsageProvider) {
				m.On("Usage", "/").Return(&disk.UsageStat{
					Path:        "/",
					Total:       1000000000,
					Free:        800000000,
					Used:        200000000,
					UsedPercent: 20.0,
				}, nil)
			},
			expectedItems:  1,
			expectedStatus: StatusOK,
			expectError:    false,
		},
		{
			name: "Single disk with WARNING status",
			diskConfig: []config.DiskConfig{
				{
					Path:      "/",
					Threshold: 80,
					Icon:      "storage",
				},
			},
			setupMock: func(m *MockDiskUsageProvider) {
				m.On("Usage", "/").Return(&disk.UsageStat{
					Path:        "/",
					Total:       1000000000,
					Free:        300000000,
					Used:        700000000,
					UsedPercent: 70.0,
				}, nil)
			},
			expectedItems:  1,
			expectedStatus: StatusWarning,
			expectError:    false,
		},
		{
			name: "Single disk with CRITICAL status",
			diskConfig: []config.DiskConfig{
				{
					Path:      "/",
					Threshold: 80,
					Icon:      "storage",
				},
			},
			setupMock: func(m *MockDiskUsageProvider) {
				m.On("Usage", "/").Return(&disk.UsageStat{
					Path:        "/",
					Total:       1000000000,
					Free:        100000000,
					Used:        900000000,
					UsedPercent: 90.0,
				}, nil)
			},
			expectedItems:  1,
			expectedStatus: StatusCritical,
			expectError:    false,
		},
		{
			name: "Multiple disks with different statuses",
			diskConfig: []config.DiskConfig{
				{
					Path:      "/",
					Threshold: 80,
					Icon:      "storage",
				},
				{
					Path:      "/home",
					Threshold: 70,
					Icon:      "storage",
				},
			},
			setupMock: func(m *MockDiskUsageProvider) {
				m.On("Usage", "/").Return(&disk.UsageStat{
					Path:        "/",
					Total:       1000000000,
					Free:        800000000,
					Used:        200000000,
					UsedPercent: 20.0,
				}, nil)
				m.On("Usage", "/home").Return(&disk.UsageStat{
					Path:        "/home",
					Total:       1000000000,
					Free:        200000000,
					Used:        800000000,
					UsedPercent: 80.0,
				}, nil)
			},
			expectedItems: 2,
			expectError:   false,
		},
		{
			name: "Error getting disk usage",
			diskConfig: []config.DiskConfig{
				{
					Path:      "/",
					Threshold: 80,
					Icon:      "storage",
				},
			},
			setupMock: func(m *MockDiskUsageProvider) {
				m.On("Usage", "/").Return(nil, errors.New("failed to get disk usage"))
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock provider
			mockProvider := new(MockDiskUsageProvider)

			// Set up mock expectations
			tc.setupMock(mockProvider)

			// Create test configuration
			cfg := config.DefaultConfig()
			cfg.Monitoring.Disk = tc.diskConfig

			// Create disk monitor with mock provider
			monitor := NewDiskMonitorWithProvider(cfg, mockProvider)

			// Check disk usage
			items, err := monitor.Check()

			// Verify results
			if tc.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectedItems, len(items))

			// Check status of first item if expected
			if tc.expectedStatus != "" && len(items) > 0 {
				assert.Equal(t, tc.expectedStatus, items[0].Status)
			}

			// Check specific items
			for _, item := range items {
				assert.Contains(t, item.ID, "disk-")
				assert.Contains(t, item.Name, "Disk")
				assert.Equal(t, "disk", item.Type)
				assert.NotEmpty(t, item.Message)
			}

			// Verify all expectations were met
			mockProvider.AssertExpectations(t)
		})
	}
}

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
		isZFSVolume    bool
		zfsPoolHealth  *ZFSPoolHealth
		zfsPoolError   error
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
			isZFSVolume:    false,
			zfsPoolHealth:  nil,
			zfsPoolError:   nil,
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
			isZFSVolume:    false,
			zfsPoolHealth:  nil,
			zfsPoolError:   nil,
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
			isZFSVolume:    false,
			zfsPoolHealth:  nil,
			zfsPoolError:   nil,
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
			isZFSVolume:   false,
			zfsPoolHealth: nil,
			zfsPoolError:  nil,
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
			isZFSVolume:    false,
			zfsPoolHealth:  nil,
			zfsPoolError:   nil,
			expectedItems:  1,
			expectedStatus: StatusCritical,
			expectError:    false,
		},
		{
			name: "ZFS volume with ONLINE status",
			diskConfig: []config.DiskConfig{
				{
					Path:      "/zfs/pool",
					Threshold: 80,
					Icon:      "storage",
				},
			},
			setupMock: func(m *MockDiskUsageProvider) {
				m.On("Usage", "/zfs/pool").Return(&disk.UsageStat{
					Path:        "/zfs/pool",
					Total:       1000000000,
					Free:        800000000,
					Used:        200000000,
					UsedPercent: 20.0,
				}, nil)
			},
			isZFSVolume: true,
			zfsPoolHealth: &ZFSPoolHealth{
				Pool:   "zpool",
				Status: "ONLINE",
			},
			zfsPoolError:   nil,
			expectedItems:  1,
			expectedStatus: StatusOK,
			expectError:    false,
		},
		{
			name: "ZFS volume with DEGRADED status",
			diskConfig: []config.DiskConfig{
				{
					Path:      "/zfs/pool",
					Threshold: 80,
					Icon:      "storage",
				},
			},
			setupMock: func(m *MockDiskUsageProvider) {
				m.On("Usage", "/zfs/pool").Return(&disk.UsageStat{
					Path:        "/zfs/pool",
					Total:       1000000000,
					Free:        800000000,
					Used:        200000000,
					UsedPercent: 20.0,
				}, nil)
			},
			isZFSVolume: true,
			zfsPoolHealth: &ZFSPoolHealth{
				Pool:   "zpool",
				Status: "DEGRADED",
			},
			zfsPoolError:   nil,
			expectedItems:  1,
			expectedStatus: StatusCritical,
			expectError:    false,
		},
		{
			name: "ZFS volume with error getting pool health",
			diskConfig: []config.DiskConfig{
				{
					Path:      "/zfs/pool",
					Threshold: 80,
					Icon:      "storage",
				},
			},
			setupMock: func(m *MockDiskUsageProvider) {
				m.On("Usage", "/zfs/pool").Return(&disk.UsageStat{
					Path:        "/zfs/pool",
					Total:       1000000000,
					Free:        800000000,
					Used:        200000000,
					UsedPercent: 20.0,
				}, nil)
			},
			isZFSVolume:    true,
			zfsPoolHealth:  nil,
			zfsPoolError:   errors.New("failed to get ZFS pool health"),
			expectedItems:  1,
			expectedStatus: StatusOK,
			expectError:    false,
		},
		{
			name: "Multiple disks with one failing",
			diskConfig: []config.DiskConfig{
				{
					Path:      "/",
					Threshold: 80,
					Icon:      "storage",
				},
				{
					Path:      "/missing",
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
				m.On("Usage", "/missing").Return(nil, errors.New("disk not found"))
				m.On("Usage", "/home").Return(&disk.UsageStat{
					Path:        "/home",
					Total:       1000000000,
					Free:        200000000,
					Used:        800000000,
					UsedPercent: 80.0,
				}, nil)
			},
			isZFSVolume:   false,
			zfsPoolHealth: nil,
			zfsPoolError:  nil,
			expectedItems: 3,
			expectError:   false,
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

			// Create custom ZFS functions
			isZFSVolume := func(path string) bool {
				return tc.isZFSVolume
			}

			getZFSPoolHealth := func(path string) (*ZFSPoolHealth, error) {
				return tc.zfsPoolHealth, tc.zfsPoolError
			}

			// Create disk monitor with mock provider and custom ZFS functions
			monitor := NewDiskMonitorWithCustomFuncs(cfg, mockProvider, isZFSVolume, getZFSPoolHealth)

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
				// Name format changed from "Disk (name)" to "Name (path)"
				assert.Contains(t, item.Name, "(")
				assert.Equal(t, "disk", item.Type)
				assert.NotEmpty(t, item.Message)
			}

			// Check for specific test cases
			if tc.name == "Multiple disks with one failing" {
				// Find the missing disk item
				var missingDiskItem *Item
				for _, item := range items {
					if item.ID == "disk-/missing" {
						missingDiskItem = item
						break
					}
				}

				// Verify the missing disk is reported correctly
				require.NotNil(t, missingDiskItem)
				assert.Equal(t, StatusCritical, missingDiskItem.Status)
				assert.Contains(t, missingDiskItem.Message, "not accessible")
			} else if tc.name == "ZFS volume with ONLINE status" {
				// Verify ZFS pool health is included in the message
				assert.Contains(t, items[0].Message, "ZFS Pool 'zpool' health: ONLINE")
			} else if tc.name == "ZFS volume with DEGRADED status" {
				// Verify ZFS pool health is included in the message and status is CRITICAL
				assert.Contains(t, items[0].Message, "ZFS Pool 'zpool' health: DEGRADED")
				assert.Equal(t, StatusCritical, items[0].Status)
			}

			// Verify all expectations were met
			mockProvider.AssertExpectations(t)
		})
	}
}

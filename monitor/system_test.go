package monitor

import (
	"errors"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"lmon/config"
)

// MockCPUUsageProvider is a mock implementation of CPUUsageProvider
type MockCPUUsageProvider struct {
	mock.Mock
}

// Percent is a mock implementation of the Percent method
func (m *MockCPUUsageProvider) Percent(interval time.Duration, percpu bool) ([]float64, error) {
	args := m.Called(interval, percpu)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]float64), args.Error(1)
}

// Times is a mock implementation of the Times method
func (m *MockCPUUsageProvider) Times(percpu bool) ([]cpu.TimesStat, error) {
	args := m.Called(percpu)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]cpu.TimesStat), args.Error(1)
}

// MockMemoryUsageProvider is a mock implementation of MemoryUsageProvider
type MockMemoryUsageProvider struct {
	mock.Mock
}

// VirtualMemory is a mock implementation of the VirtualMemory method
func (m *MockMemoryUsageProvider) VirtualMemory() (*mem.VirtualMemoryStat, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mem.VirtualMemoryStat), args.Error(1)
}

func TestSystemMonitor_Check(t *testing.T) {
	// Test cases
	tests := []struct {
		name           string
		systemConfig   config.SystemConfig
		setupCPUMock   func(*MockCPUUsageProvider)
		setupMemMock   func(*MockMemoryUsageProvider)
		expectedItems  int
		expectedStatus map[string]Status
		expectError    bool
	}{
		{
			name: "CPU and Memory OK",
			systemConfig: config.SystemConfig{
				CPUThreshold:    80,
				MemoryThreshold: 80,
				CPUIcon:         "memory",
				MemoryIcon:      "memory",
			},
			setupCPUMock: func(m *MockCPUUsageProvider) {
				m.On("Percent", mock.Anything, false).Return([]float64{30.0}, nil)
				m.On("Times", false).Return([]cpu.TimesStat{{
					User:   100,
					System: 50,
					Idle:   850,
				}}, nil)
			},
			setupMemMock: func(m *MockMemoryUsageProvider) {
				m.On("VirtualMemory").Return(&mem.VirtualMemoryStat{
					Total:       16000000000, // 16GB
					Used:        4000000000,  // 4GB
					Free:        12000000000, // 12GB
					UsedPercent: 25.0,
				}, nil)
			},
			expectedItems: 2,
			expectedStatus: map[string]Status{
				"cpu":    StatusOK,
				"memory": StatusOK,
			},
			expectError: false,
		},
		{
			name: "CPU Warning, Memory OK",
			systemConfig: config.SystemConfig{
				CPUThreshold:    80,
				MemoryThreshold: 80,
				CPUIcon:         "memory",
				MemoryIcon:      "memory",
			},
			setupCPUMock: func(m *MockCPUUsageProvider) {
				m.On("Percent", mock.Anything, false).Return([]float64{70.0}, nil)
				m.On("Times", false).Return([]cpu.TimesStat{{
					User:   350,
					System: 350,
					Idle:   300,
				}}, nil)
			},
			setupMemMock: func(m *MockMemoryUsageProvider) {
				m.On("VirtualMemory").Return(&mem.VirtualMemoryStat{
					Total:       16000000000, // 16GB
					Used:        4000000000,  // 4GB
					Free:        12000000000, // 12GB
					UsedPercent: 25.0,
				}, nil)
			},
			expectedItems: 2,
			expectedStatus: map[string]Status{
				"cpu":    StatusWarning,
				"memory": StatusOK,
			},
			expectError: false,
		},
		{
			name: "CPU Critical, Memory Warning",
			systemConfig: config.SystemConfig{
				CPUThreshold:    80,
				MemoryThreshold: 80,
				CPUIcon:         "memory",
				MemoryIcon:      "memory",
			},
			setupCPUMock: func(m *MockCPUUsageProvider) {
				m.On("Percent", mock.Anything, false).Return([]float64{90.0}, nil)
				m.On("Times", false).Return([]cpu.TimesStat{{
					User:   450,
					System: 450,
					Idle:   100,
				}}, nil)
			},
			setupMemMock: func(m *MockMemoryUsageProvider) {
				m.On("VirtualMemory").Return(&mem.VirtualMemoryStat{
					Total:       16000000000, // 16GB
					Used:        12000000000, // 12GB
					Free:        4000000000,  // 4GB
					UsedPercent: 75.0,
				}, nil)
			},
			expectedItems: 2,
			expectedStatus: map[string]Status{
				"cpu":    StatusCritical,
				"memory": StatusWarning,
			},
			expectError: false,
		},
		{
			name: "CPU Error",
			systemConfig: config.SystemConfig{
				CPUThreshold:    80,
				MemoryThreshold: 80,
				CPUIcon:         "memory",
				MemoryIcon:      "memory",
			},
			setupCPUMock: func(m *MockCPUUsageProvider) {
				m.On("Percent", mock.Anything, false).Return(nil, errors.New("failed to get CPU usage"))
				m.On("Times", false).Return(nil, errors.New("failed to get CPU times"))
			},
			setupMemMock: func(m *MockMemoryUsageProvider) {
				// Memory provider should not be called if CPU check fails
			},
			expectedItems: 0,
			expectError:   true,
		},
		{
			name: "Memory Error",
			systemConfig: config.SystemConfig{
				CPUThreshold:    80,
				MemoryThreshold: 80,
				CPUIcon:         "memory",
				MemoryIcon:      "memory",
			},
			setupCPUMock: func(m *MockCPUUsageProvider) {
				m.On("Percent", mock.Anything, false).Return([]float64{30.0}, nil)
				m.On("Times", false).Return([]cpu.TimesStat{{
					User:   100,
					System: 50,
					Idle:   850,
				}}, nil)
			},
			setupMemMock: func(m *MockMemoryUsageProvider) {
				m.On("VirtualMemory").Return(nil, errors.New("failed to get memory usage"))
			},
			expectedItems: 0,
			expectError:   true,
		},
		{
			name: "Empty CPU data",
			systemConfig: config.SystemConfig{
				CPUThreshold:    80,
				MemoryThreshold: 80,
				CPUIcon:         "memory",
				MemoryIcon:      "memory",
			},
			setupCPUMock: func(m *MockCPUUsageProvider) {
				m.On("Percent", mock.Anything, false).Return([]float64{}, nil)
				m.On("Times", false).Return([]cpu.TimesStat{}, nil)
			},
			setupMemMock: func(m *MockMemoryUsageProvider) {
				// Memory provider should not be called if CPU check fails
			},
			expectedItems: 0,
			expectError:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock providers
			mockCPUProvider := new(MockCPUUsageProvider)
			mockMemProvider := new(MockMemoryUsageProvider)

			// Set up mock expectations
			tc.setupCPUMock(mockCPUProvider)
			tc.setupMemMock(mockMemProvider)

			// Create test configuration
			cfg := config.DefaultConfig()
			cfg.Monitoring.System = tc.systemConfig

			// Create system monitor with mock providers
			monitor := NewSystemMonitorWithProviders(cfg, mockCPUProvider, mockMemProvider)

			// Check system usage
			items, err := monitor.Check()

			// Verify results
			if tc.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectedItems, len(items))

			// Check status of items
			for _, item := range items {
				if expectedStatus, ok := tc.expectedStatus[item.ID]; ok {
					assert.Equal(t, expectedStatus, item.Status, "Item %s has wrong status", item.ID)
				}

				// Check common properties
				assert.NotEmpty(t, item.Name)
				assert.NotEmpty(t, item.Message)
				assert.NotEmpty(t, item.Icon)
				assert.NotZero(t, item.LastCheck)
			}

			// Verify all expectations were met
			mockCPUProvider.AssertExpectations(t)
			mockMemProvider.AssertExpectations(t)
		})
	}
}

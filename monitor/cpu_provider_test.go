package monitor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultCPUUsageProvider_Percent(t *testing.T) {
	// Create a new DefaultCPUUsageProvider
	provider := &DefaultCPUUsageProvider{}

	// Call the Percent method
	percentages, err := provider.Percent(time.Millisecond*100, false)

	// Verify the result
	assert.NoError(t, err)
	assert.NotNil(t, percentages)
	assert.GreaterOrEqual(t, len(percentages), 1)

	// Verify the percentages are within a valid range (0-100)
	for _, p := range percentages {
		assert.GreaterOrEqual(t, p, 0.0)
		assert.LessOrEqual(t, p, 100.0)
	}
}

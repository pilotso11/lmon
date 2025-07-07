package healthcheck

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMockHealthcheckProvider(t *testing.T) {
	m := NewMockHealthcheckProvider(200)
	assert.NotNil(t, m, "should not be nil")
	assert.Equal(t, int32(200), m.Result.Load(), "Initial value should be 200")
}

func TestMockHealthcheckProvider_Check(t *testing.T) {
	m := NewMockHealthcheckProvider(200)
	resp, err := m.Check(nil, nil, 0)
	assert.NoError(t, err, "should not error")
	assert.NotNil(t, resp, "response should not be nil")
	assert.Equal(t, 200, resp.StatusCode, "should return status code 200")

	// Test with an error
	m.err = assert.AnError
	resp, err = m.Check(nil, nil, 0)
	assert.Error(t, err, "should return an error")
	assert.Nil(t, resp, "response should be nil on error")
}

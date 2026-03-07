package web

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"lmon/config"
	"lmon/monitors"
)

// TestMetricsEndpoint hits GET /metrics and verifies the JSON structure has node, timestamp, and monitors array.
func TestMetricsEndpoint(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	// Wait for initial monitor checks to complete
	time.Sleep(50 * time.Millisecond)

	r, body := GetTestRequest(ctx, t, s, "/metrics")
	assert.Equal(t, http.StatusOK, r.StatusCode, "status code")
	assert.Contains(t, r.Header.Get("Content-Type"), "application/json", "content type")

	var payload MetricsPayload
	err := json.Unmarshal([]byte(body), &payload)
	assert.NoError(t, err, "unmarshal metrics payload")

	// Verify structure
	assert.NotEmpty(t, payload.Node, "node should be set to hostname")
	assert.False(t, payload.Timestamp.IsZero(), "timestamp should not be zero")
	assert.NotNil(t, payload.Monitors, "monitors should not be nil")
}

// TestMetricsContainsAllMonitors adds a disk monitor and verifies it appears in the metrics response.
func TestMetricsContainsAllMonitors(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	// Add a disk monitor
	diskCfg := config.DiskConfig{
		Threshold: 80,
		Icon:      "hdd",
		Path:      ".",
	}
	resp, body := PostTestRequest(ctx, t, s, "/api/config/disk/test-disk", diskCfg)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "add disk status code")
	assert.Equal(t, "OK\n", body)

	// Wait for checks to run
	assert.Eventually(t, func() bool {
		r, body := GetTestRequest(ctx, t, s, "/metrics")
		if r.StatusCode != http.StatusOK {
			return false
		}
		var payload MetricsPayload
		err := json.Unmarshal([]byte(body), &payload)
		if err != nil {
			return false
		}

		// Check that system_cpu, system_mem, and disk_test-disk all appear
		foundCPU := false
		foundMem := false
		foundDisk := false
		for _, m := range payload.Monitors {
			switch m.ID {
			case "system_cpu":
				foundCPU = true
			case "system_mem":
				foundMem = true
			case "disk_test-disk":
				foundDisk = true
			}
		}
		return foundCPU && foundMem && foundDisk
	}, time.Second, 10*time.Millisecond, "metrics should contain cpu, mem, and disk monitors")
}

// TestMetricsRAGStringMapping verifies that status strings in the metrics response are
// "Green", "Amber", "Red", "Error", or "Unknown".
func TestMetricsRAGStringMapping(t *testing.T) {
	// Test the RAG String() method directly for all known values
	assert.Equal(t, "Green", monitors.RAGGreen.String())
	assert.Equal(t, "Amber", monitors.RAGAmber.String())
	assert.Equal(t, "Red", monitors.RAGRed.String())
	assert.Equal(t, "Error", monitors.RAGError.String())
	assert.Equal(t, "Unknown", monitors.RAGUnknown.String())

	// Test through the metrics builder to ensure mapping is preserved
	results := map[string]monitors.Result{
		"test_green": {
			Status: monitors.RAGGreen,
			Group:  "test",
			Value:  "50%",
			Value2: "ok",
		},
		"test_amber": {
			Status: monitors.RAGAmber,
			Group:  "test",
			Value:  "75%",
			Value2: "warning",
		},
		"test_red": {
			Status: monitors.RAGRed,
			Group:  "test",
			Value:  "95%",
			Value2: "critical",
		},
		"test_error": {
			Status: monitors.RAGError,
			Group:  "test",
			Value:  "err",
			Value2: "failed",
		},
		"test_unknown": {
			Status: monitors.RAGUnknown,
			Group:  "test",
			Value:  "",
			Value2: "",
		},
	}

	metrics := buildMetrics(results)
	assert.Len(t, metrics, 5, "should have 5 metrics")

	statusMap := make(map[string]string)
	for _, m := range metrics {
		statusMap[m.ID] = m.Status
	}

	assert.Equal(t, "Green", statusMap["test_green"])
	assert.Equal(t, "Amber", statusMap["test_amber"])
	assert.Equal(t, "Red", statusMap["test_red"])
	assert.Equal(t, "Error", statusMap["test_error"])
	assert.Equal(t, "Unknown", statusMap["test_unknown"])
}

// TestMetricsEmptyResults verifies that an empty monitors array is returned (not nil) when no monitors are configured.
func TestMetricsEmptyResults(t *testing.T) {
	// Test via buildMetrics with an empty results map
	metrics := buildMetrics(map[string]monitors.Result{})
	assert.NotNil(t, metrics, "metrics should not be nil even when empty")
	assert.Len(t, metrics, 0, "metrics should be empty")

	// Also verify through the HTTP endpoint by checking the raw JSON
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	// The test server adds CPU and memory monitors by default, so we just verify
	// that the monitors field is an array (not null) by checking the JSON structure.
	time.Sleep(50 * time.Millisecond)

	r, body := GetTestRequest(ctx, t, s, "/metrics")
	assert.Equal(t, http.StatusOK, r.StatusCode, "status code")

	// Parse as raw JSON to verify structure
	var raw map[string]json.RawMessage
	err := json.Unmarshal([]byte(body), &raw)
	assert.NoError(t, err, "unmarshal raw JSON")

	// Verify "monitors" field exists and is a JSON array (starts with '[')
	monitorsJSON, ok := raw["monitors"]
	assert.True(t, ok, "monitors field should exist in JSON response")
	assert.True(t, len(monitorsJSON) > 0 && monitorsJSON[0] == '[', "monitors should be a JSON array, not null")
}

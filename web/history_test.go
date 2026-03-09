package web

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"lmon/db"
)

// TestHistoryAPIUnavailable verifies that GET /api/history returns 503 when no store is configured.
func TestHistoryAPIUnavailable(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	// No store set, should return 503
	r, _ := GetTestRequest(ctx, t, s, "/api/history")
	assert.Equal(t, http.StatusServiceUnavailable, r.StatusCode, "should return 503 when DB unavailable")
}

// TestSummaryAPIUnavailable verifies that GET /api/summary returns 503 when no store is configured.
func TestSummaryAPIUnavailable(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	r, _ := GetTestRequest(ctx, t, s, "/api/summary")
	assert.Equal(t, http.StatusServiceUnavailable, r.StatusCode, "should return 503 when DB unavailable")
}

// TestHistoryAPIWithNoopStore verifies that GET /api/history returns 503 with a noop store.
func TestHistoryAPIWithNoopStore(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.SetStore(db.NewNoopStore())
	s.Start(ctx)

	r, _ := GetTestRequest(ctx, t, s, "/api/history")
	assert.Equal(t, http.StatusServiceUnavailable, r.StatusCode, "noop store should be unavailable")
}

// TestHistoryPageUnavailable verifies that GET /history renders the unavailable message.
func TestHistoryPageUnavailable(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.Start(ctx)

	r, body := GetTestRequest(ctx, t, s, "/history")
	assert.Equal(t, http.StatusOK, r.StatusCode, "history page should return 200 even without DB")
	assert.Contains(t, body, "Database is not configured", "should show unavailable message")
}

// TestHistoryPageWithNoopStore verifies that the history page shows a friendly message with a noop store.
func TestHistoryPageWithNoopStore(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")
	s.SetStore(db.NewNoopStore())
	s.Start(ctx)

	r, body := GetTestRequest(ctx, t, s, "/history")
	assert.Equal(t, http.StatusOK, r.StatusCode)
	assert.Contains(t, body, "Database is not configured")
}

// mockStore implements db.Store for web handler testing.
type mockStore struct {
	snapshots []db.MonitorSnapshot
	summaries []db.MonitorSummary
	available bool
}

func (m *mockStore) SaveSnapshots(_ context.Context, snapshots []db.MonitorSnapshot) error {
	m.snapshots = append(m.snapshots, snapshots...)
	return nil
}

func (m *mockStore) GetHistory(_ context.Context, _, _ string, _, _ time.Time, limit int) ([]db.MonitorSnapshot, error) {
	if limit > len(m.snapshots) {
		limit = len(m.snapshots)
	}
	return m.snapshots[:limit], nil
}

func (m *mockStore) GetSummary(_ context.Context, _, _ time.Time) ([]db.MonitorSummary, error) {
	return m.summaries, nil
}

func (m *mockStore) PurgeOlderThan(_ context.Context, _ time.Time, _ int) (int64, error) {
	return 0, nil
}

func (m *mockStore) CompactOlderThan(_ context.Context, _, _ time.Time, _, _ int) (int64, error) {
	return 0, nil
}

func (m *mockStore) Close() error {
	return nil
}

func (m *mockStore) IsAvailable() bool {
	return m.available
}

// TestHistoryAPIWithMockStore verifies that GET /api/history returns snapshots from the store.
func TestHistoryAPIWithMockStore(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")

	now := time.Now().UTC()
	store := &mockStore{
		available: true,
		snapshots: []db.MonitorSnapshot{
			{ID: 1, RecordedAt: now, Node: "node1", MonitorID: "cpu", MonitorType: "system", Status: "Green", Message: "ok"},
			{ID: 2, RecordedAt: now, Node: "node1", MonitorID: "disk_root", MonitorType: "disk", Status: "Amber", Message: "warning"},
		},
	}
	s.SetStore(store)
	s.Start(ctx)

	r, body := GetTestRequest(ctx, t, s, "/api/history")
	assert.Equal(t, http.StatusOK, r.StatusCode, "status code")
	assert.Contains(t, r.Header.Get("Content-Type"), "application/json")

	var snapshots []db.MonitorSnapshot
	err := json.Unmarshal([]byte(body), &snapshots)
	require.NoError(t, err, "unmarshal snapshots")
	assert.Len(t, snapshots, 2, "should return 2 snapshots")
}

// TestSummaryAPIWithMockStore verifies that GET /api/summary returns aggregated data.
func TestSummaryAPIWithMockStore(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")

	store := &mockStore{
		available: true,
		summaries: []db.MonitorSummary{
			{Node: "node1", MonitorID: "cpu", MonitorType: "system", GreenCount: 10, AmberCount: 2, RedCount: 1, ErrorCount: 0, TotalCount: 13},
		},
	}
	s.SetStore(store)
	s.Start(ctx)

	r, body := GetTestRequest(ctx, t, s, "/api/summary")
	assert.Equal(t, http.StatusOK, r.StatusCode, "status code")
	assert.Contains(t, r.Header.Get("Content-Type"), "application/json")

	var summaries []db.MonitorSummary
	err := json.Unmarshal([]byte(body), &summaries)
	require.NoError(t, err, "unmarshal summaries")
	assert.Len(t, summaries, 1, "should return 1 summary")
	assert.Equal(t, int64(10), summaries[0].GreenCount)
	assert.Equal(t, int64(13), summaries[0].TotalCount)
}

// TestHistoryAPIWithQueryParams verifies that query parameters are accepted.
func TestHistoryAPIWithQueryParams(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")

	store := &mockStore{
		available: true,
		snapshots: []db.MonitorSnapshot{
			{ID: 1, RecordedAt: time.Now(), Node: "node1", MonitorID: "cpu", MonitorType: "system", Status: "Green"},
		},
	}
	s.SetStore(store)
	s.Start(ctx)

	// Test with various query params
	from := time.Now().Add(-48 * time.Hour).Format(time.RFC3339)
	to := time.Now().Format(time.RFC3339)
	path := "/api/history?node=node1&monitor=cpu&from=" + from + "&to=" + to + "&limit=10"

	r, body := GetTestRequest(ctx, t, s, path)
	assert.Equal(t, http.StatusOK, r.StatusCode)

	var snapshots []db.MonitorSnapshot
	err := json.Unmarshal([]byte(body), &snapshots)
	require.NoError(t, err)
	assert.Len(t, snapshots, 1)
}

// TestHistoryPageWithMockStore verifies the history page renders with data from the store.
func TestHistoryPageWithMockStore(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")

	now := time.Now().UTC()
	store := &mockStore{
		available: true,
		snapshots: []db.MonitorSnapshot{
			{ID: 1, RecordedAt: now, Node: "node1", MonitorID: "cpu", MonitorType: "system", Status: "Green", Message: "ok"},
		},
	}
	s.SetStore(store)
	s.Start(ctx)

	r, body := GetTestRequest(ctx, t, s, "/history")
	assert.Equal(t, http.StatusOK, r.StatusCode, "history page should return 200")
	assert.Contains(t, body, "Monitor History", "should contain page title")
	assert.Contains(t, body, "Sparkline", "should contain sparkline section")
	assert.Contains(t, body, "svg", "should contain SVG sparkline")
}

// TestSummaryAPIWithQueryParams verifies that query parameters are accepted for summary.
func TestSummaryAPIWithQueryParams(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")

	store := &mockStore{
		available: true,
		summaries: []db.MonitorSummary{
			{Node: "node1", MonitorID: "cpu", MonitorType: "system", GreenCount: 5, TotalCount: 5},
		},
	}
	s.SetStore(store)
	s.Start(ctx)

	from := time.Now().Add(-48 * time.Hour).Format(time.RFC3339)
	to := time.Now().Format(time.RFC3339)
	path := "/api/summary?from=" + from + "&to=" + to

	r, body := GetTestRequest(ctx, t, s, path)
	assert.Equal(t, http.StatusOK, r.StatusCode)

	var summaries []db.MonitorSummary
	err := json.Unmarshal([]byte(body), &summaries)
	require.NoError(t, err)
	assert.Len(t, summaries, 1)
}

// TestHistoryPageWithQueryParams verifies that the history page accepts node and monitor query params.
func TestHistoryPageWithQueryParams(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")

	now := time.Now().UTC()
	store := &mockStore{
		available: true,
		snapshots: []db.MonitorSnapshot{
			{ID: 1, RecordedAt: now, Node: "node1", MonitorID: "cpu", MonitorType: "system", Status: "Green", Message: "ok"},
		},
	}
	s.SetStore(store)
	s.Start(ctx)

	from := now.Add(-2 * time.Hour).Format(time.RFC3339)
	to := now.Add(time.Hour).Format(time.RFC3339)
	path := "/history?node=node1&monitor=cpu&from=" + from + "&to=" + to

	r, body := GetTestRequest(ctx, t, s, path)
	assert.Equal(t, http.StatusOK, r.StatusCode)
	assert.Contains(t, body, "Monitor History")
}

// TestSetStoreAndSetWriter verifies the setter methods on the Server.
func TestSetStoreAndSetWriter(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s, _ := StartTestServer(ctx, t, "")

	// Initially nil
	assert.Nil(t, s.store, "store should be nil initially")
	assert.Nil(t, s.writer, "writer should be nil initially")

	// Set store
	store := db.NewNoopStore()
	s.SetStore(store)
	assert.NotNil(t, s.store, "store should be set")

	// Set writer
	writer := db.NewBufferedWriter(store, 10, 0)
	s.SetWriter(writer)
	assert.NotNil(t, s.writer, "writer should be set")
	writer.Close()
}

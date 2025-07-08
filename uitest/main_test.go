package uitest

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/stretchr/testify/assert"

	"lmon/web"
)

// TestServerHealth tests that the server is healthy
func TestServerHealth(t *testing.T) {
	ctx, canncel := context.WithCancel(t.Context())
	defer canncel()

	s, _ := web.StartTestServer(ctx, t, "")
	s.Start(ctx)

	resp, body := web.GetTestRequest(ctx, t, s, "/healthz")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, body, "body returned")
	_ = resp.Body.Close()
}

// TestDefaultConfigUIRod verifies the UI for the default config using rod: green CPU, green Memory, no disk/healthcheck items.
func TestDefaultConfigUIRod(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	s, _ := web.StartTestServer(ctx, t, "")
	s.Start(ctx)

	browser := rod.New().MustConnect()
	defer browser.MustClose()
	page := browser.MustPage(s.ServerUrl)
	defer page.MustClose()

	// Wait for CPU and Memory items to appear by data-id
	page.Timeout(1 * time.Second).MustElement(`#system-items .list-group-item[data-id="system_cpu"]`)
	page.Timeout(1 * time.Second).MustElement(`#system-items .list-group-item[data-id="system_mem"]`)

	// Check for green status on CPU
	cpuItem := page.MustElement(`#system-items .list-group-item[data-id="system_cpu"]`)
	cpuGreen := cpuItem.MustHas(".status-indicator.status-ok")
	assert.True(t, cpuGreen, "CPU item is green")

	// Check for green status on Memory
	memItem := page.MustElement(`#system-items .list-group-item[data-id="system_mem"]`)
	memGreen := memItem.MustHas(".status-indicator.status-ok")
	assert.True(t, memGreen, "Memory item is green")

	// Disk card should show "No items to display"
	diskText := page.MustElement("#disk-items").MustText()
	assert.Contains(t, diskText, "No items", "No disk items in default config")

	// Health check card should show "No items to display"
	healthText := page.MustElement("#health-items").MustText()
	assert.Contains(t, healthText, "No items", "No healthcheck items in default config")
}

func TestAddDiskViaConfigUIRod(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	s, _ := web.StartTestServer(ctx, t, "")
	s.Start(ctx)

	browser := rod.New().MustConnect()
	defer browser.MustClose()
	page := browser.MustPage(s.ServerUrl)
	defer page.MustClose()

	// Navigate to the config tab
	page.MustElement(`a.nav-link[href="/config"]`).MustClick()
	// Wait for the config form to appear
	page.Timeout(1 * time.Second).MustElement(`#add-disk-form`)

	// Fill out the Add Disk Monitor form
	page.MustElement(`#disk-name`).MustInput("root")
	page.MustElement(`#disk-path`).MustInput("/")
	// Optionally set threshold or icon if needed

	// Submit the form
	page.MustElement(`#add-disk-form button[type="submit"]`).MustClick()

	// Wait for the disk to appear in the config list
	page.Timeout(1*time.Second).MustElementR("#disk-config-items .config-item", "root.*\\(/\\)")

	// Navigate back to dashboard
	page.MustElement(`a.nav-link[href="/"]`).MustClick()
	// Wait for dashboard system items to appear
	page.Timeout(1 * time.Second).MustElement(`#system-items`)

	// Wait for the disk item to appear in the dashboard
	page.Timeout(1 * time.Second).MustElement(`#disk-items .list-group-item[data-id="disk_root"]`)

	// Assert its presence
	diskItem := page.MustElement(`#disk-items .list-group-item[data-id="disk_root"]`)
	assert.NotNil(t, diskItem, "Disk item 'root' is present in dashboard")
}

func TestAddHealthCheckViaConfigUIRod(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	s, _ := web.StartTestServer(ctx, t, "")
	s.Start(ctx)

	browser := rod.New().MustConnect()
	defer browser.MustClose()
	page := browser.MustPage(s.ServerUrl)
	defer page.MustClose()

	// Navigate to the config tab
	page.MustElement(`a.nav-link[href="/config"]`).MustClick()
	// Wait for the health check form to appear
	page.Timeout(5 * time.Second).MustElement(`#add-health-form`)

	// Fill out the Add Health Check form
	page.MustElement(`#health-name`).MustInput("local")
	page.MustElement(`#health-url`).MustInput("http://localhost:8080")
	page.MustElement(`#health-timeout`).MustInput("10")
	// Optionally set icon if needed (leave as default)

	// Submit the form
	page.MustElement(`#add-health-form button[type="submit"]`).MustClick()

	// Wait for the health check to appear in the config list
	page.Timeout(1*time.Second).MustElementR("#health-config-items .config-item", "local")

	// Navigate back to dashboard
	page.MustElement(`a.nav-link[href="/"]`).MustClick()
	// Wait for dashboard health items to appear
	page.Timeout(1 * time.Second).MustElement(`#health-items`)

	// Wait for the health check item to appear in the dashboard
	page.Timeout(1 * time.Second).MustElement(`#health-items .list-group-item[data-id="health_local"]`)

	// Assert its presence
	healthItem := page.MustElement(`#health-items .list-group-item[data-id="health_local"]`)
	assert.NotNil(t, healthItem, "Health check item 'local' is present in dashboard")
}

package uitest

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
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

func getBrowser() *rod.Browser {
	var browser *rod.Browser
	if os.Getenv("CI") != "" {
		u := launcher.New().Set("no-sandbox").MustLaunch()
		browser = rod.New().ControlURL(u).MustConnect()
	} else {
		browser = rod.New().MustConnect()
	}
	return browser
}

// TestDefaultConfigUIRod verifies the UI for the default config using rod: green CPU, green Memory, no disk/healthcheck items.
func TestDefaultConfigUIRod(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	s, _ := web.StartTestServer(ctx, t, "")
	s.Start(ctx)

	browser := getBrowser()
	defer browser.MustClose()
	page := browser.MustPage(s.ServerUrl)
	defer page.MustClose()

	// Wait for CPU and Memory items to appear by data-id
	page.Timeout(1 * time.Second).MustElement(`#system-items .list-group-item[data-id="system_cpu"]`)
	page.Timeout(1 * time.Second).MustElement(`#system-items .list-group-item[data-id="system_mem"]`)

	// Check for green status on CPU
	cpuItem := page.MustElement(`#system-items .list-group-item[data-id="system_cpu"]`)
	assert.Contains(t, cpuItem.MustText(), "cpu", "CPU display name is shown")
	assert.Contains(t, cpuItem.MustText(), "50.0%", "CPU value is shown")

	cpuGreen := cpuItem.MustHas(".status-indicator.status-ok")
	assert.True(t, cpuGreen, "CPU item is green")

	// Check for green status on Memory
	memItem := page.MustElement(`#system-items .list-group-item[data-id="system_mem"]`)
	assert.Contains(t, memItem.MustText(), "mem", "Memory display name is shown")
	assert.Contains(t, memItem.MustText(), "50.0%", "Memory value is shown")
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

	browser := getBrowser()
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

	// Assert its presence on the dashboar
	page.MustElement(`a.nav-link[href="/"]`).MustClick()
	diskItem := page.MustElement(`#disk-items .list-group-item[data-id="disk_root"]`)
	assert.NotNil(t, diskItem, "Disk item 'root' is present in dashboard")
	diskText := diskItem.MustText()
	assert.Contains(t, diskText, "root (/)", "Disk display name is shown")
	assert.Regexp(t, `\d+(\.\d+)?%`, diskText, "Disk value is shown")

	// --- MOBILE PAGE CHECKS ---
	page.MustElement(`a.nav-link[href="/mobile"]`).MustClick()
	page.Timeout(1 * time.Second).MustElement(`#mobile-items-list`)
	diskMobile := page.MustElement(`#mobile-items-list .mobile-list-item[data-id="disk_root"]`)
	diskMobileText := diskMobile.MustText()
	assert.Contains(t, diskMobileText, "root (/)", "Disk display name is shown on mobile")
	assert.Regexp(t, `\d+(\.\d+)?%`, diskMobileText, "Disk value is shown on mobile")

	// Go back to config and delete the disk
	page.MustElement(`a.nav-link[href="/config"]`).MustClick()
	page.Timeout(1 * time.Second).MustElement(`#disk-config-items`)
	page.MustElementR(`#disk-config-items .config-item`, "root.*\\(/\\)")
	el := page.Timeout(1 * time.Second).MustElement(`button.delete-btn[data-type="disk"][data-id="root"]`)
	wait, handle := page.HandleDialog()
	go el.MustClick()
	_ = wait()
	_ = handle(&proto.PageHandleJavaScriptDialog{Accept: true})

	// Wait for disk to be removed from config list
	page.Timeout(1*time.Second).MustElementR(`#disk-config-items`, "No disk monitors configured")

	// Go back to dashboard and verify disk is gone
	page.MustElement(`a.nav-link[href="/"]`).MustClick()
	page.Timeout(1 * time.Second).MustElement(`#disk-items`)
	assert.Panics(t, func() {
		page.Timeout(1 * time.Second).MustElement(`#disk-items .list-group-item[data-id="disk_root"]`)
	}, "Disk item 'root' should be gone from dashboard")
}

func TestAddHealthCheckViaConfigUIRod(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	s, _ := web.StartTestServer(ctx, t, "")
	s.Start(ctx)

	browser := getBrowser()
	defer browser.MustClose()
	page := browser.MustPage(s.ServerUrl)
	defer page.MustClose()

	// Navigate to the config tab
	page.MustElement(`a.nav-link[href="/config"]`).MustClick()
	// Wait for the health check form to appear
	page.Timeout(1 * time.Second).MustElement(`#add-health-form`)

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
	healthText := healthItem.MustText()
	assert.Contains(t, healthText, "local", "Health check display name is shown")
	assert.Contains(t, healthText, "http://localhost:8080", "Health check URL is shown")
	assert.Regexp(t, `\d+(\.\d+)?`, healthText, "Health check value is shown")
	// healthItem.MustClick()
	// modal := page.MustElement("#itemDetailModal")
	// modalText := modal.MustText()
	// assert.Contains(t, modalText, "local", "Modal shows display name")
	// assert.Contains(t, modalText, "http://localhost:8080", "Modal shows URL")
	// assert.Contains(t, modalText, "Value", "Modal shows value")
	// assert.Contains(t, modalText, "Threshold", "Modal shows threshold")
	// modal.MustClick()

	// --- MOBILE PAGE CHECKS ---
	page.MustElement(`a.nav-link[href="/mobile"]`).MustClick()
	page.Timeout(1 * time.Second).MustElement(`#mobile-items-list`)
	healthMobile := page.MustElement(`#mobile-items-list .mobile-list-item[data-id="health_local"]`)
	healthMobileText := healthMobile.MustText()
	assert.Contains(t, healthMobileText, "local", "Health check display name is shown on mobile")
	assert.Contains(t, healthMobileText, "http://localhost:8080", "Health check URL is shown on mobile")
	assert.Regexp(t, `\d+(\.\d+)?`, healthMobileText, "Health check value is shown on mobile")

	// healthMobile.MustClick()
	// modalMobile := page.MustElement("#itemDetailModal")
	// modalMobileText := modalMobile.MustText()
	// assert.Contains(t, modalMobileText, "local", "Modal shows display name on mobile")
	// assert.Contains(t, modalMobileText, "http://localhost:8080", "Modal shows URL on mobile")
	// assert.Contains(t, modalMobileText, "Value", "Modal shows value on mobile")
	// assert.Contains(t, modalMobileText, "Threshold", "Modal shows threshold on mobile")

	// Go back to config and delete the health check
	page.MustElement(`a.nav-link[href="/config"]`).MustClick()
	page.Timeout(1 * time.Second).MustElement(`#health-config-items`)
	page.MustElementR(`#health-config-items .config-item`, "local")
	el := page.Timeout(1 * time.Second).MustElement(`button.delete-btn[data-type="health"][data-id="local"]`)
	wait, handle := page.HandleDialog()
	go el.MustClick()
	_ = wait()
	_ = handle(&proto.PageHandleJavaScriptDialog{Accept: true})

	// Wait for health check to be removed from config list
	page.Timeout(1*time.Second).MustElementR(`#health-config-items`, "No health checks configured")

	// Go back to dashboard and verify health check is gone
	page.MustElement(`a.nav-link[href="/"]`).MustClick()
	page.Timeout(1 * time.Second).MustElement(`#health-items`)
	assert.Panics(t, func() {
		page.Timeout(1 * time.Second).MustElement(`#health-items .list-group-item[data-id="health_local"]`)
	}, "Health check item 'local' should be gone from dashboard")
}

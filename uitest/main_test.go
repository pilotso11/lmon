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
	"github.com/stretchr/testify/require"

	"lmon/web"
)

func requireNoErrorWithScreenshot(t *testing.T, page *rod.Page, err error, s string) {
	t.Helper()
	if err != nil {
		page.MustScreenshot("test-error.png")
	}
	require.NoError(t, err, s)
}

func closeBrowser(browser *rod.Browser) {
	_ = browser.Close()
}

func closePage(page *rod.Page) {
	_ = page.Close()
}

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

func getBrowser(t *testing.T) *rod.Browser {
	l := launcher.New()
	if bin := os.Getenv("ROD_BROWSER"); bin != "" {
		l = l.Bin(bin)
	}
	if os.Getenv("CI") != "" {
		l = l.Set("no-sandbox")
	}
	l = l.Headless(true)
	u, err := l.Launch()
	require.NoError(t, err, "rod launch")
	browser := rod.New().ControlURL(u).MustConnect()
	return browser
}

// TestDefaultConfigUIRod verifies the UI for the default config using rod: green CPU, green Memory, no disk/healthcheck items.
func TestDefaultConfigUIRod(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	s, _ := web.StartTestServer(ctx, t, "")
	s.Start(ctx)

	browser := getBrowser(t)
	defer closeBrowser(browser)
	page, err := browser.Page(proto.TargetCreateTarget{URL: s.ServerUrl})
	require.NoError(t, err, "open page")
	defer closePage(page)

	// Wait for CPU and Memory items to appear by data-id
	_, err = page.Timeout(1 * time.Second).Element(`#system-items .list-group-item[data-id="system_cpu"]`)
	requireNoErrorWithScreenshot(t, page, err, "wait for CPU item")
	_, err = page.Timeout(1 * time.Second).Element(`#system-items .list-group-item[data-id="system_mem"]`)
	requireNoErrorWithScreenshot(t, page, err, "wait for Memory item")

	// Check for green status on CPU
	cpuItem, err := page.Element(`#system-items .list-group-item[data-id="system_cpu"]`)
	requireNoErrorWithScreenshot(t, page, err, "find CPU item")
	cpuText, err := cpuItem.Text()
	requireNoErrorWithScreenshot(t, page, err, "get CPU text")
	assert.Contains(t, cpuText, "cpu", "CPU display name is shown")
	assert.Contains(t, cpuText, "50.0%", "CPU value is shown")

	cpuGreen, _, err := cpuItem.Has(".status-indicator.status-ok")
	requireNoErrorWithScreenshot(t, page, err, "check CPU green status")
	assert.True(t, cpuGreen, "CPU item is green")

	// Check for green status on Memory
	memItem, err := page.Element(`#system-items .list-group-item[data-id="system_mem"]`)
	requireNoErrorWithScreenshot(t, page, err, "find Memory item")
	memText, err := memItem.Text()
	requireNoErrorWithScreenshot(t, page, err, "get Memory text")
	assert.Contains(t, memText, "mem", "Memory display name is shown")
	assert.Contains(t, memText, "50.0%", "Memory value is shown")
	memGreen, _, err := memItem.Has(".status-indicator.status-ok")
	requireNoErrorWithScreenshot(t, page, err, "check Memory green status")
	assert.True(t, memGreen, "Memory item is green")

	// Disk card should show "No items to display"
	diskItem, err := page.Element("#disk-items")
	requireNoErrorWithScreenshot(t, page, err, "find disk-items")
	diskText, err := diskItem.Text()
	requireNoErrorWithScreenshot(t, page, err, "get disk-items text")
	assert.Contains(t, diskText, "No items", "No disk items in default config")

	// Health check card should show "No items to display"
	healthItem, err := page.Element("#health-items")
	requireNoErrorWithScreenshot(t, page, err, "find health-items")
	healthText, err := healthItem.Text()
	requireNoErrorWithScreenshot(t, page, err, "get health-items text")
	assert.Contains(t, healthText, "No items", "No healthcheck items in default config")
}

func TestAddDiskViaConfigUIRod(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	s, _ := web.StartTestServer(ctx, t, "")
	s.Start(ctx)

	browser := getBrowser(t)
	defer closeBrowser(browser)
	page, err := browser.Page(proto.TargetCreateTarget{URL: s.ServerUrl})
	require.NoError(t, err, "open page")
	defer closePage(page)

	// Navigate to the config tab
	el, err := page.Element(`a.nav-link[href="/config"]`)
	requireNoErrorWithScreenshot(t, page, err, "find config tab")
	el.MustClick()
	// Wait for the config form to appear
	_, err = page.Timeout(1 * time.Second).Element(`#add-disk-form`)
	requireNoErrorWithScreenshot(t, page, err, "wait for add-disk-form")

	// Fill out the Add Disk Monitor form
	nameEl, err := page.Element(`#disk-name`)
	requireNoErrorWithScreenshot(t, page, err, "find disk-name input")
	err = nameEl.Input("root")
	requireNoErrorWithScreenshot(t, page, err, "input disk-name")
	pathEl, err := page.Element(`#disk-path`)
	requireNoErrorWithScreenshot(t, page, err, "find disk-path input")
	err = pathEl.Input("/")
	requireNoErrorWithScreenshot(t, page, err, "input disk-path")
	// Optionally set threshold or icon if needed

	// Submit the form
	submitEl, err := page.Element(`#add-disk-form button[type="submit"]`)
	requireNoErrorWithScreenshot(t, page, err, "find submit button")
	submitEl.MustClick()

	// Wait for the disk to appear in the config list
	// The config list now uses two divs per item, and may require a refresh for the new item to appear.
	// Refresh the page to ensure the new disk appears.
	err = page.Timeout(time.Second).Reload()
	requireNoErrorWithScreenshot(t, page, err, "wait for reload after adding disk")
	// Instead of using ElementR with a regex, explicitly check for the presence of both spans.
	// Wait for a config item with disk name "root" and path "/"
	require.Eventually(t, func() bool {
		items, err := page.Elements(`#disk-config-items .config-item`)
		if err != nil {
			return false // If we can't get items, return false to retry
		}
		for _, item := range items {
			nameSpan, err := item.Element(`.config-item-name`)
			if err != nil {
				continue
			}
			nameText, err := nameSpan.Text()
			if err != nil {
				continue
			}
			pathSpan, err := item.Element(`.config-item-path`)
			if err != nil {
				continue
			}
			pathText, err := pathSpan.Text()
			if err != nil {
				continue
			}
			if nameText == "root" && pathText == "(/)" {
				return true
			}
		}
		return false
	}, 2*time.Second, 50*time.Millisecond, "wait for disk item in config list")

	// Navigate back to dashboard
	el, err = page.Element(`a.nav-link[href="/"]`)
	requireNoErrorWithScreenshot(t, page, err, "find dashboard tab")
	el.MustClick()
	// Wait for dashboard system items to appear
	_, err = page.Timeout(1 * time.Second).Element(`#system-items`)
	requireNoErrorWithScreenshot(t, page, err, "wait for system-items")

	// Wait for the disk item to appear in the dashboard
	_, err = page.Timeout(1 * time.Second).Element(`#disk-items .list-group-item[data-id="disk_root"]`)
	requireNoErrorWithScreenshot(t, page, err, "wait for disk_root in dashboard")

	// Assert its presence on the dashboard
	el, err = page.Element(`a.nav-link[href="/"]`)
	requireNoErrorWithScreenshot(t, page, err, "find dashboard tab (again)")
	el.MustClick()
	diskItem, err := page.Element(`#disk-items .list-group-item[data-id="disk_root"]`)
	requireNoErrorWithScreenshot(t, page, err, "find disk_root item")
	assert.NotNil(t, diskItem, "Disk item 'root' is present in dashboard")
	diskText, err := diskItem.Text()
	requireNoErrorWithScreenshot(t, page, err, "get disk_root text")
	assert.Contains(t, diskText, "root (/)", "Disk display name is shown")
	assert.Regexp(t, `\d+(\.\d+)?%`, diskText, "Disk value is shown")

	// --- MOBILE PAGE CHECKS ---
	el, err = page.Element(`a.nav-link[href="/mobile"]`)
	requireNoErrorWithScreenshot(t, page, err, "find mobile tab")
	el.MustClick()
	_, err = page.Timeout(1 * time.Second).Element(`#mobile-items-list`)
	requireNoErrorWithScreenshot(t, page, err, "wait for mobile-items-list")
	diskMobile, err := page.Element(`#mobile-items-list .mobile-list-item[data-id="disk_root"]`)
	requireNoErrorWithScreenshot(t, page, err, "find disk_root on mobile")
	diskMobileText, err := diskMobile.Text()
	requireNoErrorWithScreenshot(t, page, err, "get disk_root text on mobile")
	assert.Contains(t, diskMobileText, "root (/)", "Disk display name is shown on mobile")
	assert.Regexp(t, `\d+(\.\d+)?%`, diskMobileText, "Disk value is shown on mobile")

	// Go back to config and delete the disk
	el, err = page.Element(`a.nav-link[href="/config"]`)
	requireNoErrorWithScreenshot(t, page, err, "find config tab (delete)")
	el.MustClick()

	delEl, err := page.Timeout(1 * time.Second).Element(`button.delete-disk-btn[data-id="root"]`)
	requireNoErrorWithScreenshot(t, page, err, "find delete button for disk_root")
	wait, handle := page.HandleDialog()
	go delEl.MustClick()
	_ = wait()
	_ = handle(&proto.PageHandleJavaScriptDialog{Accept: true})

	// Wait for disk to be removed from config list
	_, err = page.Timeout(1*time.Second).ElementR(`#disk-config-items`, "No disk monitors configured")
	requireNoErrorWithScreenshot(t, page, err, "wait for disk removed from config list")

	// Go back to dashboard and verify disk is gone
	el, err = page.Element(`a.nav-link[href="/"]`)
	requireNoErrorWithScreenshot(t, page, err, "find dashboard tab (final)")
	el.MustClick()
	_, err = page.Timeout(1 * time.Second).Element(`#disk-items`)
	requireNoErrorWithScreenshot(t, page, err, "wait for disk-items")
	assert.Panics(t, func() {
		page.Timeout(1 * time.Second).MustElement(`#disk-items .list-group-item[data-id="disk_root"]`)
	}, "Disk item 'root' should be gone from dashboard")
}

func TestAddHealthCheckViaConfigUIRod(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	s, _ := web.StartTestServer(ctx, t, "")
	s.Start(ctx)

	browser := getBrowser(t)
	defer closeBrowser(browser)
	page, err := browser.Page(proto.TargetCreateTarget{URL: s.ServerUrl})
	require.NoError(t, err, "open page")
	defer closePage(page)

	// Navigate to the config tab
	// err := page.WaitStable(time.Second)
	// requireNoErrorWithScreenshot(t, page, err, "page wait stable")

	// Click on the config link
	el, err := page.Element(`a.nav-link[href="/config"]`)
	requireNoErrorWithScreenshot(t, page, err, "find config link")
	el.MustClick()

	// Wait for the unified monitor form to appear
	_, err = page.Timeout(1 * time.Second).Element(`#add-monitor-form`)
	requireNoErrorWithScreenshot(t, page, err, "find monitor form")

	// HTTP should be selected by default, but verify
	httpRadio, err := page.Element(`#monitor-type-http`)
	requireNoErrorWithScreenshot(t, page, err, "find http radio button")
	if !httpRadio.MustProperty("checked").Bool() {
		httpLabel, err := page.Element(`label[for="monitor-type-http"]`)
		requireNoErrorWithScreenshot(t, page, err, "find http radio button label")
		httpLabel.MustClick()
	}

	// Fill out the Add Health Check form
	el, err = page.Element(`#monitor-name`)
	requireNoErrorWithScreenshot(t, page, err, "find monitor name input")
	_ = el.MustInput("local")
	el, err = page.Element(`#monitor-target`)
	requireNoErrorWithScreenshot(t, page, err, "find monitor target input")
	_ = el.Input("http://localhost:8080")
	el, err = page.Element(`#monitor-timeout`)
	requireNoErrorWithScreenshot(t, page, err, "find monitor timeout input")
	el.MustSelectAllText().MustInput("10")
	el, err = page.Element(`#monitor-respcode`)
	requireNoErrorWithScreenshot(t, page, err, "input response code")
	el.MustSelectAllText().MustInput("401")

	// Submit the form
	el, err = page.Element(`#add-monitor-form button[type="submit"]`)
	requireNoErrorWithScreenshot(t, page, err, "find monitor submit button")
	el.MustClick()

	// Wait for the health check to appear in the config list
	_, err = page.Timeout(1*time.Second).ElementR("#health-config-items .config-item", "local")
	requireNoErrorWithScreenshot(t, page, err, "find health check item in config list")

	// Navigate back to dashboard
	el, err = page.Element(`a.nav-link[href="/"]`)
	requireNoErrorWithScreenshot(t, page, err, "find dashboard link")
	el.MustClick()

	// Wait for dashboard health items to appear
	_, err = page.Timeout(1 * time.Second).Element(`#health-items`)
	requireNoErrorWithScreenshot(t, page, err, "find health items")

	// Wait for the health check item to appear in the dashboard
	_, err = page.Timeout(1 * time.Second).Element(`#health-items .list-group-item[data-id="health_local"]`)
	requireNoErrorWithScreenshot(t, page, err, "find health check item in dashboard")

	// Assert its presence
	healthItem, err := page.Element(`#health-items .list-group-item[data-id="health_local"]`)
	requireNoErrorWithScreenshot(t, page, err, "find health check item in dashboard")
	assert.NotNil(t, healthItem, "Health check item 'local' is present in dashboard")
	healthText := healthItem.MustText()
	assert.Contains(t, healthText, "local", "Health check display name is shown")
	assert.Contains(t, healthText, "http://localhost:8080", "Health check URL is shown")
	assert.Contains(t, healthText, "401", "Health check expected code is shown")
	assert.Contains(t, healthText, "200 (OK)", "Health check status is shown")
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
	el, err = page.Element(`a.nav-link[href="/mobile"]`)
	requireNoErrorWithScreenshot(t, page, err, "find mobile link")
	el.MustClick()

	page.Timeout(1 * time.Second).MustElement(`#mobile-items-list`)
	healthMobile := page.MustElement(`#mobile-items-list .mobile-list-item[data-id="health_local"]`)
	healthMobileText := healthMobile.MustText()
	assert.Contains(t, healthMobileText, "local", "Health check display name is shown on mobile")
	assert.Contains(t, healthMobileText, "http://localhost:8080", "Health check URL is shown on mobile")
	assert.Contains(t, healthMobileText, "200 (OK)", "Health status on mobile")
	assert.Regexp(t, `\d+(\.\d+)?`, healthMobileText, "Health check value is shown on mobile")

	// healthMobile.MustClick()
	// modalMobile := page.MustElement("#itemDetailModal")
	// modalMobileText := modalMobile.MustText()
	// assert.Contains(t, modalMobileText, "local", "Modal shows display name on mobile")
	// assert.Contains(t, modalMobileText, "http://localhost:8080", "Modal shows URL on mobile")
	// assert.Contains(t, modalMobileText, "Value", "Modal shows value on mobile")
	// assert.Contains(t, modalMobileText, "Threshold", "Modal shows threshold on mobile")

	// Go back to config and delete the health check
	el, err = page.Element(`a.nav-link[href="/config"]`)
	requireNoErrorWithScreenshot(t, page, err, "find config link")
	el.MustClick()
	_, err = page.Timeout(1 * time.Second).Element(`#health-config-items`)
	requireNoErrorWithScreenshot(t, page, err, "find health check config items")
	_, err = page.ElementR(`#health-config-items .config-item`, "local")
	requireNoErrorWithScreenshot(t, page, err, "find health check item in config list")
	el, err = page.Timeout(1 * time.Second).Element(`button.delete-health-btn[data-id="local"]`)
	requireNoErrorWithScreenshot(t, page, err, "find health check delete button")
	wait, handle := page.HandleDialog()
	go el.MustClick()
	_ = wait()
	_ = handle(&proto.PageHandleJavaScriptDialog{Accept: true})

	// Wait for health check to be removed from config list
	page.Timeout(1*time.Second).MustElementR(`#health-config-items`, "No health checks configured")

	// Go back to dashboard and verify health check is gone
	el, err = page.Element(`a.nav-link[href="/"]`)
	requireNoErrorWithScreenshot(t, page, err, "find dashboard link")
	el.MustClick()
	page.Timeout(1 * time.Second).MustElement(`#health-items`)
	assert.Panics(t, func() {
		page.Timeout(1 * time.Second).MustElement(`#health-items .list-group-item[data-id="health_local"]`)
	}, "Health check item 'local' should be gone from dashboard")
}

func TestAddPingViaConfigUIRod(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	s, _ := web.StartTestServer(ctx, t, "")
	s.Start(ctx)

	browser := getBrowser(t)
	defer closeBrowser(browser)
	page, err := browser.Page(proto.TargetCreateTarget{URL: s.ServerUrl})
	require.NoError(t, err, "open page")
	defer closePage(page)

	// Navigate to the config tab
	el, err := page.Element(`a.nav-link[href="/config"]`)
	requireNoErrorWithScreenshot(t, page, err, "find config link")
	el.MustClick()

	// Wait for the unified monitor form to appear
	_, err = page.Timeout(1 * time.Second).Element(`#add-monitor-form`)
	requireNoErrorWithScreenshot(t, page, err, "find monitor form")

	// Switch to ping mode
	el, err = page.Element(`label[for="monitor-type-ping"]`)
	requireNoErrorWithScreenshot(t, page, err, "find ping radio button label")
	el.MustClick()

	// Fill out the Add Ping Monitor form
	el, err = page.Element(`#monitor-name`)
	requireNoErrorWithScreenshot(t, page, err, "find monitor name input")
	err = el.Input("google")
	requireNoErrorWithScreenshot(t, page, err, "input monitor name")
	el, err = page.Element(`#monitor-target`)
	requireNoErrorWithScreenshot(t, page, err, "find monitor target input")
	err = el.Input("8.8.8.8")
	requireNoErrorWithScreenshot(t, page, err, "input monitor target")
	el, err = page.Element(`#monitor-timeout`)
	requireNoErrorWithScreenshot(t, page, err, "find monitor timeout input")
	el.MustSelectAllText().MustInput("100")
	el, err = page.Element(`#monitor-amber-threshold`)
	requireNoErrorWithScreenshot(t, page, err, "find monitor amber threshold input")
	el.MustSelectAllText().MustInput("100")

	// Wait for the icon dropdown to be initialized
	_, err = page.Timeout(500 * time.Millisecond).Element(`#monitor-icon-select`)
	if err != nil {
		// Icon dropdown may not be required for form submission
		t.Logf("Warning: monitor icon dropdown not found, continuing anyway")
	}

	// Submit the form
	el, err = page.Element(`#add-monitor-form button[type="submit"]`)
	requireNoErrorWithScreenshot(t, page, err, "find monitor submit button")
	el.MustClick()

	// Wait for the page to navigate/reload after form submission completes
	// The form submit triggers a POST and then the page redirects back to /config
	time.Sleep(100 * time.Millisecond) // Give the POST time to complete
	err = page.Timeout(2 * time.Second).WaitLoad()
	requireNoErrorWithScreenshot(t, page, err, "wait for page load after adding ping")

	// Wait for the ping monitor to appear in the config list
	require.Eventually(t, func() bool {
		items, err := page.Elements(`#ping-config-items .config-item`)
		if err != nil {
			t.Logf("Error getting ping config items: %v", err)
			return false
		}
		return len(items) > 0
	}, 3*time.Second, 100*time.Millisecond, "wait for ping item in config list")

	items, err := page.Elements(`#ping-config-items .config-item`)
	assert.NoError(t, err, "get ping config items")
	assert.Len(t, items, 1, "exactly one ping config item should be present")
	item := items[0]
	nameSpan, err := item.Element(`.config-item-name`)
	assert.NoError(t, err, "get ping item name span")
	nameText, err := nameSpan.Text()
	assert.NoError(t, err, "get ping item name text")
	addressSpan, err := item.Element(`.config-item-address`)
	assert.NoError(t, err, "get ping item address span")
	addressText, err := addressSpan.Text()
	assert.NoError(t, err, "get ping item address text")
	assert.Equal(t, "google", nameText, "Ping monitor name is shown")
	assert.Equal(t, "(8.8.8.8)", addressText, "Ping monitor address is shown")

	// Also check if "No ping monitors configured" is shown
	_, err = page.Timeout(time.Millisecond*20).ElementR(`#ping-config-items`, "No ping monitors configured")
	assert.Error(t, err, "No ping monitors configured message should be present")

	// Navigate back to dashboard
	el, err = page.Element(`a.nav-link[href="/"]`)
	requireNoErrorWithScreenshot(t, page, err, "find dashboard link")
	el.MustClick()

	// Wait for dashboard ping items to appear
	_, err = page.Timeout(1 * time.Second).Element(`#ping-items`)
	requireNoErrorWithScreenshot(t, page, err, "find ping items")

	// Wait for the ping monitor item to appear in the dashboard
	_, err = page.Timeout(1 * time.Second).Element(`#ping-items .list-group-item[data-id="ping_google"]`)
	requireNoErrorWithScreenshot(t, page, err, "find ping monitor item in dashboard")

	// Assert its presence
	pingItem, err := page.Element(`#ping-items .list-group-item[data-id="ping_google"]`)
	requireNoErrorWithScreenshot(t, page, err, "find ping monitor item in dashboard")
	assert.NotNil(t, pingItem, "Ping monitor item 'google' is present in dashboard")
	pingText := pingItem.MustText()
	assert.Contains(t, pingText, "google", "Ping monitor display name is shown")
	assert.Regexp(t, `\d+(\.\d+)?\s*ms`, pingText, "Ping response time is shown")

	// --- MOBILE PAGE CHECKS ---
	el, err = page.Element(`a.nav-link[href="/mobile"]`)
	requireNoErrorWithScreenshot(t, page, err, "find mobile link")
	el.MustClick()

	page.Timeout(1 * time.Second).MustElement(`#mobile-items-list`)
	pingMobile := page.MustElement(`#mobile-items-list .mobile-list-item[data-id="ping_google"]`)
	pingMobileText := pingMobile.MustText()
	assert.Contains(t, pingMobileText, "google", "Ping monitor display name is shown on mobile")
	assert.Regexp(t, `\d+(\.\d+)?\s*ms`, pingMobileText, "Ping response time is shown on mobile")

	// Go back to config and delete the ping monitor
	el, err = page.Element(`a.nav-link[href="/config"]`)
	requireNoErrorWithScreenshot(t, page, err, "find config link")
	el.MustClick()
	_, err = page.Timeout(1 * time.Second).Element(`#ping-config-items`)
	requireNoErrorWithScreenshot(t, page, err, "find ping config items")
	_, err = page.ElementR(`#ping-config-items .config-item`, "google")
	requireNoErrorWithScreenshot(t, page, err, "find ping item in config list")
	el, err = page.Timeout(1 * time.Second).Element(`button.delete-ping-btn[data-id="google"]`)
	requireNoErrorWithScreenshot(t, page, err, "find ping delete button")
	wait, handle := page.HandleDialog()
	go el.MustClick()
	_ = wait()
	_ = handle(&proto.PageHandleJavaScriptDialog{Accept: true})

	// Wait for ping monitor to be removed from config list
	page.Timeout(1*time.Second).MustElementR(`#ping-config-items`, "No ping monitors configured")

	// Go back to dashboard and verify ping monitor is gone
	el, err = page.Element(`a.nav-link[href="/"]`)
	requireNoErrorWithScreenshot(t, page, err, "find dashboard link")
	el.MustClick()
	page.Timeout(1 * time.Second).MustElement(`#ping-items`)
	assert.Panics(t, func() {
		page.Timeout(1 * time.Second).MustElement(`#ping-items .list-group-item[data-id="ping_google"]`)
	}, "Ping monitor item 'google' should be gone from dashboard")
}

func TestAddDockerCheckViConfigUIRod(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	s, _ := web.StartTestServer(ctx, t, "")
	s.Start(ctx)

	browser := getBrowser(t)
	defer closeBrowser(browser)
	page, err := browser.Page(proto.TargetCreateTarget{URL: s.ServerUrl})
	require.NoError(t, err, "open page")
	defer closePage(page)

	// Navigate to the config tab
	el, err := page.Element(`a.nav-link[href="/config"]`)
	requireNoErrorWithScreenshot(t, page, err, "find config link")
	el.MustClick()

	// Wait for the docker add form
	_, err = page.Timeout(1 * time.Second).Element(`#add-docker-form`)
	requireNoErrorWithScreenshot(t, page, err, "find docker add form")

	// Fill the docker add form
	name := "stack1"
	containers := "web, api"

	el, err = page.Element(`#docker-name`)
	requireNoErrorWithScreenshot(t, page, err, "find docker name input")
	err = el.Input(name)
	requireNoErrorWithScreenshot(t, page, err, "input docker name")

	el, err = page.Element(`#docker-containers`)
	requireNoErrorWithScreenshot(t, page, err, "find docker containers input")
	err = el.Input(containers)
	requireNoErrorWithScreenshot(t, page, err, "input docker containers")

	el, err = page.Element(`#docker-threshold`)
	requireNoErrorWithScreenshot(t, page, err, "find docker threshold input")
	el.MustSelectAllText().MustInput("10")

	// Icon dropdown may be initialized dynamically; not required for submission
	_, err = page.Timeout(300 * time.Millisecond).Element(`#docker-icon-dropdown`)
	assert.NoError(t, err, "docker icon dropdown should be present")

	// Submit the form
	el, err = page.Element(`#add-docker-form button[type="submit"]`)
	requireNoErrorWithScreenshot(t, page, err, "find docker submit button")
	wait2 := page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	el.MustClick()
	// Wait for the page to reload back to /config
	wait2()

	// Verify the docker config item appears
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		items, err := page.Elements(`#docker-config-items .config-item`)
		assert.NoError(t, err, "get docker config items")
		assert.Len(t, items, 1, "exactly one docker config item should be present")
	}, 3*time.Second, 100*time.Millisecond, "wait for docker item in config list")

	items, _ := page.Elements(`#docker-config-items .config-item`)
	item := items[0]
	nameSpan, err := item.Element(`.config-item-name`)
	assert.NoError(t, err, "get docker item name span")
	nameText, err := nameSpan.Text()
	assert.NoError(t, err, "get docker item name text")
	contSpan, err := item.Element(`.config-item-containers`)
	assert.NoError(t, err, "get docker item containers span")
	contText, err := contSpan.Text()
	assert.NoError(t, err, "get docker item containers text")
	assert.Equal(t, name, nameText, "Docker item name is shown")
	assert.Equal(t, "(web, api)", contText, "Docker item containers are shown")

	// Navigate back to dashboard
	el, err = page.Element(`a.nav-link[href="/"]`)
	requireNoErrorWithScreenshot(t, page, err, "find dashboard link")
	el.MustClick()

	// Wait for docker card and item
	_, err = page.Timeout(1 * time.Second).Element(`#docker-items`)
	requireNoErrorWithScreenshot(t, page, err, "find docker items container")
	_, err = page.Timeout(1 * time.Second).Element(`#docker-items .list-group-item[data-id="docker_stack1"]`)
	requireNoErrorWithScreenshot(t, page, err, "find docker item in dashboard")

	// Assert its presence and content
	dockerItem, err := page.Element(`#docker-items .list-group-item[data-id="docker_stack1"]`)
	requireNoErrorWithScreenshot(t, page, err, "get docker item element")
	assert.NotNil(t, dockerItem, "Docker item 'stack1' is present in dashboard")
	dockerText := dockerItem.MustText()
	assert.Contains(t, dockerText, "stack1 (2 containers)", "Docker display name is shown")
	assert.Regexp(t, `Max\s+restarts:\s*\d+`, dockerText, "Docker value is shown")
	assert.Contains(t, dockerText, "web", "Docker details include container 'web'")
	assert.Contains(t, dockerText, "api", "Docker details include container 'api'")

	// --- MOBILE PAGE CHECKS ---
	el, err = page.Element(`a.nav-link[href="/mobile"]`)
	requireNoErrorWithScreenshot(t, page, err, "find mobile link")
	el.MustClick()

	page.Timeout(1 * time.Second).MustElement(`#mobile-items-list`)
	dockerMobile := page.MustElement(`#mobile-items-list .mobile-list-item[data-id="docker_stack1"]`)
	dockerMobileText := dockerMobile.MustText()
	assert.Contains(t, dockerMobileText, "stack1 (2 containers)", "Docker display name is shown on mobile")
	assert.Regexp(t, `Max\s+restarts:\s*\d+`, dockerMobileText, "Docker value is shown on mobile")

	// Go back to config and delete the docker monitor
	el, err = page.Element(`a.nav-link[href="/config"]`)
	requireNoErrorWithScreenshot(t, page, err, "find config link (delete)")
	el.MustClick()

	_, err = page.Timeout(1 * time.Second).Element(`#docker-config-items`)
	requireNoErrorWithScreenshot(t, page, err, "find docker config items")
	delBtn, err := page.Timeout(1 * time.Second).Element(`button.delete-docker-btn[data-id="stack1"]`)
	requireNoErrorWithScreenshot(t, page, err, "find docker delete button")
	wait, handle := page.HandleDialog()
	go delBtn.MustClick()
	_ = wait()
	_ = handle(&proto.PageHandleJavaScriptDialog{Accept: true})

	// Wait for docker monitor to be removed from config list
	page.Timeout(1*time.Second).MustElementR(`#docker-config-items`, "No Docker monitors configured")

	// Go back to dashboard and verify docker monitor is gone
	el, err = page.Element(`a.nav-link[href="/"]`)
	requireNoErrorWithScreenshot(t, page, err, "find dashboard link (final)")
	el.MustClick()
	page.Timeout(1 * time.Second).MustElement(`#docker-items`)
	assert.Panics(t, func() {
		page.Timeout(1 * time.Second).MustElement(`#docker-items .list-group-item[data-id="docker_stack1"]`)
	}, "Docker item 'stack1' should be gone from dashboard")
}

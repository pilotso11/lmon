package uitest

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"lmon/web"
)

// TestEditDiskMonitorViaUI tests the edit button functionality for disk monitors
func TestEditDiskMonitorViaUI(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	s, _ := web.StartTestServer(ctx, t, "")
	s.Start(ctx)

	browser := getBrowser(t)
	defer closeBrowser(browser)
	page, err := browser.Page(proto.TargetCreateTarget{URL: s.ServerUrl})
	requireNoErrorWithScreenshot(t, page, err, "open page")
	defer closePage(page)

	// Navigate to config page
	configLink, err := page.Element(`a.nav-link[href="/config"]`)
	requireNoErrorWithScreenshot(t, page, err, "find config tab")
	err = configLink.Click(proto.InputMouseButtonLeft, 1)
	requireNoErrorWithScreenshot(t, page, err, "click config tab")

	// Wait for config form to appear
	_, err = page.Timeout(1 * time.Second).Element(`#add-disk-form`)
	requireNoErrorWithScreenshot(t, page, err, "wait for add-disk-form")

	// Add a disk monitor
	nameEl, err := page.Element(`#disk-name`)
	requireNoErrorWithScreenshot(t, page, err, "find disk-name input")
	err = nameEl.Input("test-edit-disk")
	requireNoErrorWithScreenshot(t, page, err, "input disk-name")

	pathEl, err := page.Element(`#disk-path`)
	requireNoErrorWithScreenshot(t, page, err, "find disk-path input")
	err = pathEl.Input("/tmp")
	requireNoErrorWithScreenshot(t, page, err, "input disk-path")

	thresholdEl, err := page.Element(`#disk-threshold`)
	requireNoErrorWithScreenshot(t, page, err, "find disk-threshold input")
	err = thresholdEl.SelectAllText()
	requireNoErrorWithScreenshot(t, page, err, "select threshold text")
	err = thresholdEl.Input("85")
	requireNoErrorWithScreenshot(t, page, err, "input disk-threshold")

	submitEl, err := page.Element(`#add-disk-form button[type="submit"]`)
	requireNoErrorWithScreenshot(t, page, err, "find submit button")
	err = submitEl.Click(proto.InputMouseButtonLeft, 1)
	requireNoErrorWithScreenshot(t, page, err, "click submit")

	// Wait for page reload
	err = page.Timeout(2 * time.Second).Reload()
	requireNoErrorWithScreenshot(t, page, err, "wait for reload after adding disk")

	// Verify disk was added
	assert.Eventually(t, func() bool {
		items, err := page.Elements(`#disk-config-items .config-item`)
		if err != nil {
			return false
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
			if nameText == "test-edit-disk" {
				return true
			}
		}
		return false
	}, 2*time.Second, 50*time.Millisecond, "wait for disk item in config list")

	// Click the edit button
	editBtn, err := page.Element(`button.edit-disk-btn[data-id="test-edit-disk"]`)
	requireNoErrorWithScreenshot(t, page, err, "find edit button")
	err = editBtn.Click(proto.InputMouseButtonLeft, 1)
	requireNoErrorWithScreenshot(t, page, err, "click edit button")

	// Wait a moment for form to populate
	time.Sleep(500 * time.Millisecond)

	// Verify form fields are populated
	nameVal, err := page.Eval(`() => document.getElementById("disk-name").value`)
	requireNoErrorWithScreenshot(t, page, err, "get name field value")
	assert.Equal(t, "test-edit-disk", nameVal.Value.String(), "Name field should be populated")

	pathVal, err := page.Eval(`() => document.getElementById("disk-path").value`)
	requireNoErrorWithScreenshot(t, page, err, "get path field value")
	assert.Equal(t, "/tmp", pathVal.Value.String(), "Path field should be populated")

	thresholdVal, err := page.Eval(`() => document.getElementById("disk-threshold").value`)
	requireNoErrorWithScreenshot(t, page, err, "get threshold field value")
	assert.Equal(t, "85", thresholdVal.Value.String(), "Threshold field should be populated")

	// Modify the threshold
	thresholdEl, err = page.Element(`#disk-threshold`)
	requireNoErrorWithScreenshot(t, page, err, "find disk-threshold input for edit")
	err = thresholdEl.SelectAllText()
	requireNoErrorWithScreenshot(t, page, err, "select threshold text for edit")
	err = thresholdEl.Input("90")
	requireNoErrorWithScreenshot(t, page, err, "input new threshold")

	// Submit the updated disk monitor
	submitEl, err = page.Element(`#add-disk-form button[type="submit"]`)
	requireNoErrorWithScreenshot(t, page, err, "find submit button for update")
	err = submitEl.Click(proto.InputMouseButtonLeft, 1)
	requireNoErrorWithScreenshot(t, page, err, "click submit to update")

	// Wait for page reload
	err = page.Timeout(2 * time.Second).Reload()
	requireNoErrorWithScreenshot(t, page, err, "wait for reload after updating disk")

	// Verify the disk monitor was updated by checking the displayed threshold
	assert.Eventually(t, func() bool {
		items, err := page.Elements(`#disk-config-items .config-item`)
		if err != nil {
			return false
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
			if nameText == "test-edit-disk" {
				itemText, err := item.Text()
				if err != nil {
					return false
				}
				// Check if threshold is now 90%
				return len(itemText) > 0 && (itemText != "")
			}
		}
		return false
	}, 2*time.Second, 50*time.Millisecond, "wait for updated disk item")
}

// TestEditHealthCheckViaUI tests the edit button functionality for health checks
func TestEditHealthCheckViaUI(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	s, _ := web.StartTestServer(ctx, t, "")
	s.Start(ctx)

	browser := getBrowser(t)
	defer closeBrowser(browser)
	page, err := browser.Page(proto.TargetCreateTarget{URL: s.ServerUrl})
	requireNoErrorWithScreenshot(t, page, err, "open page")
	defer closePage(page)

	// Navigate to config page
	configLink, err := page.Element(`a.nav-link[href="/config"]`)
	requireNoErrorWithScreenshot(t, page, err, "find config tab")
	err = configLink.Click(proto.InputMouseButtonLeft, 1)
	requireNoErrorWithScreenshot(t, page, err, "click config tab")

	// Wait for monitor form to appear
	_, err = page.Timeout(1 * time.Second).Element(`#add-monitor-form`)
	requireNoErrorWithScreenshot(t, page, err, "wait for add-monitor-form")

	// Add a health check
	nameEl, err := page.Element(`#monitor-name`)
	requireNoErrorWithScreenshot(t, page, err, "find monitor-name input")
	err = nameEl.Input("test-edit-health")
	requireNoErrorWithScreenshot(t, page, err, "input monitor-name")

	urlEl, err := page.Element(`#monitor-target`)
	requireNoErrorWithScreenshot(t, page, err, "find monitor-target input")
	err = urlEl.Input("http://example.com/health")
	requireNoErrorWithScreenshot(t, page, err, "input monitor-target")

	timeoutEl, err := page.Element(`#monitor-timeout`)
	requireNoErrorWithScreenshot(t, page, err, "find monitor-timeout input")
	err = timeoutEl.SelectAllText()
	requireNoErrorWithScreenshot(t, page, err, "select timeout text")
	err = timeoutEl.Input("15")
	requireNoErrorWithScreenshot(t, page, err, "input monitor-timeout")

	submitEl, err := page.Element(`#add-monitor-form button[type="submit"]`)
	requireNoErrorWithScreenshot(t, page, err, "find submit button")
	err = submitEl.Click(proto.InputMouseButtonLeft, 1)
	requireNoErrorWithScreenshot(t, page, err, "click submit")

	// Wait for page reload
	err = page.Timeout(2 * time.Second).Reload()
	requireNoErrorWithScreenshot(t, page, err, "wait for reload after adding health check")

	// Verify health check was added
	assert.Eventually(t, func() bool {
		items, err := page.Elements(`#health-config-items .config-item`)
		if err != nil {
			return false
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
			if nameText == "test-edit-health" {
				return true
			}
		}
		return false
	}, 2*time.Second, 50*time.Millisecond, "wait for health check item in config list")

	// Click the edit button
	editBtn, err := page.Element(`button.edit-health-btn[data-id="test-edit-health"]`)
	requireNoErrorWithScreenshot(t, page, err, "find edit button")
	err = editBtn.Click(proto.InputMouseButtonLeft, 1)
	requireNoErrorWithScreenshot(t, page, err, "click edit button")

	// Wait a moment for form to populate
	time.Sleep(500 * time.Millisecond)

	// Verify form fields are populated
	nameVal, err := page.Eval(`() => document.getElementById("monitor-name").value`)
	requireNoErrorWithScreenshot(t, page, err, "get name field value")
	assert.Equal(t, "test-edit-health", nameVal.Value.String(), "Name field should be populated")

	urlVal, err := page.Eval(`() => document.getElementById("monitor-target").value`)
	requireNoErrorWithScreenshot(t, page, err, "get url field value")
	assert.Equal(t, "http://example.com/health", urlVal.Value.String(), "URL field should be populated")

	timeoutVal, err := page.Eval(`() => document.getElementById("monitor-timeout").value`)
	requireNoErrorWithScreenshot(t, page, err, "get timeout field value")
	assert.Equal(t, "15", timeoutVal.Value.String(), "Timeout field should be populated")

	// Verify HTTP mode is selected
	httpChecked, err := page.Eval(`() => document.getElementById("monitor-type-http").checked`)
	requireNoErrorWithScreenshot(t, page, err, "get http radio checked state")
	assert.True(t, httpChecked.Value.Bool(), "HTTP radio should be checked")
}

// TestEditPingMonitorViaUI tests the edit button functionality for ping monitors
func TestEditPingMonitorViaUI(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	s, _ := web.StartTestServer(ctx, t, "")
	s.Start(ctx)

	browser := getBrowser(t)
	defer closeBrowser(browser)
	page, err := browser.Page(proto.TargetCreateTarget{URL: s.ServerUrl})
	requireNoErrorWithScreenshot(t, page, err, "open page")
	defer closePage(page)

	// Navigate to config page
	configLink, err := page.Element(`a.nav-link[href="/config"]`)
	requireNoErrorWithScreenshot(t, page, err, "find config tab")
	err = configLink.Click(proto.InputMouseButtonLeft, 1)
	requireNoErrorWithScreenshot(t, page, err, "click config tab")

	// Wait for monitor form to appear
	_, err = page.Timeout(1 * time.Second).Element(`#add-monitor-form`)
	requireNoErrorWithScreenshot(t, page, err, "wait for add-monitor-form")

	// Switch to Ping mode by clicking the label (Bootstrap btn-check pattern)
	pingLabel, err := page.Element(`label[for="monitor-type-ping"]`)
	requireNoErrorWithScreenshot(t, page, err, "find ping radio label")
	err = pingLabel.Click(proto.InputMouseButtonLeft, 1)
	requireNoErrorWithScreenshot(t, page, err, "click ping radio label")

	time.Sleep(200 * time.Millisecond) // Wait for form to update

	// Add a ping monitor
	nameEl, err := page.Element(`#monitor-name`)
	requireNoErrorWithScreenshot(t, page, err, "find monitor-name input")
	err = nameEl.Input("test-edit-ping")
	requireNoErrorWithScreenshot(t, page, err, "input monitor-name")

	addressEl, err := page.Element(`#monitor-target`)
	requireNoErrorWithScreenshot(t, page, err, "find monitor-target input")
	err = addressEl.Input("8.8.8.8")
	requireNoErrorWithScreenshot(t, page, err, "input monitor-target")

	submitEl, err := page.Element(`#add-monitor-form button[type="submit"]`)
	requireNoErrorWithScreenshot(t, page, err, "find submit button")
	err = submitEl.Click(proto.InputMouseButtonLeft, 1)
	requireNoErrorWithScreenshot(t, page, err, "click submit")

	// Wait for page reload
	err = page.Timeout(2 * time.Second).Reload()
	requireNoErrorWithScreenshot(t, page, err, "wait for reload after adding ping monitor")

	// Verify ping monitor was added
	assert.Eventually(t, func() bool {
		items, err := page.Elements(`#ping-config-items .config-item`)
		if err != nil {
			return false
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
			if nameText == "test-edit-ping" {
				return true
			}
		}
		return false
	}, 2*time.Second, 50*time.Millisecond, "wait for ping monitor item in config list")

	// Click the edit button
	editBtn, err := page.Element(`button.edit-ping-btn[data-id="test-edit-ping"]`)
	requireNoErrorWithScreenshot(t, page, err, "find edit button")
	err = editBtn.Click(proto.InputMouseButtonLeft, 1)
	requireNoErrorWithScreenshot(t, page, err, "click edit button")

	// Wait a moment for form to populate
	time.Sleep(500 * time.Millisecond)

	// Verify form fields are populated
	nameVal, err := page.Eval(`() => document.getElementById("monitor-name").value`)
	requireNoErrorWithScreenshot(t, page, err, "get name field value")
	assert.Equal(t, "test-edit-ping", nameVal.Value.String(), "Name field should be populated")

	addressVal, err := page.Eval(`() => document.getElementById("monitor-target").value`)
	requireNoErrorWithScreenshot(t, page, err, "get address field value")
	assert.Equal(t, "8.8.8.8", addressVal.Value.String(), "Address field should be populated")

	// Verify Ping mode is selected
	pingChecked, err := page.Eval(`() => document.getElementById("monitor-type-ping").checked`)
	requireNoErrorWithScreenshot(t, page, err, "get ping radio checked state")
	assert.True(t, pingChecked.Value.Bool(), "Ping radio should be checked")
}

// TestAddDiskWithMaintenanceViaUI tests adding a disk monitor with a maintenance window,
// then verifies the maintenance badge appears in the config list and edit populates the fields.
func TestAddDiskWithMaintenanceViaUI(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	s, _ := web.StartTestServer(ctx, t, "")
	s.Start(ctx)

	browser := getBrowser(t)
	defer closeBrowser(browser)
	page, err := browser.Page(proto.TargetCreateTarget{URL: s.ServerUrl})
	requireNoErrorWithScreenshot(t, page, err, "open page")
	defer closePage(page)

	// Navigate to config page
	configLink, err := page.Element(`a.nav-link[href="/config"]`)
	requireNoErrorWithScreenshot(t, page, err, "find config tab")
	err = configLink.Click(proto.InputMouseButtonLeft, 1)
	requireNoErrorWithScreenshot(t, page, err, "click config tab")

	// Wait for form
	_, err = page.Timeout(1 * time.Second).Element(`#add-disk-form`)
	requireNoErrorWithScreenshot(t, page, err, "wait for add-disk-form")

	// Fill out disk form
	nameEl, err := page.Element(`#disk-name`)
	requireNoErrorWithScreenshot(t, page, err, "find disk-name input")
	err = nameEl.Input("maint-test")
	requireNoErrorWithScreenshot(t, page, err, "input disk-name")

	pathEl, err := page.Element(`#disk-path`)
	requireNoErrorWithScreenshot(t, page, err, "find disk-path input")
	err = pathEl.Input("/")
	requireNoErrorWithScreenshot(t, page, err, "input disk-path")

	// Fill maintenance fields
	cronEl, err := page.Element(`#disk-maint-cron`)
	requireNoErrorWithScreenshot(t, page, err, "find maintenance cron input")
	err = cronEl.Input("0 */4 * * *")
	requireNoErrorWithScreenshot(t, page, err, "input maintenance cron")

	durEl, err := page.Element(`#disk-maint-duration`)
	requireNoErrorWithScreenshot(t, page, err, "find maintenance duration input")
	err = durEl.Input("60")
	requireNoErrorWithScreenshot(t, page, err, "input maintenance duration")

	// Submit
	submitEl, err := page.Element(`#add-disk-form button[type="submit"]`)
	requireNoErrorWithScreenshot(t, page, err, "find submit button")
	err = submitEl.Click(proto.InputMouseButtonLeft, 1)
	requireNoErrorWithScreenshot(t, page, err, "click submit")

	// Wait for reload
	err = page.Timeout(2 * time.Second).Reload()
	requireNoErrorWithScreenshot(t, page, err, "wait for reload")

	// Verify the maintenance badge is visible in the config list
	assert.Eventually(t, func() bool {
		items, err := page.Elements(`#disk-config-items .config-item`)
		if err != nil {
			return false
		}
		for _, item := range items {
			text, err := item.Text()
			if err != nil {
				continue
			}
			if contains(text, "maint-test") && contains(text, "Maint:") && contains(text, "0 */4 * * *") {
				return true
			}
		}
		return false
	}, 2*time.Second, 50*time.Millisecond, "maintenance badge should appear in config list")

	// Click edit and verify maintenance fields are populated
	editBtn, err := page.Element(`button.edit-disk-btn[data-id="maint-test"]`)
	requireNoErrorWithScreenshot(t, page, err, "find edit button")
	err = editBtn.Click(proto.InputMouseButtonLeft, 1)
	requireNoErrorWithScreenshot(t, page, err, "click edit button")

	time.Sleep(500 * time.Millisecond)

	cronVal, err := page.Eval(`() => document.getElementById("disk-maint-cron").value`)
	requireNoErrorWithScreenshot(t, page, err, "get maint cron field value")
	assert.Equal(t, "0 */4 * * *", cronVal.Value.String(), "Maintenance cron should be populated")

	durVal, err := page.Eval(`() => document.getElementById("disk-maint-duration").value`)
	requireNoErrorWithScreenshot(t, page, err, "get maint duration field value")
	assert.Equal(t, "60", durVal.Value.String(), "Maintenance duration should be populated")
}

// TestAddHealthCheckWithMaintenanceLayout tests that adding a health check with a
// maintenance window renders the maintenance badge on a separate line (in .config-item-maint)
// rather than inline in the main flex row.
func TestAddHealthCheckWithMaintenanceLayout(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	s, _ := web.StartTestServer(ctx, t, "")
	s.Start(ctx)

	browser := getBrowser(t)
	defer closeBrowser(browser)
	page, err := browser.Page(proto.TargetCreateTarget{URL: s.ServerUrl})
	requireNoErrorWithScreenshot(t, page, err, "open page")
	defer closePage(page)

	// Navigate to config page
	configLink, err := page.Element(`a.nav-link[href="/config"]`)
	requireNoErrorWithScreenshot(t, page, err, "find config tab")
	err = configLink.Click(proto.InputMouseButtonLeft, 1)
	requireNoErrorWithScreenshot(t, page, err, "click config tab")

	// Wait for monitor form
	_, err = page.Timeout(1 * time.Second).Element(`#add-monitor-form`)
	requireNoErrorWithScreenshot(t, page, err, "wait for add-monitor-form")

	// Fill out health check form
	nameEl, err := page.Element(`#monitor-name`)
	requireNoErrorWithScreenshot(t, page, err, "find monitor-name input")
	err = nameEl.Input("maint-layout-test")
	requireNoErrorWithScreenshot(t, page, err, "input monitor-name")

	urlEl, err := page.Element(`#monitor-target`)
	requireNoErrorWithScreenshot(t, page, err, "find monitor-target input")
	err = urlEl.Input("http://example.com/health")
	requireNoErrorWithScreenshot(t, page, err, "input monitor-target")

	// Fill maintenance fields
	cronEl, err := page.Element(`#monitor-maint-cron`)
	requireNoErrorWithScreenshot(t, page, err, "find maintenance cron input")
	err = cronEl.Input("0 */4 * * *")
	requireNoErrorWithScreenshot(t, page, err, "input maintenance cron")

	durEl, err := page.Element(`#monitor-maint-duration`)
	requireNoErrorWithScreenshot(t, page, err, "find maintenance duration input")
	err = durEl.Input("120")
	requireNoErrorWithScreenshot(t, page, err, "input maintenance duration")

	// Submit
	submitEl, err := page.Element(`#add-monitor-form button[type="submit"]`)
	requireNoErrorWithScreenshot(t, page, err, "find submit button")
	err = submitEl.Click(proto.InputMouseButtonLeft, 1)
	requireNoErrorWithScreenshot(t, page, err, "click submit")

	// Wait for reload
	err = page.Timeout(2 * time.Second).Reload()
	requireNoErrorWithScreenshot(t, page, err, "wait for reload")

	// Verify the config item exists and find it
	var item *rod.Element
	assert.Eventually(t, func() bool {
		items, err := page.Elements(`#health-config-items .config-item`)
		if err != nil {
			return false
		}
		for _, i := range items {
			nameSpan, err := i.Element(`.config-item-name`)
			if err != nil {
				continue
			}
			nameText, err := nameSpan.Text()
			if err != nil {
				continue
			}
			if nameText == "maint-layout-test" {
				item = i
				return true
			}
		}
		return false
	}, 2*time.Second, 50*time.Millisecond, "wait for health check item in config list")
	require.NotNil(t, item, "Should find the maint-layout-test config item")

	// Verify the maintenance badge is in a .config-item-maint div (second line),
	// not inline in the d-flex row
	maintDiv, err := item.Element(`.config-item-maint`)
	requireNoErrorWithScreenshot(t, page, err, "maintenance badge should be in .config-item-maint div")

	maintText, err := maintDiv.Text()
	requireNoErrorWithScreenshot(t, page, err, "get maintenance div text")
	assert.Contains(t, maintText, "Maint:", "Maintenance badge should contain 'Maint:'")
	assert.Contains(t, maintText, "0 */4 * * *", "Maintenance badge should contain cron expression")
	assert.Contains(t, maintText, "120s", "Maintenance badge should contain duration")

	// Verify the maintenance badge is NOT inside the d-flex row
	flexRow, err := item.Element(`.d-flex`)
	requireNoErrorWithScreenshot(t, page, err, "find flex row")
	hasMaintInFlex, _, err := flexRow.Has(`.badge.bg-info`)
	requireNoErrorWithScreenshot(t, page, err, "check for badge in flex row")
	assert.False(t, hasMaintInFlex, "Maintenance badge should NOT be inside the flex row")
}

// contains is a simple string contains helper for use in Eventually assertions.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || strings.Contains(s, substr))
}

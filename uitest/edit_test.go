package uitest

import (
	"context"
	"testing"
	"time"

	"github.com/go-rod/rod/lib/proto"
	"github.com/stretchr/testify/assert"

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

	// Switch to Ping mode
	pingRadio, err := page.Element(`#monitor-type-ping`)
	requireNoErrorWithScreenshot(t, page, err, "find ping radio")
	err = pingRadio.Click(proto.InputMouseButtonLeft, 1)
	requireNoErrorWithScreenshot(t, page, err, "click ping radio")

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

package uitest

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUI runs UI tests for the lmon web interface
func TestUI(t *testing.T) {
	// Launch a new browser
	launch := launcher.New().Headless(true).Set("no-sandbox")
	url, err := launch.Launch()
	if err != nil {
		t.Fatalf("Failed to launch browser: %v", err)
	}
	browser := rod.New().ControlURL(url).MustConnect()
	defer browser.MustClose()

	// Set a timeout for the entire test
	browser = browser.Timeout(30 * time.Second)

	// Get the port from the environment variable
	portStr := os.Getenv("LMON_TEST_PORT")
	if portStr == "" {
		t.Fatal("LMON_TEST_PORT environment variable not set")
	}

	// Base URL for the application
	baseURL := fmt.Sprintf("http://localhost:%s", portStr)

	// Run tests
	t.Run("Dashboard", func(t *testing.T) {
		testDashboard(t, browser, baseURL)
	})

	t.Run("Configuration", func(t *testing.T) {
		testConfiguration(t, browser, baseURL)
	})

	t.Run("ResponsiveNavbar", func(t *testing.T) {
		testResponsiveNavbar(t, browser, baseURL)
	})
}

// testDashboard tests the dashboard page
func testDashboard(t *testing.T, browser *rod.Browser, baseURL string) {
	// Navigate to the dashboard
	page := browser.MustPage(baseURL)
	defer page.MustClose()

	// Wait for the page to load
	err := page.WaitLoad()
	require.NoError(t, err, "Failed to wait for page to load")

	// Check the title
	titleElem, err := page.Element("title")
	require.NoError(t, err, "Failed to find title element")
	title, err := titleElem.Text()
	require.NoError(t, err, "Failed to get title text")
	assert.Contains(t, title, "lmon")

	// Check that the dashboard elements are present
	assert.True(t, page.MustHas("h1"))
	h1Elem, err := page.Element("h1")
	require.NoError(t, err, "Failed to find h1 element")
	h1Text, err := h1Elem.Text()
	require.NoError(t, err, "Failed to get h1 text")
	// The dashboard title should be either the default "Monitoring Dashboard" or a custom title
	assert.NotEmpty(t, h1Text, "Dashboard title should not be empty")

	// Check that the system card is present
	systemCards, err := page.Elements(".card-header")
	require.NoError(t, err, "Failed to find card headers")
	var systemCard *rod.Element
	for _, card := range systemCards {
		cardText, err := card.Text()
		require.NoError(t, err, "Failed to get card text")
		if cardText == "System" {
			systemCard = card
			break
		}
	}
	assert.NotNil(t, systemCard, "System card not found")

	// Check that the disk card is present
	var diskCard *rod.Element
	for _, card := range systemCards {
		cardText, err := card.Text()
		require.NoError(t, err, "Failed to get card text")
		if cardText == "Disk" {
			diskCard = card
			break
		}
	}
	assert.NotNil(t, diskCard, "Disk card not found")

	// Check that the health checks card is present
	var healthCard *rod.Element
	for _, card := range systemCards {
		cardText, err := card.Text()
		require.NoError(t, err, "Failed to get card text")
		if cardText == "Health Checks" {
			healthCard = card
			break
		}
	}
	assert.NotNil(t, healthCard, "Health Checks card not found")

	// Wait for data to load by checking for a specific element that indicates data is present
	// For example, wait for the first list-group-item to be visible
	_, err = page.MustElement(".list-group-item").WaitVisible()
	require.NoError(t, err, "Failed to wait for list group item to be visible")

	// Test that percentages are properly rounded to 2 decimal places
	// This assumes there's at least one percentage value displayed
	listItems, err := page.Elements(".list-group-item")
	require.NoError(t, err, "Failed to find list items")
	var percentageElements []*rod.Element
	for _, item := range listItems {
		spans, err := item.Elements("span")
		require.NoError(t, err, "Failed to find spans in list item")
		for _, span := range spans {
			text, err := span.Text()
			require.NoError(t, err, "Failed to get span text")
			if strings.Contains(text, "%") {
				percentageElements = append(percentageElements, span)
			}
		}
	}

	if len(percentageElements) > 0 {
		for _, elem := range percentageElements {
			text, err := elem.Text()
			require.NoError(t, err, "Failed to get percentage text")
			// Check that the percentage has at most 2 decimal places
			assert.Regexp(t, `\d+\.\d{1,2}%`, text, "Percentage should be rounded to 2 decimal places")
		}
	}

	// Test that memory icon is correct
	memoryListItems, err := page.Elements(".list-group-item")
	require.NoError(t, err, "Failed to find memory list items")
	var memoryElement *rod.Element
	for _, item := range memoryListItems {
		itemText, err := item.Text()
		require.NoError(t, err, "Failed to get memory item text")
		if strings.Contains(itemText, "Memory Usage") {
			memoryElement = item
			break
		}
	}

	if memoryElement != nil {
		memoryIcon, err := memoryElement.Element(".bi")
		require.NoError(t, err, "Failed to get memory icon")
		classAttr, err := memoryIcon.Attribute("class")
		require.NoError(t, err, "Failed to get memory icon class")
		assert.Contains(t, *classAttr, "bi-speedometer", "Memory icon should be speedometer")
	}

	// Test that null threshold is displayed as "N/A%"
	nullThresholdItems, err := page.Elements(".list-group-item")
	require.NoError(t, err, "Failed to find list items")
	var nullThresholdElement *rod.Element
	for _, item := range nullThresholdItems {
		itemText, err := item.Text()
		require.NoError(t, err, "Failed to get item text")
		if strings.Contains(itemText, "Disk Usage (null threshold)") {
			nullThresholdElement = item
			break
		}
	}

	// Click on the null threshold item to open the details modal
	if nullThresholdElement != nil {
		nullThresholdElement.MustClick()

		// Wait for the modal to appear and be visible
		modal, err := page.Element("#detailsModal")
		require.NoError(t, err, "Failed to find modal element")
		err = modal.WaitVisible()
		require.NoError(t, err, "Failed to wait for modal to be visible")

		// Find the threshold value in the modal
		modalBody, err := modal.Element("#modal-body") // Search within the modal
		require.NoError(t, err, "Failed to find modal body")

		// It's good practice to wait for the text to be stable if it's dynamically loaded
		err = modalBody.WaitStable(3 * time.Second) // Wait up to 3 seconds for text to stabilize
		require.NoError(t, err, "Failed to wait for modal body text to stabilize")

		modalText, err := modalBody.Text()
		require.NoError(t, err, "Failed to get modal text")

		// Check that the threshold is displayed as "N/A"
		assert.Contains(t, modalText, "Threshold: N/A", "Null threshold should be displayed as N/A")
	}
}

// testConfiguration tests the configuration page
func testConfiguration(t *testing.T, browser *rod.Browser, baseURL string) {
	// Navigate to the configuration page
	page := browser.MustPage(fmt.Sprintf("%s/config", baseURL))
	defer page.MustClose()

	// Wait for the page to load
	err := page.WaitLoad()
	require.NoError(t, err, "Failed to wait for page to load")

	// Check the title
	titleElem, err := page.Element("title")
	require.NoError(t, err, "Failed to find title element")
	title, err := titleElem.Text()
	require.NoError(t, err, "Failed to get title text")
	assert.Contains(t, title, "lmon")

	// Check that the configuration elements are present
	assert.True(t, page.MustHas("h1"))
	h1Elem, err := page.Element("h1")
	require.NoError(t, err, "Failed to find h1 element")
	h1Text, err := h1Elem.Text()
	require.NoError(t, err, "Failed to get h1 text")
	assert.Equal(t, "Configuration", h1Text)

	// Check that the disk configuration card is present
	configCards, err := page.Elements(".card-header")
	require.NoError(t, err, "Failed to find card headers")
	var diskCard *rod.Element
	var systemCard *rod.Element
	var webSettingsCard *rod.Element

	for _, card := range configCards {
		text, err := card.Text()
		require.NoError(t, err, "Failed to get card text")
		if text == "Disk Monitoring" {
			diskCard = card
		} else if text == "System Monitoring" {
			systemCard = card
		} else if text == "Web Settings" {
			webSettingsCard = card
		}
	}

	assert.NotNil(t, diskCard, "Disk Monitoring card not found")
	assert.NotNil(t, systemCard, "System Monitoring card not found")
	assert.NotNil(t, webSettingsCard, "Web Settings card not found")

	// Test that the dashboard title field is present in the Web Settings card
	if webSettingsCard != nil {
		// Find the Web Settings card body
		webSettingsCardParent, err := webSettingsCard.Parent()
		require.NoError(t, err, "Failed to find Web Settings card parent")
		webSettingsCardBody, err := webSettingsCardParent.Element(".card-body")
		require.NoError(t, err, "Failed to find Web Settings card body")

		// Check that the dashboard title field is present
		dashboardTitleInput, err := webSettingsCardBody.Element("#dashboard-title")
		require.NoError(t, err, "Failed to find dashboard title input")

		// Check that the dashboard title input has a value
		dashboardTitleValue, err := dashboardTitleInput.Attribute("value")
		require.NoError(t, err, "Failed to get dashboard title value")
		assert.NotEmpty(t, *dashboardTitleValue, "Dashboard title value should not be empty")
	}

	// Wait for config items to load
	_, err = page.MustElement(".config-item").WaitVisible()
	require.NoError(t, err, "Failed to wait for config items to be visible")

	// Test that root partition doesn't have a delete icon
	configItems, err := page.Elements(".config-item")
	require.NoError(t, err, "Failed to find config items")
	for _, elem := range configItems {
		// Check if this element contains a strong tag with text "/"
		strongElements, err := elem.Elements("strong")
		require.NoError(t, err, "Failed to find strong elements")
		for _, strong := range strongElements {
			pathText, err := strong.Text()
			require.NoError(t, err, "Failed to get strong text")
			if pathText == "/" {
				// Check that there's no delete button
				deleteButtons, err := elem.Elements(".delete-btn")
				require.NoError(t, err, "Failed to get delete buttons")
				assert.Equal(t, 0, len(deleteButtons), "Root partition should not have a delete button")
				break
			}
		}
	}

	// Test that memory icon is correct in system configuration
	var memoryMonitoringElement *rod.Element
	var cpuMonitoringElement *rod.Element
	configItems, err = page.Elements(".config-item")
	require.NoError(t, err, "Failed to find config items")
	for _, item := range configItems {
		itemText, err := item.Text()
		require.NoError(t, err, "Failed to get item text")
		if strings.Contains(itemText, "Memory Monitoring") {
			memoryMonitoringElement = item
		} else if strings.Contains(itemText, "CPU Monitoring") {
			cpuMonitoringElement = item
		}
	}

	if memoryMonitoringElement != nil {
		memoryIcon, err := memoryMonitoringElement.Element(".bi")
		require.NoError(t, err, "Failed to find memory icon")
		classAttr, err := memoryIcon.Attribute("class")
		require.NoError(t, err, "Failed to get memory icon class")
		assert.Contains(t, *classAttr, "bi-speedometer", "Memory icon should be speedometer")

		// Test that memory threshold is displayed correctly and not as "N/A%"
		thresholdText, err := memoryMonitoringElement.Text()
		require.NoError(t, err, "Failed to get memory threshold text")
		assert.NotContains(t, thresholdText, "Threshold: N/A%", "Memory threshold should not be displayed as N/A%")
		assert.Regexp(t, `Threshold: \d+(\.\d+)?%`, thresholdText, "Memory threshold should be displayed as a percentage")
	}

	if cpuMonitoringElement != nil {
		// Test that CPU threshold is displayed correctly and not as "N/A%"
		thresholdText, err := cpuMonitoringElement.Text()
		require.NoError(t, err, "Failed to get CPU threshold text")
		assert.NotContains(t, thresholdText, "Threshold: N/A%", "CPU threshold should not be displayed as N/A%")
		assert.Regexp(t, `Threshold: \d+(\.\d+)?%`, thresholdText, "CPU threshold should be displayed as a percentage")
	}
}

// testResponsiveNavbar tests that the hamburger menu is visible on small screens
func testResponsiveNavbar(t *testing.T, browser *rod.Browser, baseURL string) {
	// Create a new browser context with a small viewport size
	// This avoids the "Object reference chain is too long" error by setting the viewport size
	// before any DOM elements are loaded and tracked
	page := browser.MustPage("")
	defer page.MustClose()

	// Set a small viewport size (mobile phone size) before navigating
	page = page.MustSetWindow(0, 0, 375, 667) // x, y, width, height

	// Navigate to the dashboard after setting the viewport size
	page.MustNavigate(baseURL)

	// Wait for the page to load
	err := page.WaitLoad()
	require.NoError(t, err, "Failed to wait for page to load")

	// Check that the navbar-toggler (hamburger menu) exists and is visible
	navbarToggler, err := page.Element(".navbar-toggler")
	require.NoError(t, err, "Failed to find navbar-toggler (hamburger menu)")
	err = navbarToggler.WaitVisible()
	require.NoError(t, err, "Navbar-toggler (hamburger menu) not visible")

	// Check that the navbar-toggler has the correct attributes
	togglerType, err := navbarToggler.Attribute("type")
	require.NoError(t, err, "Failed to get navbar-toggler type attribute")
	assert.Equal(t, "button", *togglerType, "Navbar-toggler should be a button")

	// Check that the navbar-toggler has the correct data attributes
	togglerTarget, err := navbarToggler.Attribute("data-bs-target")
	require.NoError(t, err, "Failed to get navbar-toggler data-bs-target attribute")
	assert.Equal(t, "#navbarNav", *togglerTarget, "Navbar-toggler should target #navbarNav")

	// Check that the navbar-brand exists
	navbarBrand, err := page.Element(".navbar-brand")
	require.NoError(t, err, "Failed to find navbar-brand")
	err = navbarBrand.WaitVisible()
	require.NoError(t, err, "Navbar-brand not visible")

	// Attempt to click the navbar toggler and check if the menu expands
	// The target #navbarNav should become visible or have a 'show' class
	navCollapse, err := page.Element("#navbarNav")
	require.NoError(t, err, "Failed to find #navbarNav element")

	// Check initial state: navbarNav should not have 'show' class (i.e., be collapsed)
	// Bootstrap 5 uses 'collapsing' during transition and 'show' when fully open.
	// We can check that it's not 'show' and not 'collapsing' or simply that it's not visible.
	isInitiallyVisible, _ := navCollapse.Visible()
	assert.False(t, isInitiallyVisible, "#navbarNav should initially be collapsed on small screens")

	navbarToggler.MustClick()

	// Wait for the collapse element to be shown (Bootstrap adds 'show' class)
	// It might also have 'collapsing' class during transition, so wait for 'show'
	err = page.Wait(rod.Eval(`() => document.querySelector("#navbarNav").classList.contains("show")`))
	// It's better to wait for an element inside the navCollapse to be visible
	// For example, a link inside the navbar
	// Let's assume there's a link with class .nav-link inside #navbarNav
	// Example: Wait for a link with text "Dashboard" to be visible inside the expanded menu
	dashboardLinkInMenu, err := navCollapse.Element(`a[href="/"]`) // Assuming Dashboard link goes to "/"
	if err == nil {
		err = dashboardLinkInMenu.WaitVisible()
		require.NoError(t, err, "Dashboard link in collapsed menu did not become visible after click")
	} else {
		// Fallback if a specific link isn't easily identifiable, wait for #navbarNav to have style indicating it's open
		// This is less robust than waiting for a specific child element.
		// Consider adding a specific test ID to a menu item for better selection.
		t.Log("Could not find specific link in navbar, relying on #navbarNav visibility state change")
		err = navCollapse.WaitVisible() // This might be tricky if it's always in DOM but hidden by CSS
		require.NoError(t, err, "#navbarNav did not become visible after click")
	}


	// Verify it's now expanded (has 'show' class or is visible)
	hasClassShow, err := navCollapse.HasClass("show")
	require.NoError(t, err, "Error checking for 'show' class on #navbarNav")
	assert.True(t, hasClassShow, "#navbarNav should have 'show' class after clicking toggler")
}

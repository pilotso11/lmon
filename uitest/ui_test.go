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
	launch := launcher.New().Headless(true)
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
	assert.Equal(t, "Monitoring Dashboard", h1Text)

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

	// Wait for data to load
	time.Sleep(2 * time.Second)

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

		// Wait for the modal to appear
		time.Sleep(1 * time.Second)

		// Find the threshold value in the modal
		modalBody, err := page.Element("#modal-body")
		require.NoError(t, err, "Failed to find modal body")
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

	for _, card := range configCards {
		text, err := card.Text()
		require.NoError(t, err, "Failed to get card text")
		if text == "Disk Monitoring" {
			diskCard = card
		} else if text == "System Monitoring" {
			systemCard = card
		}
	}

	assert.NotNil(t, diskCard, "Disk Monitoring card not found")
	assert.NotNil(t, systemCard, "System Monitoring card not found")

	// Wait for data to load
	time.Sleep(2 * time.Second)

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

package uitest

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/stretchr/testify/assert"
)

// TestUI runs UI tests for the lmon web interface
func TestUI(t *testing.T) {
	// Launch a new browser
	launcher := launcher.New().Headless(true)
	url, err := launcher.Launch()
	if err != nil {
		t.Fatalf("Failed to launch browser: %v", err)
	}
	browser := rod.New().ControlURL(url).MustConnect()
	defer browser.MustClose()

	// Set a timeout for the entire test
	browser = browser.Timeout(30 * time.Second)

	// Base URL for the application
	baseURL := "http://localhost:8080"

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
	page.MustWaitLoad()

	// Check the title
	title := page.MustElement("title").MustText()
	assert.Contains(t, title, "lmon")

	// Check that the dashboard elements are present
	assert.True(t, page.MustHas("h1"))
	assert.Equal(t, "Monitoring Dashboard", page.MustElement("h1").MustText())

	// Check that the system card is present
	systemCards := page.MustElements(".card-header")
	var systemCard *rod.Element
	for _, card := range systemCards {
		if card.MustText() == "System" {
			systemCard = card
			break
		}
	}
	assert.NotNil(t, systemCard, "System card not found")

	// Check that the disk card is present
	var diskCard *rod.Element
	for _, card := range systemCards {
		if card.MustText() == "Disk" {
			diskCard = card
			break
		}
	}
	assert.NotNil(t, diskCard, "Disk card not found")

	// Check that the health checks card is present
	var healthCard *rod.Element
	for _, card := range systemCards {
		if card.MustText() == "Health Checks" {
			healthCard = card
			break
		}
	}
	assert.NotNil(t, healthCard, "Health Checks card not found")

	// Wait for data to load
	time.Sleep(2 * time.Second)

	// Test that percentages are properly rounded to 2 decimal places
	// This assumes there's at least one percentage value displayed
	listItems := page.MustElements(".list-group-item")
	var percentageElements []*rod.Element
	for _, item := range listItems {
		spans := item.MustElements("span")
		for _, span := range spans {
			text := span.MustText()
			if strings.Contains(text, "%") {
				percentageElements = append(percentageElements, span)
			}
		}
	}

	if len(percentageElements) > 0 {
		for _, elem := range percentageElements {
			text := elem.MustText()
			// Check that the percentage has at most 2 decimal places
			assert.Regexp(t, `\d+\.\d{1,2}%`, text, "Percentage should be rounded to 2 decimal places")
		}
	}

	// Test that memory icon is correct
	memoryListItems := page.MustElements(".list-group-item")
	var memoryElement *rod.Element
	for _, item := range memoryListItems {
		if strings.Contains(item.MustText(), "Memory Usage") {
			memoryElement = item
			break
		}
	}

	if memoryElement != nil {
		memoryIcon := memoryElement.MustElement(".material-icons")
		assert.Equal(t, "memory", memoryIcon.MustText(), "Memory icon should be memory")
	}
}

// testConfiguration tests the configuration page
func testConfiguration(t *testing.T, browser *rod.Browser, baseURL string) {
	// Navigate to the configuration page
	page := browser.MustPage(fmt.Sprintf("%s/config", baseURL))
	defer page.MustClose()

	// Wait for the page to load
	page.MustWaitLoad()

	// Check the title
	title := page.MustElement("title").MustText()
	assert.Contains(t, title, "lmon")

	// Check that the configuration elements are present
	assert.True(t, page.MustHas("h1"))
	assert.Equal(t, "Configuration", page.MustElement("h1").MustText())

	// Check that the disk configuration card is present
	configCards := page.MustElements(".card-header")
	var diskCard *rod.Element
	var systemCard *rod.Element

	for _, card := range configCards {
		text := card.MustText()
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
	configItems := page.MustElements(".config-item")
	for _, elem := range configItems {
		// Check if this element contains a strong tag with text "/"
		strongElements := elem.MustElements("strong")
		for _, strong := range strongElements {
			pathText := strong.MustText()
			if pathText == "/" {
				// Check that there's no delete button
				deleteButtons, err := elem.Elements(".delete-btn")
				if err != nil {
					t.Fatalf("Failed to get delete buttons: %v", err)
				}
				assert.Equal(t, 0, len(deleteButtons), "Root partition should not have a delete button")
				break
			}
		}
	}

	// Test that memory icon is correct in system configuration
	var memoryMonitoringElement *rod.Element
	configItems = page.MustElements(".config-item")
	for _, item := range configItems {
		if strings.Contains(item.MustText(), "Memory Monitoring") {
			memoryMonitoringElement = item
			break
		}
	}

	if memoryMonitoringElement != nil {
		memoryIcon := memoryMonitoringElement.MustElement(".material-icons")
		assert.Equal(t, "memory", memoryIcon.MustText(), "Memory icon should be memory")
	}
}

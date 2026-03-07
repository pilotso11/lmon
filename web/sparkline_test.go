package web

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"lmon/db"
)

// TestGenerateSparklineSVG_Empty verifies that an empty snapshot slice returns an empty SVG.
func TestGenerateSparklineSVG_Empty(t *testing.T) {
	svg := GenerateSparklineSVG(nil)
	assert.Contains(t, string(svg), "<svg")
	assert.Contains(t, string(svg), "</svg>")
	assert.NotContains(t, string(svg), "<rect")
}

// TestGenerateSparklineSVG_SingleGreen verifies a single green point produces one green rect.
func TestGenerateSparklineSVG_SingleGreen(t *testing.T) {
	snapshots := []db.MonitorSnapshot{
		{RecordedAt: time.Now(), Status: "Green"},
	}
	svg := GenerateSparklineSVG(snapshots)
	svgStr := string(svg)
	assert.Contains(t, svgStr, "<svg")
	assert.Contains(t, svgStr, "</svg>")
	assert.Contains(t, svgStr, "<rect")
	assert.Contains(t, svgStr, `fill="#28a745"`, "Green should use green color")
}

// TestGenerateSparklineSVG_MixedStatuses verifies correct colors for each status type.
func TestGenerateSparklineSVG_MixedStatuses(t *testing.T) {
	snapshots := []db.MonitorSnapshot{
		{RecordedAt: time.Now(), Status: "Green"},
		{RecordedAt: time.Now(), Status: "Amber"},
		{RecordedAt: time.Now(), Status: "Red"},
		{RecordedAt: time.Now(), Status: "Error"},
	}
	svg := GenerateSparklineSVG(snapshots)
	svgStr := string(svg)

	assert.Contains(t, svgStr, `fill="#28a745"`, "Green color")
	assert.Contains(t, svgStr, `fill="#ffc107"`, "Amber color")
	assert.Contains(t, svgStr, `fill="#dc3545"`, "Red color")
	assert.Contains(t, svgStr, `fill="#6c757d"`, "Error/Unknown color")

	// Should have exactly 4 rects
	assert.Equal(t, 4, strings.Count(svgStr, "<rect"), "should have 4 bars")
}

// TestGenerateSparklineSVG_UnknownStatus verifies unknown statuses are handled.
func TestGenerateSparklineSVG_UnknownStatus(t *testing.T) {
	snapshots := []db.MonitorSnapshot{
		{RecordedAt: time.Now(), Status: "Unknown"},
	}
	svg := GenerateSparklineSVG(snapshots)
	svgStr := string(svg)
	assert.Contains(t, svgStr, `fill="#6c757d"`, "Unknown should use grey color")
}

// TestGenerateSparklineSVG_ManyPoints verifies the sparkline handles many points.
func TestGenerateSparklineSVG_ManyPoints(t *testing.T) {
	snapshots := make([]db.MonitorSnapshot, 200)
	for i := range snapshots {
		snapshots[i] = db.MonitorSnapshot{RecordedAt: time.Now(), Status: "Green"}
	}
	svg := GenerateSparklineSVG(snapshots)
	svgStr := string(svg)
	assert.Contains(t, svgStr, "<svg")
	assert.Contains(t, svgStr, "</svg>")
	// Bar width should be at minimum 1 even with many points
	assert.Equal(t, 200, strings.Count(svgStr, "<rect"), "should have 200 bars")
}

// TestGenerateSparklineSVG_SVGDimensions verifies the SVG has correct dimensions.
func TestGenerateSparklineSVG_SVGDimensions(t *testing.T) {
	snapshots := []db.MonitorSnapshot{
		{RecordedAt: time.Now(), Status: "Green"},
	}
	svg := GenerateSparklineSVG(snapshots)
	svgStr := string(svg)
	assert.Contains(t, svgStr, `width="100"`)
	assert.Contains(t, svgStr, `height="20"`)
}

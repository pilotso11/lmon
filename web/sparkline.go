package web

import (
	"fmt"
	"html/template"
	"strings"

	"lmon/db"
)

// GenerateSparklineSVG creates an inline SVG sparkline from monitor snapshots.
// Maps status to height: Green=100%, Amber=66%, Red=33%, Error=10%.
// Returns an empty SVG if no snapshots are provided.
func GenerateSparklineSVG(snapshots []db.MonitorSnapshot) template.HTML {
	if len(snapshots) == 0 {
		return template.HTML(`<svg width="100" height="20" xmlns="http://www.w3.org/2000/svg"></svg>`)
	}

	width := 100
	height := 20
	barWidth := float64(width) / float64(len(snapshots))
	if barWidth < 1 {
		barWidth = 1
	}

	var bars strings.Builder
	for i, snap := range snapshots {
		var barHeight float64
		var color string
		switch snap.Status {
		case "Green":
			barHeight = float64(height)
			color = "#28a745"
		case "Amber":
			barHeight = float64(height) * 0.66
			color = "#ffc107"
		case "Red":
			barHeight = float64(height) * 0.33
			color = "#dc3545"
		default:
			barHeight = float64(height) * 0.1
			color = "#6c757d"
		}
		x := float64(i) * barWidth
		y := float64(height) - barHeight
		bars.WriteString(fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s"/>`, x, y, barWidth, barHeight, color))
	}

	svg := fmt.Sprintf(`<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg">%s</svg>`, width, height, bars.String())
	return template.HTML(svg)
}

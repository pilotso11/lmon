package web

import (
	"lmon/config"
	"lmon/monitors"
	"lmon/monitors/disk"
	"lmon/monitors/healthcheck"
	"lmon/monitors/system"
)

type UIResult struct {
	ID          string       // Unique identifier for the monitor
	Icon        string       // Icon representing the monitor type
	Status      monitors.RAG // The RAG status of the check
	Value       string       // Human-readable value or message
	Value2      string       // Optional second value for additional context
	Group       string       // Group/category of the monitor
	TypeLabel   string       // Type of the monitor (e.g., disk, healthcheck)
	DisplayName string       // Display Name for UI
	Threshold   int          // Threshold for the monitor, if applicable
	EvenRow     bool         // Flag to indicate if the row is even for styling purposes
	StatusClass string       // CSS class for the status, used for styling in the UI
}

// Find Icon and add it to the result.
func newUIResult(id string, item monitors.Result, c *config.Config) UIResult {
	icon := "folder" // default Icon if no specific Icon is found
	threshold := 0   // default threshold if not set
	typeLabel := ""
	switch item.Group {
	case disk.Group:
		icon = disk.Icon // fallback to the default disk Icon
		typeLabel = "Disk"
		for k, d := range c.Monitoring.Disk {
			if item.Group+"_"+k == id {
				icon = d.Icon
				threshold = d.Threshold
				break
			}
		}
	case healthcheck.Group:
		typeLabel = "Health"
		icon = healthcheck.Icon // fallback to the default health Icon
		for k, h := range c.Monitoring.Healthcheck {
			if item.Group+"_"+k == id {
				icon = h.Icon
				break
			}
		}
	case system.Group:
		typeLabel = "System"
		switch item.DisplayName {
		case system.CPUDisplayName:
			icon = c.Monitoring.System.CPU.Icon
			threshold = c.Monitoring.System.CPU.Threshold
		case system.MemDisplayName:
			icon = c.Monitoring.System.Memory.Icon
			threshold = c.Monitoring.System.Memory.Threshold
		}
	default:
		// fallback to a generic Icon if no specific Icon is found
	}

	statusClass := ""
	switch item.Status {
	case monitors.RAGError:
		statusClass = "status-critical"
	case monitors.RAGAmber:
		statusClass = "status-warning"
	case monitors.RAGGreen:
		statusClass = "status-ok"
	case monitors.RAGRed:
		statusClass = "status-error"
	default:
		statusClass = "status-unknown"
	}

	return UIResult{
		ID:          id,
		Icon:        icon,
		Status:      item.Status,
		Value:       item.Value,
		Value2:      item.Value2,
		Group:       item.Group,
		DisplayName: item.DisplayName,
		Threshold:   threshold,
		TypeLabel:   typeLabel,
		StatusClass: statusClass,
	}
}

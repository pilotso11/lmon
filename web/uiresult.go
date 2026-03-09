package web

import (
	"lmon/config"
	"lmon/monitors"
	"lmon/monitors/disk"
	"lmon/monitors/docker"
	"lmon/monitors/healthcheck"
	"lmon/monitors/k8sevents"
	"lmon/monitors/k8snodes"
	"lmon/monitors/k8sservice"
	"lmon/monitors/ping"
	"lmon/monitors/system"
)

type UIResult struct {
	ID                   string       // Unique identifier for the monitor
	Icon                 string       // Icon representing the monitor type
	Status               monitors.RAG // The RAG status of the check
	Value                string       // Human-readable value or message
	Value2               string       // Optional second value for additional context
	Group                string       // Group/category of the monitor
	TypeLabel            string       // Type of the monitor (e.g., disk, healthcheck)
	DisplayName          string       // Display Name for UI
	Threshold            int          // Threshold for the monitor, if applicable
	AlertThreshold       int          // Number of consecutive failures before triggering alert
	FailureCount         int          // Current consecutive failure count
	EvenRow              bool         // Flag to indicate if the row is even for styling purposes
	StatusClass          string       // CSS class for the status, used for styling in the UI
	HasRestartContainers bool         // Whether this healthcheck has restart containers configured
}

// Find icon and add it to the result.
func newUIResult(id string, item monitors.Result, c *config.Config, failureCount int) UIResult {
	icon := "folder" // default icon if no specific icon is found
	threshold := 0   // default threshold if not set
	alertThreshold := 1 // default alert threshold
	typeLabel := ""
	hasRestartContainers := false
	switch item.Group {
	case disk.Group:
		icon = disk.Icon // fallback to the default disk icon
		typeLabel = "Disk"
		for k, d := range c.Monitoring.Disk {
			if item.Group+"_"+k == id {
				icon = d.Icon
				threshold = d.Threshold
				alertThreshold = d.AlertThreshold
				if alertThreshold <= 0 {
					alertThreshold = 1
				}
				break
			}
		}
	case healthcheck.Group:
		typeLabel = "Health"
		icon = healthcheck.Icon // fallback to the default health icon
		for k, h := range c.Monitoring.Healthcheck {
			if item.Group+"_"+k == id {
				icon = h.Icon
				hasRestartContainers = h.RestartContainers != ""
				alertThreshold = h.AlertThreshold
				if alertThreshold <= 0 {
					alertThreshold = 1
				}
				break
			}
		}
	case ping.Group:
		typeLabel = "Ping"
		icon = ping.Icon // fallback to the default ping icon
		for k, p := range c.Monitoring.Ping {
			if item.Group+"_"+k == id {
				icon = p.Icon
				threshold = p.AmberThreshold
				alertThreshold = p.AlertThreshold
				if alertThreshold <= 0 {
					alertThreshold = 1
				}
				break
			}
		}
	case system.Group:
		typeLabel = "System"
		switch item.DisplayName {
		case system.CPUDisplayName:
			icon = c.Monitoring.System.CPU.Icon
			threshold = c.Monitoring.System.CPU.Threshold
			alertThreshold = c.Monitoring.System.CPU.AlertThreshold
			if alertThreshold <= 0 {
				alertThreshold = 1
			}
		case system.MemDisplayName:
			icon = c.Monitoring.System.Memory.Icon
			threshold = c.Monitoring.System.Memory.Threshold
			alertThreshold = c.Monitoring.System.Memory.AlertThreshold
			if alertThreshold <= 0 {
				alertThreshold = 1
			}
		}
	case docker.Group:
		typeLabel = "Docker"
		icon = docker.Icon // fallback to the default docker icon
		for k, d := range c.Monitoring.Docker {
			if item.Group+"_"+k == id {
				icon = d.Icon
				threshold = d.Threshold
				alertThreshold = d.AlertThreshold
				if alertThreshold <= 0 {
					alertThreshold = 1
				}
				break
			}
		}
	case k8sevents.Group:
		typeLabel = "K8s Events"
		icon = k8sevents.Icon // fallback to the default k8s events icon
		for k, e := range c.Monitoring.K8sEvents {
			if item.Group+"_"+k == id {
				icon = e.Icon
				threshold = e.Threshold
				alertThreshold = e.AlertThreshold
				if alertThreshold <= 0 {
					alertThreshold = 1
				}
				break
			}
		}
	case k8snodes.Group:
		typeLabel = "K8s Nodes"
		icon = k8snodes.Icon // fallback to the default k8s nodes icon
		for k, n := range c.Monitoring.K8sNodes {
			if item.Group+"_"+k == id {
				icon = n.Icon
				alertThreshold = n.AlertThreshold
				if alertThreshold <= 0 {
					alertThreshold = 1
				}
				break
			}
		}
	case k8sservice.Group:
		typeLabel = "K8s Service"
		icon = k8sservice.Icon // fallback to the default k8s service icon
		for k, svc := range c.Monitoring.K8sService {
			if item.Group+"_"+k == id {
				icon = svc.Icon
				threshold = svc.Threshold
				alertThreshold = svc.AlertThreshold
				if alertThreshold <= 0 {
					alertThreshold = 1
				}
				break
			}
		}
	default:
		// fallback to a generic icon if no specific icon is found
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
		ID:                   id,
		Icon:                 icon,
		Status:               item.Status,
		Value:                item.Value,
		Value2:               item.Value2,
		Group:                item.Group,
		DisplayName:          item.DisplayName,
		Threshold:            threshold,
		AlertThreshold:       alertThreshold,
		FailureCount:         failureCount,
		TypeLabel:            typeLabel,
		StatusClass:          statusClass,
		HasRestartContainers: hasRestartContainers,
	}
}

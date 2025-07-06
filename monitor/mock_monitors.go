package monitor

import (
	"github.com/shirou/gopsutil/v3/disk"
)

// AlwaysHealthyDiskUsageProvider always returns a healthy disk usage stat for any path.
type AlwaysHealthyDiskUsageProvider struct{}

func (m *AlwaysHealthyDiskUsageProvider) Usage(path string) (*disk.UsageStat, error) {
	return &disk.UsageStat{
		Path:        path,
		Total:       1000000000,
		Free:        800000000,
		Used:        200000000,
		UsedPercent: 20.0,
	}, nil
}

// AlwaysNoopWebhookSender is a mock WebhookSenderInterface that does nothing and always succeeds.
type AlwaysNoopWebhookSender struct{}

func (s *AlwaysNoopWebhookSender) Send(url string, item *Item) error {
	return nil
}

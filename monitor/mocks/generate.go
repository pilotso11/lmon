//go:generate go run go.uber.org/mock/mockgen -destination=./disk_monitor_mock.go -package=mocks lmon/monitor DiskMonitorInterface
//go:generate go run go.uber.org/mock/mockgen -destination=./system_monitor_mock.go -package=mocks lmon/monitor SystemMonitorInterface
//go:generate go run go.uber.org/mock/mockgen -destination=./health_monitor_mock.go -package=mocks lmon/monitor HealthMonitorInterface
//go:generate go run go.uber.org/mock/mockgen -destination=./webhook_sender_mock.go -package=mocks lmon/monitor WebhookSenderInterface
//go:generate go run go.uber.org/mock/mockgen -destination=./http_client_mock.go -package=mocks lmon/monitor HTTPClientInterface

package mocks
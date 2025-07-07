package healthcheck

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"lmon/config"
	"lmon/monitors"
)

const Icon = "activity"
const Group = "app"

// UsageProvider is an interface for getting healthcheck usage
type UsageProvider interface {
	Check(ctx context.Context, path *url.URL, timeout int) (*http.Response, error)
}

// DefaultHealthcheckProvider is the default implementation of HealthcheckUsageProvider
type DefaultHealthcheckProvider struct {
	client http.Client
}

func NewDefaultHealthcheckProvider(msTimeout int) *DefaultHealthcheckProvider {
	if msTimeout == 0 {
		msTimeout = 5
	}
	return &DefaultHealthcheckProvider{
		client: http.Client{
			Timeout: time.Millisecond * time.Duration(msTimeout),
		},
	}
}

// Check returns healthcheck usage statistics
func (p *DefaultHealthcheckProvider) Check(ctx context.Context, path *url.URL, msTimeout int) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(msTimeout)*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", path.String(), nil)
	if err != nil {
		return nil, err
	}
	return p.client.Do(req)
}

type Healthcheck struct {
	name    string
	timeout int
	url     *url.URL
	icon    string
	impl    UsageProvider
}

func NewHealthcheck(name string, urlRaw string, timeout int, icon string, impl UsageProvider) (Healthcheck, error) {
	if icon == "" {
		icon = Icon
	}
	if impl == nil {
		// todo: is the filesystem zfs?
		impl = NewDefaultHealthcheckProvider(0)
	}
	parsedUrl, err := url.Parse(urlRaw)
	if err != nil {
		return Healthcheck{}, err
	}
	return Healthcheck{
		name:    name,
		url:     parsedUrl,
		icon:    icon,
		impl:    impl,
		timeout: timeout,
	}, nil
}

func (d Healthcheck) DisplayName() string {
	u := fmt.Sprintf("%s://%s", d.url.Scheme, d.url.Host)
	return fmt.Sprintf("%s (%s)", d.name, u)
}

func (d Healthcheck) Group() string {
	return Group
}

func (d Healthcheck) Name() string {
	return fmt.Sprintf("healthcheck_%s", d.name)
}

func (d Healthcheck) Save(cfg *config.Config) {
	cfg.Monitoring.Healthcheck[d.name] = config.HealthcheckConfig{
		URL:     d.url.String(),
		Timeout: d.timeout,
		Icon:    d.icon,
	}
}

func (d Healthcheck) Check(ctx context.Context) monitors.Result {
	response, err := d.impl.Check(ctx, d.url, d.timeout)
	if err != nil {
		return monitors.Result{
			Status: monitors.RAGError,
			Value:  fmt.Sprintf("error getting healthcheck: %v", err),
		}
	}

	res := fmt.Sprintf("%d (%s)", response.StatusCode, response.Status)
	status := monitors.RAGGreen
	switch {
	case response.StatusCode >= 500:
		status = monitors.RAGRed
	case response.StatusCode >= 300 && response.StatusCode < 500:
		status = monitors.RAGAmber
	}

	return monitors.Result{
		Status: status,
		Value:  res,
	}
}

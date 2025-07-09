//go:screenshots
package uitest

import (
	"context"
	"testing"
	"time"

	"github.com/go-rod/rod/lib/devices"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"lmon/web"
)

func Test_GenerateScreenshots(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	s, _ := web.StartTestServer(ctx, t, "./screenshots.yaml")
	s.Start(ctx)

	browser := getBrowser(t)
	defer browser.MustClose()
	page := browser.MustPage(s.ServerUrl)
	defer page.MustClose()

	_, err := page.Timeout(1 * time.Second).Element(`#system-items .list-group-item[data-id="system_cpu"]`)
	require.NoError(t, err, "should find cpu monitor")
	page.MustScreenshot("../screenshots/dashboard.png")

	el, err := page.Element(`a.nav-link[href="/config"]`)
	require.NoError(t, err, "should find config link")
	el.MustClick()
	page.Timeout(1 * time.Second).MustElement(`#add-disk-form`)
	page.MustScreenshot("../screenshots/config.png")

	el, err = page.Element(`a.nav-link[href="/mobile"]`)
	require.NoError(t, err, "should find mobile link")
	el.MustClick()
	// Iphone pro size
	err = page.Emulate(devices.IPhoneX)
	assert.NoError(t, err, "should emulate iphone pro size")
	page.Timeout(1 * time.Second).MustElement(`#mobile-items-list`)
	page.MustScreenshot("../screenshots/mobile.png")

}

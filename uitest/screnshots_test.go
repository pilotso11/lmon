//go:screenshots
package uitest

import (
	"context"
	"testing"
	"time"

	"github.com/go-rod/rod/lib/devices"
	"github.com/stretchr/testify/assert"

	"lmon/web"
)

func Test_GenerateScreenshots(t *testing.T) {
	assert.NotPanics(t, func() {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		s, _ := web.StartTestServer(ctx, t, "./screenshots.yaml")
		s.Start(ctx)

		browser := getBrowser(t)
		defer browser.MustClose()
		page := browser.MustPage(s.ServerUrl)
		defer page.MustClose()

		page.Timeout(1 * time.Second).MustElement(`#system-items .list-group-item[data-id="system_cpu"]`)
		page.MustScreenshot("../screenshots/dashboard.png")

		page.MustElement(`a.nav-link[href="/config"]`).MustClick()
		page.Timeout(1 * time.Second).MustElement(`#add-disk-form`)
		page.MustScreenshot("../screenshots/config.png")

		page.MustElement(`a.nav-link[href="/mobile"]`).MustClick()
		// Iphone pro size
		err := page.Emulate(devices.IPhoneX)
		assert.NoError(t, err, "should emulate iphone pro size")
		page.Timeout(1 * time.Second).MustElement(`#mobile-items-list`)
		page.MustScreenshot("../screenshots/mobile.png")

	})
}

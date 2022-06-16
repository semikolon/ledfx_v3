package util

import (
	_ "embed"
	"fmt"
	"ledfx/logger"

	"fyne.io/systray"
)

func StartTray(url string) func() {
	return func() {
		systray.SetIcon(logo)
		systray.SetTooltip("LedFx")
		mOpen := systray.AddMenuItem("Open", "Open LedFx Web Interface in Browser")
		mGithub := systray.AddMenuItem("Github", "Open LedFx Github in Browser")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit", "Shutdown LedFx")
		go func() {
			for {
				select {
				case <-mOpen.ClickedCh:
					OpenBrowser(fmt.Sprintf("http://%s/#/?newCore=1", url))
				case <-mGithub.ClickedCh:
					OpenBrowser("https://github.com/LedFx/ledfx_rewrite")
				case <-mQuit.ClickedCh:
					systray.Quit()
					return
				}
			}
		}()
	}
}

func StopTray() {
	// TODO kill ledfx from here
	logger.Logger.WithField("category", "Systray Handler").Warnln("Closing systray...")
}

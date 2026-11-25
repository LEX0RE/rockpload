//go:build android || ios

package app

import (
	"fmt"
	"net/url"

	"fyne.io/fyne/v2"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

func (u *Updater) ApplyUpdate(onProgress UpdateProgress) error {
	logger.FuncDebug()

	apkURL, err := url.Parse(u.UpdateInfo.URL)
	if err != nil {
		return fmt.Errorf("invalid update URL: %v", err)
	}

	fyne.DoAndWait(func() {
		err = fyne.CurrentApp().OpenURL(apkURL)
	})
	if err != nil {
		return fmt.Errorf("failed to open browser for update: %v", err)
	}

	return nil
}

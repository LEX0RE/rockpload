//go:build !windows

package rocket_network

import "fmt"

func (ra *Auth) AuthenticateWithBrowserCookies() error {
	return fmt.Errorf("installed browser cookie authentication is only supported on Windows")
}

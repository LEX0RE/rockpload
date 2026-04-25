package tools

import (
	"lexore/rockpload/app/tools/logger"
	"os/exec"
	"runtime"
)

func OpenBrowser(url string) {
	logger.FuncDebug()
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "linux":
		cmd = "xdg-open"
	case "windows":
		cmd = "rundll32"
		args = append(args, "url.dll,FileProtocolHandler")
	case "darwin":
		cmd = "open"
	}
	args = append(args, url)
	exec.Command(cmd, args...).Start()
}
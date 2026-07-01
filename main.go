package main

//go:generate go run github.com/tc-hib/go-winres@latest simply --icon app/assets/logo.png

import (
	"os"
	"runtime"

	"github.com/LEX0RE/rockpload/app"
	"github.com/LEX0RE/rockpload/app/constant"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

var Version = "dev"

// This function run BEFORE main()
func init() {
	if runtime.GOOS == "windows" {
		// Handle Window Driver without OpenGL, but client will need to download opengl32.dll and libgallium_wgl.dll from mesa-dist-win on git
		os.Setenv("GALLIUM_DRIVER", "llvmpipe")
	}

	// Always before logger as logger need to know the paths to log to file
	constant.InitPaths(false)

	logger.SetLogger()
}

func main() {
	logger.Rlogger.Info("Starting Rockpload")
	logger.FuncDebug()

	app := app.NewApp(Version)

	app.Run()
}

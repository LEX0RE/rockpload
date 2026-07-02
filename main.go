package main

//go:generate go run github.com/tc-hib/go-winres@latest simply --icon app/assets/logo.png

import (
	"fmt"
	"os"
	"runtime"

	fyneApp "fyne.io/fyne/v2/app"

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
}

func main() {
	fmt.Printf("Init Rockpload (GOOS: %s)", runtime.GOOS)
	fApp := fyneApp.NewWithID("gg.lexore.rockpload")

	// Always before logger as logger need to know the paths to log to file
	constant.InitPaths(fApp)
	logger.SetLogger()

	logger.Rlogger.Info("Starting Rockpload")
	logger.FuncDebug()

	app := app.NewApp(Version, fApp)
	app.Run()
}

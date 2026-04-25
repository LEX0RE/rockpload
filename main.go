package main

import (
	"lexore/rockpload/app"
	"lexore/rockpload/app/tools/logger"
	"os"
	"runtime"
)

// TODO Upload when we detect Rocket League is closed

var Version = "dev"

// This function run BEFORE main()
func init() {
	if runtime.GOOS == "windows" {
		// Handle Window Driver without OpenGL, but client will need to download opengl32.dll and libgallium_wgl.dll from mesa-dist-win on git
		os.Setenv("GALLIUM_DRIVER", "llvmpipe")
	}
	
	logger.SetLogger()
}

func main() {

	logger.Rlogger.Debug("Starting Rockpload")
	logger.FuncDebug()

	app := app.NewApp(Version)

	app.Run()
}
package app

// import (
// 	"lexore/rockpload/logger"
// 	"strings"
// 	"time"

// 	ps "github.com/mitchellh/go-ps"
// )

// var processMonitoringCheckTime = time.Minute

// func isRocketLeagueRunning() bool {
// 	logger.Rlogger.Debug("monitoring:isRocketLeagueRunning")
// 	processes, _ := ps.Processes()

// 	for _, p := range processes {
// 		name := strings.ToLower(p.Executable())

// 		if name == "rocketleague.exe" || name == "rocket league.exe" {
// 			return true
// 		}
// 	}
// 	return false
// }

// func monitorRocketLeague(onStart func(), onStop func()) {
// 	logger.Rlogger.Debug("monitoring:isRocketLeagueRunning")
// 	wasRunning := false

//     ticker := time.NewTicker(autoUploadTickerTime)
//     defer ticker.Stop()

//     for {
//         select {
//         case <-ticker.C:
//             logger.Rlogger.Info("Auto Upload triggered")
// 			rockpload.LaunchUpload()
//         case <-rockpload.autoUploadChannel:
//             logger.Rlogger.Info("Stopping Auto Upload")
//             return
//         }
//     }

// 	for {
// 		running := isRocketLeagueRunning()

// 		if running && !wasRunning {
// 			onStart()
// 		}

// 		if !running && wasRunning {
// 			onStop()
// 		}

// 		wasRunning = running
// 		time.Sleep(processMonitoringCheckTime)
// 	}
// }
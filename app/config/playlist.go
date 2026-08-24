package config

import (
	"slices"

	"github.com/LEX0RE/rockpload/app/constant"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

type PlaylistFilterEntry struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func PlaylistDisplayName(playlist int, knownPlaylists []PlaylistFilterEntry) string {
	logger.FuncDebug()

	for _, entry := range knownPlaylists {
		if entry.ID == playlist && entry.Name != "" {
			return entry.Name
		}
	}

	return constant.PlaylistName(playlist)
}

func (sc *StorageConfig) AllowsPlaylist(playlist int) bool {
	logger.FuncDebug()

	listed := slices.ContainsFunc(sc.FilteredPlaylists, func(p PlaylistFilterEntry) bool {
		return p.ID == playlist
	})

	switch sc.PlaylistFilterStyle {
	case PlaylistFilterWhitelist:
		return listed
	case PlaylistFilterBlacklist:
		return !listed
	default:
		return true
	}
}

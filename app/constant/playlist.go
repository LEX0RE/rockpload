package constant

import (
	"slices"
	"strconv"
)

// playlistNames maps a Rocket League playlist ID to its human-readable name.
// Based from https://bakkesplugins.com/wiki/bakkesmod-sdk/enums/playlistids
var playlistNames = map[int]string{
	0:  "Casual",
	1:  "Casual Duel",
	2:  "Casual Doubles",
	3:  "Casual Standard",
	4:  "Casual Chaos",
	6:  "Private",
	7:  "Season",
	8:  "OfflineSplitscreen",
	9:  "Training",
	10: "Ranked Duel",
	11: "Ranked Doubles",
	13: "Ranked Standard",
	15: "Snow Day",
	16: "Experimental",
	17: "Hoops",
	18: "Rumble",
	19: "Workshop",
	20: "UGC Training Editor",
	21: "UGC Training",
	22: "Tournament",
	23: "Dropshot",
	25: "10th Anniversary",
	26: "Face It",
	27: "Ranked Hoops",
	28: "Ranked Rumble",
	29: "Ranked Dropshot",
	30: "Ranked Snow Day",
	31: "Haunted Ball",
	32: "Beach Ball",
	33: "Rugby",
	34: "Auto Tournament",
	35: "Rocket Labs",
	37: "Rum Shot",
	38: "God Ball",
	40: "Coop Vs AI",
	41: "Boomer Ball",
	43: "God Ball Doubles",
	44: "Special Snow Day",
	46: "Football",
	47: "Cubic",
	48: "Tactical Rumble",
	49: "Spring Loaded",
	50: "Speed Demon",
	52: "Rumble BM",
	54: "Knockout",
	55: "Thirdwheel",
	62: "Magnus Futball",
}

var PredefinedPlaylistIDs = sortedPlaylistIDs()

func sortedPlaylistIDs() []int {
	ids := make([]int, 0, len(playlistNames))
	for id := range playlistNames {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	return ids
}

func PlaylistName(playlist int) string {
	if name, ok := playlistNames[playlist]; ok {
		return name
	}

	return "Playlist " + strconv.Itoa(playlist)
}

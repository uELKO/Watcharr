package watchedutil

import (
	"strconv"

	"github.com/sbondCo/Watcharr/database/entity"
)

// Get biggest season watching or biggest season watched.
func GetLatestWatchedInTv(
	ws []entity.WatchedSeason,
	we []entity.WatchedEpisode,
) string {
	if len(ws) <= 0 && len(we) <= 0 {
		return ""
	}

	seasonNum := getLatestWatchedSeasonInTv(ws, we)
	episode := getLatestWatchedEpisodeInTv(we, seasonNum)

	if seasonNum > -1 && episode.ID != 0 {
		// If we have a season num and an episode
		return SeasonAndEpToReadable(seasonNum, episode.EpisodeNumber)
	} else if seasonNum > -1 {
		// If we only have a season num
		return "Season " + strconv.Itoa(seasonNum)
	} else if episode.ID != 0 {
		// If we only have an episode
		return SeasonAndEpToReadable(episode.SeasonNumber, episode.EpisodeNumber)
	}

	return ""
}

// Finds the season we should consider "current". Prefers a season that's
// explicitly WATCHING at the season level. If none is, also checks for a
// season that has episode-level entries but no season-level entry at all -
// that means the user is tracking it purely episode-by-episode without ever
// setting its season status, which still makes it the one in progress (eg a
// season that just got auto-finished by the season->episode cascade
// shouldn't shadow a later season being watched one episode at a time).
// Only falls back to the biggest explicitly-FINISHED season if neither
// of those find anything.
func getLatestWatchedSeasonInTv(ws []entity.WatchedSeason, we []entity.WatchedEpisode) int {
	knownSeasons := make(map[int]bool, len(ws))
	biggestWatched := -1
	biggestWatching := -1

	for i := range ws {
		v := &ws[i]
		knownSeasons[v.SeasonNumber] = true
		switch v.Status {
		case entity.WATCHING:
			if v.SeasonNumber > biggestWatching {
				biggestWatching = v.SeasonNumber
			}
		case entity.FINISHED:
			if v.SeasonNumber > biggestWatched {
				biggestWatched = v.SeasonNumber
			}
		}
	}

	if biggestWatching >= 0 {
		return biggestWatching
	}

	biggestOrphanSeason := -1
	for i := range we {
		sn := we[i].SeasonNumber
		if !knownSeasons[sn] && sn > biggestOrphanSeason {
			biggestOrphanSeason = sn
		}
	}
	if biggestOrphanSeason >= 0 {
		return biggestOrphanSeason
	}

	return biggestWatched
}

func getLatestWatchedEpisodeInTv(
	we []entity.WatchedEpisode,
	seasonNum int,
) entity.WatchedEpisode {
	if len(we) <= 0 {
		return entity.WatchedEpisode{}
	}

	biggestWatchedIdx := -1
	biggestWatchingIdx := -1

	for i := range we {
		v := &we[i]

		if seasonNum >= 0 && v.SeasonNumber != seasonNum {
			// If we have a seasonNum, ensure the episode we find is in
			// that season.
			continue
		}

		switch v.Status {
		case entity.WATCHING:
			if biggestWatchingIdx == -1 {
				biggestWatchingIdx = i
				continue
			}
			if v.EpisodeNumber > we[biggestWatchingIdx].EpisodeNumber ||
				v.SeasonNumber > we[biggestWatchingIdx].SeasonNumber {
				biggestWatchingIdx = i
			}
		case entity.FINISHED:
			if biggestWatchedIdx == -1 {
				biggestWatchedIdx = i
				continue
			}
			if v.EpisodeNumber > we[biggestWatchedIdx].EpisodeNumber ||
				v.SeasonNumber > we[biggestWatchedIdx].SeasonNumber {
				biggestWatchedIdx = i
			}
		}
	}

	if biggestWatchingIdx > -1 {
		return we[biggestWatchingIdx]
	} else if biggestWatchedIdx > -1 {
		return we[biggestWatchedIdx]
	}

	return entity.WatchedEpisode{}
}

func SeasonAndEpToReadable(
	seasonNum int,
	episodeNum int,
) string {
	return "S" + strconv.Itoa(seasonNum) + "E" + strconv.Itoa(episodeNum)
}

// GetWatchProgress returns (remainingEpisodes, progressPercent) for a show,
// using its cached total episode count (no TMDB call needed here - the
// caller already has Content loaded). Returns (0, 0) if the total is
// unknown (0), since we can't say anything meaningful without it.
func GetWatchProgress(numberOfEpisodes uint32, we []entity.WatchedEpisode) (int, int) {
	if numberOfEpisodes <= 0 {
		return 0, 0
	}
	watched := 0
	for _, e := range we {
		if e.Status == entity.FINISHED || e.Status == entity.DROPPED {
			watched++
		}
	}
	total := int(numberOfEpisodes)
	if watched > total {
		// Our watched count can exceed TMDB's total if it's gone stale
		// (eg content cached before a season was fully added upstream).
		watched = total
	}
	return total - watched, (watched * 100) / total
}

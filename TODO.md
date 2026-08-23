# Personal Roadmap (fork: uELKO/Watcharr)

Tracking custom features for this fork. Upstream: https://github.com/sbondCo/Watcharr
(`upstream` remote). Work happens on `custom-dev`, rebased periodically on `dev`
(kept in sync with `upstream/dev`).

Excluded on purpose: #1012 (IMDb rating + custom lists) — not planned for now.

## Quick wins

- [x] **#1052** — "In Library" / "Watched" badges in actor filmography
      https://github.com/sbondCo/Watcharr/issues/1052
      Mostly frontend: reuse the `ExtraDetails` poster-overlay pattern already used
      on /search, apply it to the person filmography grid.

- [x] **Discover: "Hide watched" filter** (not from an upstream issue, our own idea)
      Hides FINISHED items from Discover grids. Lives in a new reusable `FilterPopover`
      (trigger button + panel) instead of a standalone toggle, so genre/provider filters
      below can share the same panel instead of growing the title row.

- [x] **#869** — Homepage sections by watch status
      https://github.com/sbondCo/Watcharr/issues/869
      Layout/grouping by existing watch status, no new backend logic needed.

- [x] **Home "Watching" row: current episode badge** ("S3 E1", not from an upstream
      issue) — uses the `watchingSeason` string the server already computes
      (server/feature/watched/watchedutil), space-formatted for display only in
      `PosterEpisodeBadge.svelte`.

- [x] **#1008** — "Up Next" row (next unwatched episode per Watching show)
      https://github.com/sbondCo/Watcharr/issues/1008
      Adapted (not copy-pasted) from community draft PR #1069 — that draft used axios
      directly, predating the `req` wrapper migration. New `GET /watched/upnext`, new
      `UpNext.svelte` at the top of the homepage, reusing PosterEpisodeBadge/
      PosterProgressBar/PosterRating/PosterStatus verbatim so it's pixel- and behavior-
      identical to posters elsewhere. Not done: the "upcoming calendar" half of #1008
      (movies/episodes not yet released) — the PR's own notes flagged that as depending
      on a periodic refresh mechanism (#1048/#1049), separate scope from this row.

- [x] **Poster hover: TMDB community rating pill** (not from an upstream issue — asked
      as "IMDb rating on hover", scoped down to the TMDB score we already have instead
      of adding an OMDb API dependency for a real IMDb number). New
      `PosterCommunityRating.svelte`, shown app-wide on every poster's hover overlay.
      Fixed a real bug it surfaced: watched-list items built their `Rating` via
      `uint(VoteAverage)` instead of `uint(VoteAverage*10)` like every other conversion
      in the codebase, truncating the decimal (8.9 → 8) — nothing had rendered that
      field for watched-list media before, so it went unnoticed.

## Discover-filter extensions (same backend touch point, do together)

- [x] **#869** (genre-filter half) + **Discover: multi-select genre filter**
      Multi-select checklist (OR semantics) in the `FilterPopover` panel, new
      `/content/genres` endpoint, `WithGenres` on `DiscoverOptions`. Works for every
      filter mode including Trending — TMDB's `/trending` has no genre param, so that
      case filters the already-fetched results server-side using the `genre_ids` TMDB
      already includes on them (`discoverMultiTrending` in discover.go).

- [x] **Streaming-provider trending** (Netflix/Disney+/AppleTV etc. — no upstream issue)
      Multi-select checklist in the `FilterPopover` panel, next to Genre, with each
      provider's real TMDB logo. New `/content/watch-providers` endpoint, `WithWatchProviders`
      on `DiscoverOptions` (+ `watch_region`, which TMDB requires alongside it). Checked:
      unlike genres, TMDB doesn't include per-item provider data on trending results, so
      it's disabled (with tooltip) for Trending — no post-filter trick available there.
      Upstream's dead `DiscoverFilter.streaming` scaffolding was left alone; we filter as
      an add-on to Popular/Upcoming/In Theatres instead of a separate mode.

## Medium

- [x] **Home "Watching" row: "+N remaining episodes" badge / progress bar**
      Used the show's already-cached `Content.NumberOfEpisodes` (no extra TMDB call)
      vs FINISHED/DROPPED episode count → `remainingEpisodes` + `watchProgress` on
      `WatchedDto` (watchedutil.GetWatchProgress). Surfaced a real pre-existing bug:
      `getLatestWatchedSeasonInTv` only looked at season-level statuses, so a season
      auto-finished by the season→episode cascade kept "winning" over a later season
      being tracked purely episode-by-episode with no season-level entry of its own.
      Fixed + regression test added.

- [x] **#766** — Auto move "Finished" → "Planned" when a show gets a new season
      https://github.com/sbondCo/Watcharr/issues/766
      New recurring task "Refresh Finished Shows" (24h) in server/task/task.go. Only
      touches shows whose status is FINISHED, so a deliberately DROPPED show is never
      reset. Bundled with two related, not-from-an-issue cascades we built alongside it:
      marking a whole season FINISHED backfills any episode in it with no status yet
      (season.go's hookSeasonStatusChanged, episode.Service injected in after both
      services exist to resolve the two-way dependency), and once every season is
      FINISHED/DROPPED the show itself flips to FINISHED too - unless it's already
      DROPPED, which automation never overrides.

- [ ] **German metadata mode** (titles, overviews, covers via TMDB — related to #772)
      Don't build from scratch: PR #1079 already implements this well — a global
      `TMDB_LANG` server setting (dropdown, defaults en-US), with automatic English
      fallback per-field when a translation is missing (title/overview/poster), applied
      on detail pages, add/import, and search. Author runs it daily in production
      (French). Currently `CONFLICTING` against current `dev` (needs a rebase) and is a
      global, not per-user, setting — fine for our single-user case. Plan: fetch the PR
      branch, rebase onto our `custom-dev`, resolve conflicts, adopt as-is or set
      TMDB_LANG=de-DE by default for us.
      https://github.com/sbondCo/Watcharr/pull/1079

## Larger builds

- [ ] **#409** — Notifications for upcoming releases
      https://github.com/sbondCo/Watcharr/issues/409
      Scheduler exists; still need delivery mechanism (e.g. ntfy/webhook, as
      suggested in the issue), "already notified" tracking, user settings.

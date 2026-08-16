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

- [ ] **#869** — Homepage sections by watch status
      https://github.com/sbondCo/Watcharr/issues/869
      Layout/grouping by existing watch status, no new backend logic needed.

- [ ] **#1008** — Chronological "Up Next" view (upcoming episodes/movies + unwatched)
      https://github.com/sbondCo/Watcharr/issues/1008
      Unfinished community draft exists: PR #1069 (overview "Up Next" row) — evaluate
      adopting/finishing that instead of starting from scratch.

## Discover-filter extensions (same backend touch point, do together)

- [ ] **Streaming-provider trending** (Netflix/Disney+/AppleTV etc. — no upstream issue)
      Add `WithWatchProviders` + `WatchRegion` to `DiscoverOptions`
      (server/media/tmdb/tmdb_discover.go / structs.go:859), wire into a new Discover
      filter analogous to existing Popular/In Theatres/Upcoming. Display already
      solved via `ProviderIcon.svelte` / `ProvidersList.svelte`.

- [ ] **#869** (genre-filter half) — Add `WithGenres` to `DiscoverOptions` the same way,
      plus genre picker UI (TMDB `/genre/movie/list`, `/genre/tv/list`).

## Medium

- [ ] **#766** — Auto move "Finished" → "Planned" when a show gets a new season
      https://github.com/sbondCo/Watcharr/issues/766
      New recurring task in the existing gocron scheduler (server/task/task.go).

## Larger builds

- [ ] **#409** — Notifications for upcoming releases
      https://github.com/sbondCo/Watcharr/issues/409
      Scheduler exists; still need delivery mechanism (e.g. ntfy/webhook, as
      suggested in the issue), "already notified" tracking, user settings.

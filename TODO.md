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
      solved via `ProviderIcon.svelte` / `ProvidersList.svelte`. Note: upstream already
      has dead scaffolding for this — `DiscoverFilter.streaming` exists in types.ts and
      FilterDropDown.svelte's option map, but is never pushed into the actual options
      array and has no backend case. Worth checking their intent before building our own.
      The provider picker itself (which service) belongs in the `FilterPopover` panel.

- [ ] **#869** (genre-filter half) — Add `WithGenres` to `DiscoverOptions` the same way,
      plus genre picker UI (TMDB `/genre/movie/list`, `/genre/tv/list`) as another row in
      the `FilterPopover` panel, next to "Hide watched".

## Medium

- [ ] **#766** — Auto move "Finished" → "Planned" when a show gets a new season
      https://github.com/sbondCo/Watcharr/issues/766
      New recurring task in the existing gocron scheduler (server/task/task.go).

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

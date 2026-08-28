# Personal Roadmap (fork: uELKO/Watcharr)

Tracking custom features for this fork. Upstream: https://github.com/sbondCo/Watcharr
(`upstream` remote). Work happens on `custom-dev`, rebased periodically on `dev`
(kept in sync with `upstream/dev`).

Excluded on purpose: #1012 (custom lists) — not planned for now. (Its IMDb-rating
half is covered separately — see JustWatch scoring below.)

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
      identical to posters elsewhere. The "upcoming calendar" half (PLANNED movies/shows
      with a known future release date) was later added too — see below.

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

- [x] **Configurable UI: hide People (Discover) / hide Tags** (not from an upstream
      issue, our own idea). Two new per-user settings, `hideDiscoverPeople` and
      `hideTags`, following the existing `entity.UserSettings` pointer-field pattern.
      `hideDiscoverPeople` only affects Discover's `MediaTypeFilter` (new `hidePeople`
      prop) — Search uses the same component and is left untouched. `hideTags` hides
      the nav bar Tags icon/menu and the "add to tag" button on movie/TV/game pages.
      Toggles added to Profile → Settings, matching the existing Hide Spoilers pattern.

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

- [x] **Discover: fix double-popup on Year/Rating, persist filters per-user**
      (not from an upstream issue, our own idea). The Year/Rating `FilterPopover`
      panels had a nested `DropDown` inside them — two popups stacked for one
      filter. New `SingleSelectFilter` renders the options as a flat list
      directly in the panel instead, matching how Genres/Streaming already work
      as checklists. Also persists the whole filter selection (type, filter
      mode, hide watched, genres, providers, year, rating) server-side per-user
      as an opaque JSON blob (new `DiscoverFilters` setting) so refreshing the
      page no longer resets the filter bar.

- [x] **German metadata mode** (titles, overviews, covers via TMDB — related to #772)
      Adapted from upstream PR #1079 (not copy-pasted — its base predates a big
      refactor that moved movie/tv/search logic directly onto `*TMDB`, so the
      fallback logic was ported onto that current shape). New global `TMDB_LANG`
      server setting (dropdown, defaults en-US); English fallback per-field when
      a translation is missing (title/overview/poster on movie/show details,
      overview on search), only firing the extra en-US request when something
      is actually missing. New `/content/languages` endpoint backs the dropdown;
      picking a Default Country pre-fills a matching language (overridable).
      https://github.com/sbondCo/Watcharr/pull/1079

- [x] **Poster hover: move TMDB rating pill to top-left** (not from an upstream
      issue). It was overlapping another top-right badge in the hover overlay.

- [x] **Settings: hide "Following" nav button** (not from an upstream issue).
      Same `hideDiscoverPeople`/`hideTags` pattern — new `hideFollowing`
      per-user setting, toggle on the Profile page.

- [x] **Home: unify "Library" filter bar with Discover's UI** (not from an
      upstream issue, our own idea). Replaced the nav-bar Filter/Sort icon
      menus on Home with an inline bar using the exact same `PageTitle` +
      `MediaTypeFilter` + `FilterPopover` structure Discover uses — a
      "Library" title, the same Movies/TV Shows buttons (now single-select,
      mapped onto the existing multi-select `Filters.type`), Status/Sort as
      FilterPopover panels pushed right. Nav-bar Filter/Sort icons stay for
      Lists/Tag pages, which weren't part of this ask. Along the way: fixed
      Up Next's single-card row being pinned to the left instead of centered
      (`HorizontalList` gained a `center` prop using `justify-content: safe
      center`), and made the filter bar's width track whichever content is
      shown below (1200 in list view to match `PosterList`'s own cap, 1800 in
      grouped view) so it doesn't overhang past the content.

- [x] **Nav: JustWatch-style Home/Discover text links** (not from an upstream
      issue). With the app effectively down to two destinations, replaced the
      icon-only Discover nav button with explicit "Home"/"Discover" text
      links next to the logo (muted when inactive, highlighted for the
      current page), matching JustWatch's own top nav.

- [x] **Poster hover zoom reduced further (25% of previous growth)** (not
      from an upstream issue). `scale(1.2)`/`scale(1.1)` → `scale(1.05)`/
      `scale(1.025)`.

- [x] **Up Next: "upcoming calendar" half of #1008** — PLANNED movies/shows
      with a known future release date. Completes the deferred half of #1008
      above. New `UpNextItemKind` ("episode" | "release") in one response;
      release cards skip the episode badge/progress bar for a release-date
      badge instead, link to `/movie` or `/tv` based on content type, and
      don't repurpose the status picker (that only makes sense for episode
      cards) — status just updates normally.

- [x] **Nav: keep the nav bar fixed instead of hiding on scroll down** (not
      from an upstream issue). Removed the scroll listener that slid it out
      of view; it's just sticky at `top:0` now.

- [x] **Style: subtle blue tint on the dark theme background** (not from an
      upstream issue). `rgb(15,15,15)` → `rgb(11,13,20)`, light theme untouched.

- [x] **Discover: fix filter persistence lost on navigate-away** (not from an
      upstream issue). Two bugs: filters started at hardcoded defaults on
      mount and were only restored later inside `onMount`, so the persist
      effect could fire in between and overwrite the saved filters with
      those defaults; and `onDestroy` just cancelled the debounce timeout
      instead of flushing it, so a filter change followed quickly by
      navigating into a movie/show never got saved at all. Both fixed
      (restore synchronously at `$state` declaration time; flush pending
      changes on teardown). Same flush fix applied to the Charts page's
      provider-selection persistence, which had the identical pattern.

## Larger builds

- [x] **Charts page: Top 10 per streaming provider** (not from an upstream
      issue, our own idea). New `/charts` page — pick providers, see a ranked
      Top 10 (movies+shows merged) per provider with a rank-movement
      indicator (▲/▼). Originally approximated from TMDB popularity + our
      own daily snapshot table (TMDB has no trending-by-provider concept at
      all); replaced with JustWatch's own public GraphQL API (unofficial,
      undocumented - apis.justwatch.com/graphql, the same endpoint
      justwatch.com's frontend uses) once we found it returns real
      rank/trend data per provider directly - see `server/media/justwatch`.
      Own provider picker (JustWatch's own short names/icons, not TMDB's
      provider IDs); each entry's `tmdb_id` is used to pull full TMDB
      details so cards look like every other poster in the app. Snapshot
      table + task removed as no longer needed.

- [x] **Content pages: IMDb + Rotten Tomatoes scores** (not from an upstream
      issue — the scoped-down half of the original "IMDb rating" ask, now
      essentially free with JustWatch integration already in place for
      Charts). JustWatch has no lookup-by-TMDB-id query, so this searches by
      title and matches whichever result's own `tmdb_id` equals ours -
      best-effort, silently skipped if no match. New `Search` alongside
      `Popular`/`Providers` in `server/media/justwatch`; all three cached.

- [x] **Discover: genre exclude filter** (not from an upstream issue).
      Genre filter cycles three states per click, JustWatch-style: neutral →
      include → exclude → neutral. New `WithoutGenres`/`without_genres`
      threaded through every discover mode alongside the existing include
      list, including the Trending post-filter.

- [ ] **#409** — Notifications for upcoming releases
      https://github.com/sbondCo/Watcharr/issues/409
      Scheduler exists; still need delivery mechanism (e.g. ntfy/webhook, as
      suggested in the issue), "already notified" tracking, user settings.

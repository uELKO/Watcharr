<script lang="ts">
	import Spinner from "@/lib/Spinner.svelte";
	import { req } from "@/lib/util/api";
	import Poster from "@/lib/poster/Poster.svelte";
	import PageTitle from "@/lib/generic/PageTitle.svelte";
	import MediaTypeFilter from "@/lib/search/MediaTypeFilter.svelte";
	import infScroll from "@/lib/util/infScroll";
	import paginatedLoader, {
		PaginatedLoaderRunFnAction,
	} from "@/lib/util/paginatedLoader.svelte";
	import {
		DiscoverFilter,
		MediaTypeE,
		SearchType,
		type DiscoverRequest,
		type DropDownItem,
		type Media,
		type PaginationResponse,
	} from "@/types";
	import { page } from "$app/state";
	import { afterNavigate, goto } from "$app/navigation";
	import { onDestroy, onMount } from "svelte";
	import PosterList from "@/lib/poster/PosterList.svelte";
	import Error from "@/lib/Error.svelte";
	import PersonPoster from "@/lib/poster/PersonPoster.svelte";
	import FilterDropDown from "./FilterDropDown.svelte";
	import { resolve } from "$app/paths";
	import Checkbox from "@/lib/Checkbox.svelte";
	import FilterPopover from "@/lib/generic/FilterPopover.svelte";
	import GenreFilter from "./GenreFilter.svelte";
	import ProviderFilter from "./ProviderFilter.svelte";
	import SingleSelectFilter from "./SingleSelectFilter.svelte";
	import { store } from "@/store.svelte";

	const scroll = infScroll({ callback: onScrollToBottom });
	const dataLoader = paginatedLoader<Media, undefined>(load);

	const currentYear = new Date().getFullYear();
	// "X or newer", same style as the rating options below (not an exact year).
	const yearOptions: DropDownItem[] = [
		{ id: 0, value: "Any Year" },
		...Array.from({ length: currentYear - 1900 + 2 }, (_, i) => currentYear + 1 - i).map(
			(y) => ({ id: y, value: `${y}+` }) as DropDownItem,
		),
	];
	const ratingOptions: DropDownItem[] = [
		{ id: 0, value: "Any Rating" },
		{ id: 9, value: "9+" },
		{ id: 8, value: "8+" },
		{ id: 7, value: "7+" },
		{ id: 6, value: "6+" },
		{ id: 5, value: "5+" },
	];

	let discoverFilter: DiscoverFilter = $state(DiscoverFilter.trending);
	let hideWatchedFilter = $state(false);
	let selectedGenres: number[] = $state([]);
	let selectedProviders: number[] = $state([]);
	let selectedYear: string | number = $state(0);
	let selectedMinRating: string | number = $state(0);
	let discoverType: SearchType | undefined = $derived.by(() => {
		const t = page.url.searchParams.get("type");
		if (t) {
			return t as SearchType;
		}
		return SearchType.multi;
	});
	// Genre/Provider lists are per movie/tv, and the pickers only ever load
	// options for those two types.
	let typeFilterSupported = $derived(
		discoverType === SearchType.movie || discoverType === SearchType.show,
	);
	let genreDisabledReason = $derived(
		typeFilterSupported ? "" : "Only available for Movies or Shows.",
	);
	// Unlike genres, TMDB gives us no per-item provider data on trending
	// results, so there's no server-side post-filter trick available here -
	// providers are simply unsupported for Trending.
	let providerFilterSupported = $derived(discoverFilter !== DiscoverFilter.trending);
	let providerDisabledReason = $derived(
		!typeFilterSupported
			? "Only available for Movies or Shows."
			: !providerFilterSupported
				? "Not available for Trending (TMDB doesn't expose providers on trending results)."
				: "",
	);
	// Persisted (per-user, server-side) filter state, so the filter bar
	// survives a refresh instead of resetting every time. Stored as an
	// opaque JSON blob on the user's settings - the backend doesn't need to
	// understand its shape, only the frontend encodes/decodes it.
	interface DiscoverFilterState {
		type?: SearchType;
		filter?: DiscoverFilter;
		hideWatched?: boolean;
		genres?: number[];
		providers?: number[];
		year?: number;
		minRating?: number;
	}

	function loadSavedFilters(): DiscoverFilterState | undefined {
		const raw = store.userSettings?.discoverFilters;
		if (!raw) return undefined;
		try {
			return JSON.parse(raw) as DiscoverFilterState;
		} catch (err) {
			console.error("discover: Failed to parse saved filters", err);
			return undefined;
		}
	}

	let lastPersistedFilters = store.userSettings?.discoverFilters || "";
	let persistFiltersTimeout: ReturnType<typeof setTimeout> | undefined;

	function persistFilters(json: string) {
		clearTimeout(persistFiltersTimeout);
		persistFiltersTimeout = setTimeout(() => {
			lastPersistedFilters = json;
			if (store.userSettings) store.userSettings.discoverFilters = json;
			req.post("/user/update", { discoverFilters: json }).catch((err) => {
				console.error("discover: Failed to persist filters", err);
			});
		}, 600);
	}

	let nextLoadParams: DiscoverRequest = $derived({
		page: dataLoader.state.page + 1,
		type: discoverType,
		filter: discoverFilter,
		// Pipe separated = TMDB "match any of these genres/providers" (OR),
		// not AND. Genres/year/rating are supported for every filter mode,
		// including Trending: the server filters trending results itself
		// using data TMDB already includes on them (genre_ids, release date,
		// vote_average). Providers can't do that (no per-item provider data
		// on trending results at all), so those are dropped for Trending.
		genres: selectedGenres.length > 0 ? selectedGenres.join("|") : undefined,
		providers:
			providerFilterSupported && selectedProviders.length > 0
				? selectedProviders.join("|")
				: undefined,
		year: selectedYear ? Number(selectedYear) : undefined,
		minRating: selectedMinRating ? Number(selectedMinRating) : undefined,
	});

	async function load(signal: AbortSignal) {
		console.debug("load: loadParams:", nextLoadParams);
		if (nextLoadParams.page === dataLoader.state.page) {
			console.warn("load: Already on this page, not loading it again!");
			return;
		}
		const r = await req.get<PaginationResponse<Media, undefined>>(`/discover`, {
			params: nextLoadParams,
			signal,
		});
		scroll.dataLoaded();
		return r;
	}

	async function onScrollToBottom() {
		// If an error is being shown, no more infinite scroll.
		if (dataLoader.state.reqLoadError) {
			return;
		}
		dataLoader.runFn();
	}

	function setActiveDiscoverType(to: SearchType | undefined) {
		console.debug("setActiveDiscoverType: to:", to);
		const curLocation = new URL(page.url);
		if (!to || discoverType === to) {
			curLocation.searchParams.delete("type");
		} else {
			curLocation.searchParams.set("type", to);
		}
		// Running the goto will cause afterNavigate hook to be called,
		// which will run a fresh search, so nothing else to do here.
		goto(resolve(`/discover?${curLocation.searchParams.toString()}`));
	}

	$effect(() => {
		const snapshot: DiscoverFilterState = {
			type: discoverType,
			filter: discoverFilter,
			hideWatched: hideWatchedFilter,
			genres: selectedGenres,
			providers: selectedProviders,
			year: selectedYear ? Number(selectedYear) : undefined,
			minRating: selectedMinRating ? Number(selectedMinRating) : undefined,
		};
		const json = JSON.stringify(snapshot);
		if (json === lastPersistedFilters) return;
		persistFilters(json);
	});

	onMount(() => {
		const saved = loadSavedFilters();
		if (saved) {
			discoverFilter = saved.filter ?? discoverFilter;
			hideWatchedFilter = saved.hideWatched ?? false;
			selectedGenres = saved.genres ?? [];
			selectedProviders = saved.providers ?? [];
			selectedYear = saved.year ?? 0;
			selectedMinRating = saved.minRating ?? 0;
		}
		// discoverType is derived from the URL - only apply the saved type
		// when the URL doesn't already specify one (a shared/bookmarked link
		// should win), and do it via a replaced navigation so it doesn't
		// clutter browser history.
		if (saved?.type && saved.type !== SearchType.multi && !page.url.searchParams.get("type")) {
			const curLocation = new URL(page.url);
			curLocation.searchParams.set("type", saved.type);
			goto(resolve(`/discover?${curLocation.searchParams.toString()}`), {
				replaceState: true,
			});
		} else {
			dataLoader.runFn(PaginatedLoaderRunFnAction.Reset);
		}
	});

	afterNavigate(() => {
		console.log(
			"afterNavigate: Query changed, performing request.",
			"searchParams:",
			page.url.searchParams,
		);
		dataLoader.abortReq("navigated away");
		dataLoader.runFn(PaginatedLoaderRunFnAction.Reset);
	});

	onDestroy(() => {
		console.debug("DISCOVER PAGE DESTROYED");
		clearTimeout(persistFiltersTimeout);
		scroll.destroy();
		dataLoader.abortReq("page destroyed");
	});
</script>

<svelte:head>
	<title>Discover Content</title>
</svelte:head>

<div class="content">
	<div class="inner">
		<PageTitle title="Discover">
			<div class="pagetitle-mediatypefilter">
				<MediaTypeFilter
					active={discoverType}
					disabled={false}
					hidePeople={store.userSettings?.hideDiscoverPeople}
					onChange={(nowActive) => {
						// Reset discoverFilter as we change type filter
						// to avoid going into new type filter with unsupported
						// discoverFilter that was set in previous type.
						discoverFilter = DiscoverFilter.trending;
						setActiveDiscoverType(nowActive as SearchType | undefined);
					}}
				/>
			</div>
			<div class="pagetitle-filters">
				<div class="hide-watched">
					<span>Hide watched</span>
					<Checkbox name="Hide watched" bind:value={hideWatchedFilter} />
				</div>
				<FilterPopover label="Year" active={!!selectedYear}>
					<SingleSelectFilter
						options={yearOptions}
						bind:active={selectedYear}
						onChange={() => dataLoader.runFn(PaginatedLoaderRunFnAction.Reset)}
					/>
				</FilterPopover>
				<FilterPopover
					label="Genres"
					active={selectedGenres.length > 0}
					disabled={!!genreDisabledReason}
					disabledReason={genreDisabledReason}
				>
					<GenreFilter
						{discoverType}
						bind:active={selectedGenres}
						onChange={() => dataLoader.runFn(PaginatedLoaderRunFnAction.Reset)}
					/>
				</FilterPopover>
				<FilterPopover label="Rating" active={!!selectedMinRating}>
					<SingleSelectFilter
						options={ratingOptions}
						bind:active={selectedMinRating}
						onChange={() => dataLoader.runFn(PaginatedLoaderRunFnAction.Reset)}
					/>
				</FilterPopover>
				<FilterPopover
					label="Streaming"
					active={providerFilterSupported && selectedProviders.length > 0}
					disabled={!!providerDisabledReason}
					disabledReason={providerDisabledReason}
				>
					<ProviderFilter
						{discoverType}
						disabled={!providerFilterSupported}
						bind:active={selectedProviders}
						onChange={() => dataLoader.runFn(PaginatedLoaderRunFnAction.Reset)}
					/>
				</FilterPopover>
				{#if hideWatchedFilter || selectedGenres.length > 0 || selectedProviders.length > 0 || selectedYear || selectedMinRating}
					<button
						type="button"
						class="plain reset-all"
						onclick={() => {
							hideWatchedFilter = false;
							selectedGenres = [];
							selectedProviders = [];
							selectedYear = 0;
							selectedMinRating = 0;
							dataLoader.runFn(PaginatedLoaderRunFnAction.Reset);
						}}
					>
						✕ Reset
					</button>
				{/if}
				<FilterDropDown
					{discoverType}
					bind:active={discoverFilter}
					onChange={() => {
						console.log("Discover FilterDropDown Selected Change");
						dataLoader.runFn(PaginatedLoaderRunFnAction.Reset);
					}}
				/>
			</div>
		</PageTitle>

		<PosterList>
			{#if dataLoader.state.data?.length > 0}
				{#each dataLoader.state.data as w, i (`${i}-${w.type}`)}
					{#if w.type === MediaTypeE.tmdbPerson}
						<PersonPoster
							id={w.ids.tmdb}
							name={w.name}
							path={w.extPosterPath}
						/>
					{:else if w.type === MediaTypeE.tmdbMovie || w.type === MediaTypeE.tmdbShow || w.type === MediaTypeE.igdbGame}
						<Poster
							media={w}
							bind:watched={dataLoader.state.data[i].watched}
							fluidSize
							showStatusBadge
							hideIfWatched={hideWatchedFilter}
						/>
					{/if}
				{/each}
			{:else if !dataLoader.state.reqLoading && !dataLoader.state.reqLoadError}
				<!-- If request is running or we have an error, no point in showing 'no results' message. -->
				<h2 class="norm" title="Hovering over me doesn't change the facts ;(">
					No Results!
				</h2>
			{/if}
		</PosterList>

		{#if dataLoader.state.reqLoading}
			<div style="margin-bottom: 60px;">
				<Spinner />
			</div>
		{/if}

		{#if dataLoader.state.reqLoadError}
			<div style="margin-bottom: 60px;">
				<Error
					pretty="Failed to load results!"
					error={dataLoader.state.reqLoadError}
					onRetry={() => {
						dataLoader.state.reqLoadError = undefined;
						dataLoader.runFn(PaginatedLoaderRunFnAction.ResetIfOnFirstOrNoPage);
					}}
				/>
			</div>
		{/if}
	</div>
</div>

<style lang="scss">
	.pagetitle-mediatypefilter {
		@media screen and (max-width: 745px) {
			width: 100%;
			order: 2;
		}
	}

	.pagetitle-filters {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 14px;
		margin-left: auto;
	}

	.hide-watched {
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 14px;
	}

	.reset-all {
		width: min-content;
		white-space: nowrap;
		font-size: 14px;
		color: $text-color-accent;

		&:hover {
			color: $text-color;
		}
	}

	.content {
		display: flex;
		width: 100%;
		justify-content: center;

		.inner {
			width: 100%;
			max-width: 1200px;
		}
	}
</style>

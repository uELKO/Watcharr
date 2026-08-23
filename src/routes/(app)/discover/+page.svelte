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

	const scroll = infScroll({ callback: onScrollToBottom });
	const dataLoader = paginatedLoader<Media, undefined>(load);

	let discoverFilter: DiscoverFilter = $state(DiscoverFilter.trending);
	let hideWatchedFilter = $state(false);
	let selectedGenres: number[] = $state([]);
	let selectedProviders: number[] = $state([]);
	// Unlike genres, TMDB gives us no per-item provider data on trending
	// results, so there's no server-side post-filter trick available here -
	// providers are simply unsupported for Trending.
	let providerFilterSupported = $derived(discoverFilter !== DiscoverFilter.trending);
	let discoverType: SearchType | undefined = $derived.by(() => {
		const t = page.url.searchParams.get("type");
		if (t) {
			return t as SearchType;
		}
		return SearchType.multi;
	});
	let nextLoadParams: DiscoverRequest = $derived({
		page: dataLoader.state.page + 1,
		type: discoverType,
		filter: discoverFilter,
		// Pipe separated = TMDB "match any of these genres" (OR), not AND.
		// Supported for every filter mode, including Trending: the server
		// filters trending results itself using the genre_ids TMDB already
		// includes on them, since /trending has no genre query param.
		genres: selectedGenres.length > 0 ? selectedGenres.join("|") : undefined,
		providers:
			providerFilterSupported && selectedProviders.length > 0
				? selectedProviders.join("|")
				: undefined,
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

	onMount(() => {
		dataLoader.runFn(PaginatedLoaderRunFnAction.Reset);
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
				<FilterPopover
					active={hideWatchedFilter ||
						selectedGenres.length > 0 ||
						(providerFilterSupported && selectedProviders.length > 0)}
				>
					<div class="filter-row">
						<span>Hide watched</span>
						<Checkbox name="Hide watched" bind:value={hideWatchedFilter} />
					</div>
					<div class="filter-section">
						<GenreFilter
							{discoverType}
							bind:active={selectedGenres}
							onChange={() => dataLoader.runFn(PaginatedLoaderRunFnAction.Reset)}
						/>
					</div>
					<ProviderFilter
						{discoverType}
						disabled={!providerFilterSupported}
						bind:active={selectedProviders}
						onChange={() => dataLoader.runFn(PaginatedLoaderRunFnAction.Reset)}
					/>
				</FilterPopover>
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
		gap: 10px;
		margin-left: auto;
	}

	.filter-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 10px;
		padding-bottom: 8px;
		margin-bottom: 4px;
		border-bottom: 1px solid rgba(255, 255, 255, 0.1);
	}

	.filter-section {
		padding-bottom: 8px;
		margin-bottom: 4px;
		border-bottom: 1px solid rgba(255, 255, 255, 0.1);
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

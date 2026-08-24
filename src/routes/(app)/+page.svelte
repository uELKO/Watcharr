<script lang="ts">
	import { goto } from "$app/navigation";
	import { resolve } from "$app/paths";
	import Error from "@/lib/Error.svelte";
	import Icon from "@/lib/Icon.svelte";
	import HorizontalList from "@/lib/HorizontalList.svelte";
	import UpNext from "@/lib/UpNext.svelte";
	import Poster from "@/lib/poster/Poster.svelte";
	import PosterList from "@/lib/poster/PosterList.svelte";
	import Spinner from "@/lib/Spinner.svelte";
	import { req } from "@/lib/util/api";
	import infScroll from "@/lib/util/infScroll";
	import paginatedLoader from "@/lib/util/paginatedLoader.svelte";
	import { clearActiveFilters, defaultSort, store } from "@/store.svelte";
	import {
		type Media,
		type PaginationResponse,
		type WatchedStatus,
	} from "@/types";
	import { onDestroy, onMount, untrack } from "svelte";
	import FilterPopover from "@/lib/generic/FilterPopover.svelte";
	import MediaTypeFilter from "@/lib/search/MediaTypeFilter.svelte";
	import PageTitle from "@/lib/generic/PageTitle.svelte";
	import HomeStatusFilter from "./HomeStatusFilter.svelte";
	import HomeSortFilter from "./HomeSortFilter.svelte";

	let sortActive = $derived(
		store.activeSort?.length === 2 &&
			!!store.activeSort[1] &&
			JSON.stringify(store.activeSort) !== JSON.stringify(defaultSort),
	);

	// Up Next, the Library filter bar, and the content below it (posters or
	// grouped sections) all need to share the same effective width so their
	// left/right edges line up - PosterList caps itself at 1200px in list
	// view, while grouped view is capped at 1800px.
	let contentWidth = $derived(groupedView ? 1800 : 1200);

	// MediaTypeFilter (same component/style Discover uses) is single-select
	// and uses "show" for TV, while store.activeFilters.type is a multi-
	// select array using "tv" - map between the two so Home's Type button
	// looks and behaves exactly like Discover's.
	let activeMediaType = $derived.by(() => {
		const t = store.activeFilters.type[0];
		return t === "tv" ? "show" : t;
	});

	function setActiveMediaType(to: string | undefined) {
		const mapped = to === "show" ? "tv" : to;
		if (!mapped || store.activeFilters.type[0] === mapped) {
			store.activeFilters.type = [];
		} else {
			store.activeFilters.type = [mapped];
		}
		store.activeFilters = store.activeFilters;
	}

	const scroll = infScroll({ callback: onScrollToBottom });
	const dataLoader = paginatedLoader<Media, undefined>(load);

	// Sections shown in the "Group by status" view, in a sensible
	// most-relevant-first order (rather than alphabetical/enum order).
	const statusSections: { status: WatchedStatus; label: string }[] = [
		{ status: "WATCHING", label: "Watching" },
		{ status: "PLANNED", label: "Planned" },
		{ status: "HOLD", label: "On Hold" },
		{ status: "FINISHED", label: "Finished" },
		{ status: "DROPPED", label: "Dropped" },
	];

	let groupedView = $state(false);
	let groupedLoading = $state(false);
	let groupedData: Partial<
		Record<WatchedStatus, { items: Media[]; total: number }>
	> = $state({});

	// So actions taken in the Up Next row (mark watched, drop, rate, status
	// change) don't leave the Watching section/list below showing stale data.
	function refreshAfterUpNextAction() {
		if (groupedView) {
			loadGrouped();
		} else {
			dataLoader.reset();
			dataLoader.runFn();
		}
	}

	async function loadGrouped() {
		groupedLoading = true;
		try {
			const results = await Promise.all(
				statusSections.map((s) =>
					req.get<PaginationResponse<Media, undefined>>(`/watched`, {
						params: {
							...store.sortAndFiltersForQueryParams,
							status: s.status,
							page: 1,
							limit: 30,
						},
					}),
				),
			);
			const next: typeof groupedData = {};
			results.forEach((r, i) => {
				if (r.results && r.results.length > 0) {
					next[statusSections[i].status] = {
						items: r.results,
						total: r.totalResults,
					};
				}
			});
			groupedData = next;
		} finally {
			groupedLoading = false;
		}
	}

	let nextLoadParams: {
		page: number;
		[x: string]: unknown;
	} = $derived({
		page: dataLoader.state.page + 1,
		...store.sortAndFiltersForQueryParams,
	});

	async function load(signal: AbortSignal) {
		console.debug("load: loadParams:", nextLoadParams);
		if (nextLoadParams.page === dataLoader.state.page) {
			console.warn("load: Already on this page, not loading it again!");
			return;
		}
		const r = await req.get<PaginationResponse<Media, undefined>>(`/watched`, {
			params: nextLoadParams,
			signal,
		});
		scroll.dataLoaded();
		return r;
	}

	async function onScrollToBottom() {
		// No infinite scroll in the grouped view, each section loads its own
		// (small, capped) batch upfront instead.
		if (groupedView) {
			return;
		}
		// If an error is being shown, no more infinite scroll.
		if (dataLoader.state.reqLoadError) {
			return;
		}
		dataLoader.runFn();
	}

	// NOTE: This effect also handles initial load of data.
	$effect(() => {
		// When our sort/filter query params change,
		// load our list again.
		// Since it exists at load, this performs our
		// initial load of data too.
		if (!groupedView && store.sortAndFiltersForQueryParams) {
			untrack(() => {
				// We don't want to trigger another re-run of this
				// effect when state inside these funcs changes.
				dataLoader.reset();
				dataLoader.runFn();
			});
		}
	});

	$effect(() => {
		if (groupedView && store.sortAndFiltersForQueryParams) {
			untrack(() => {
				loadGrouped();
			});
		}
	});

	onMount(() => {
		const saved = localStorage.getItem("homeGroupedView");
		if (saved) {
			groupedView = saved === "true";
		}
	});

	$effect(() => {
		localStorage.setItem("homeGroupedView", String(groupedView));
	});

	onDestroy(() => {
		console.log("MAIN PAGE DESTROYED");
		scroll.destroy();
		dataLoader.abortReq("page destroyed");
	});
</script>

<svelte:head>
	<title>Watched List</title>
</svelte:head>

<!-- <span
	style="position: fixed; top: 80px; background-color: white; color: black; z-index: 60;"
>
	<b>listPage</b>: {dataLoader.state.page}
	listPageMax: {dataLoader.state.pageMax}
	listLoading: {dataLoader.state.reqLoading}
	<b>sort:</b>
	{JSON.stringify(store.activeSort)}
	<b>filter:</b>
	{JSON.stringify(store.activeFilters)}
	<b>queryp:</b>
	{JSON.stringify(store.sortAndFiltersForQueryParams)}
	paginatedLoader.state.meta: {JSON.stringify(dataLoader.state.meta)}
</span> -->

<div class="capped-content" style="max-width: {contentWidth}px">
	<UpNext onUpdated={refreshAfterUpNextAction} />
</div>

<div class="home-controls" style="max-width: {contentWidth}px">
	<PageTitle title="Library">
		<div class="pagetitle-mediatypefilter">
			<MediaTypeFilter
				active={activeMediaType}
				disabled={false}
				hidePeople
				onChange={(nowActive) => setActiveMediaType(nowActive)}
			/>
		</div>
		<div class="pagetitle-filters">
			<FilterPopover label="Status" active={store.activeFilters.status.length > 0}>
				<HomeStatusFilter />
			</FilterPopover>
			<FilterPopover label="Sort" active={sortActive}>
				<HomeSortFilter />
			</FilterPopover>
			{#if store.hasActiveFilters}
				<button
					type="button"
					class="plain reset-all"
					onclick={() => clearActiveFilters()}
				>
					✕ Reset
				</button>
			{/if}
			<button class="plain view-toggle" onclick={() => (groupedView = !groupedView)}>
				{groupedView ? "List view" : "Group by status"}
			</button>
		</div>
	</PageTitle>
</div>

{#if groupedView}
	<div class="capped-content grouped-content">
		{#if groupedLoading && Object.keys(groupedData).length === 0}
			<div style="margin-bottom: 60px;">
				<Spinner />
			</div>
		{:else if Object.keys(groupedData).length === 0}
			<div class="empty-list">
				<Icon i={store.hasActiveFilters ? "filter-circle" : "reel"} wh={80} />
				<h2 class="norm">Your list looks empty!</h2>
				<h4 class="norm">
					Try {`${store.hasActiveFilters ? "removing your active filters or" : ""}`}
					searching for something you would like to add.
				</h4>
				{#if !store.hasActiveFilters}
					<button onclick={() => goto(resolve("/import"))}>Import</button>
				{/if}
				{#if store.hasActiveFilters}
					<button onclick={() => clearActiveFilters()}>Clear Filters</button>
				{/if}
			</div>
		{:else}
			{#each statusSections as s (s.status)}
				{@const section = groupedData[s.status]}
				{#if section}
					<HorizontalList title={`${s.label} (${section.total})`}>
						{#each section.items as w, i (`${s.status}-${i}-${w.type}`)}
							<Poster
								bind:watched={section.items[i].watched}
								media={w}
								small
								showEpisodeBadge={s.status === "WATCHING"}
								onUpdated={loadGrouped}
							/>
						{/each}
					</HorizontalList>
				{/if}
			{/each}
		{/if}
	</div>
{:else}
	<PosterList>
		{#if dataLoader.state.data?.length > 0}
			{#each dataLoader.state.data as w, i (`${i}-${w.type}`)}
				{#if w}
					<Poster
						bind:watched={dataLoader.state.data[i].watched}
						media={w}
						fluidSize={true}
					/>
				{/if}
			{/each}
		{:else if !dataLoader.state.reqLoading && !dataLoader.state.reqLoadError}
			<div class="empty-list">
				<Icon i={store.hasActiveFilters ? "filter-circle" : "reel"} wh={80} />
				<h2 class="norm">Your list looks empty!</h2>
				<h4 class="norm">
					Try {`${store.hasActiveFilters ? "removing your active filters or" : ""}`}
					searching for something you would like to add.
				</h4>
				{#if !store.hasActiveFilters}
					<button onclick={() => goto(resolve("/import"))}>Import</button>
				{/if}
				{#if store.hasActiveFilters}
					<button onclick={() => clearActiveFilters()}>Clear Filters</button>
				{/if}
			</div>
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
					dataLoader.runFn();
				}}
			/>
		</div>
	{/if}
{/if}

<!-- TODO: A 'That's it' message when you reach bottom of your list? -->
<!-- {#if !dataLoader.state.reqLoadError && dataLoader.state.page === dataLoader.state.pageMax}
	<b>That's it!</b>
{/if} -->

<style lang="scss">
	.capped-content {
		max-width: 1800px;
		margin: 0 auto;
	}

	.home-controls {
		// max-width set inline (1200 in list view to match PosterList's own
		// cap, 1800 in grouped view to match .grouped-content) so the filter
		// row's edges always line up with whichever content is shown below.
		margin: 10px auto 0;

		// Match Up Next's heading (HorizontalList's h2) - PageTitle's own h2
		// has no explicit size, so it falls back to the browser default.
		:global(h2) {
			font-size: 30px;
			font-weight: bold;
		}

		// HorizontalList's h2 has margin-left:30px (vs PageTitle's own
		// margin:0 15px on its wrapper) - match it so "Library" lines up
		// exactly under "Up Next".
		:global(.results-filters-header) {
			margin-left: 30px;
		}
	}

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

	.reset-all {
		width: min-content;
		white-space: nowrap;
		font-size: 14px;
		color: $text-color-accent;

		&:hover {
			color: $text-color;
		}
	}

	button.view-toggle {
		width: min-content;
		white-space: nowrap;
		font-size: 14px;
	}

	.empty-list {
		display: flex;
		flex-flow: column;
		gap: 5px;
		align-items: center;
		max-width: 400px;

		h2 {
			margin-top: 10px;
		}

		h4 {
			font-weight: normal;
			text-align: center;
		}

		button {
			width: max-content;
			padding-left: 20px;
			padding-right: 20px;
			margin-top: 15px;
		}
	}
</style>

<script lang="ts">
	import { onDestroy, onMount } from "svelte";
	import { req } from "@/lib/util/api";
	import Poster from "@/lib/poster/Poster.svelte";
	import HorizontalList from "@/lib/HorizontalList.svelte";
	import PageTitle from "@/lib/generic/PageTitle.svelte";
	import FilterPopover from "@/lib/generic/FilterPopover.svelte";
	import Spinner from "@/lib/Spinner.svelte";
	import Error from "@/lib/Error.svelte";
	import ChartsProviderFilter from "./ChartsProviderFilter.svelte";
	import { type JustWatchProvider, type ProviderChart } from "@/types";
	import { store } from "@/store.svelte";

	function parseProviders(raw: string | undefined): string[] {
		if (!raw) return [];
		return raw.split("|").filter((s) => s.length > 0);
	}

	// store.userSettings is guaranteed populated before this page can mount
	// (the root layout awaits it before rendering routes), so this can
	// restore synchronously rather than via onMount - avoids a first-render
	// window where selectedProviders is empty and would overwrite the saved
	// value the moment the persist effect runs.
	let selectedProviders: string[] = $state(
		parseProviders(store.userSettings?.chartsProviders),
	);
	let providers: JustWatchProvider[] = $state([]);
	let charts: ProviderChart[] = $state([]);
	let loading = $state(false);
	let loadError: unknown = $state();

	async function loadProviders() {
		providers = await req.get<JustWatchProvider[]>(`/discover/charts/providers`);
	}

	function providerFor(shortName: string) {
		return providers.find((p) => p.shortName === shortName);
	}

	async function loadCharts() {
		if (selectedProviders.length === 0) {
			charts = [];
			return;
		}
		loading = true;
		loadError = undefined;
		try {
			charts = await req.get<ProviderChart[]>(`/discover/charts`, {
				params: { providers: selectedProviders.join("|") },
			});
			// Keep chart order matching the order providers were selected in.
			charts.sort(
				(a, b) =>
					selectedProviders.indexOf(a.provider) -
					selectedProviders.indexOf(b.provider),
			);
		} catch (err) {
			loadError = err;
		} finally {
			loading = false;
		}
	}

	// Persisted (per-user, server-side) provider selection, so it survives a
	// refresh - same idea as Discover's discoverFilters. pendingPersistJson
	// tracks a change that's debounced but not sent/saved yet, so it can be
	// flushed immediately on teardown (navigating away) instead of silently
	// dropped by the cancelled timeout.
	let lastPersistedProviders = store.userSettings?.chartsProviders || "";
	let persistProvidersTimeout: ReturnType<typeof setTimeout> | undefined;
	let pendingPersistJson: string | undefined;

	function sendPersistProviders(json: string) {
		pendingPersistJson = undefined;
		lastPersistedProviders = json;
		if (store.userSettings) store.userSettings.chartsProviders = json;
		req.post("/user/update", { chartsProviders: json }).catch((err) => {
			console.error("charts: Failed to persist providers", err);
		});
	}

	function persistProviders(json: string) {
		pendingPersistJson = json;
		clearTimeout(persistProvidersTimeout);
		persistProvidersTimeout = setTimeout(() => sendPersistProviders(json), 600);
	}

	onDestroy(() => {
		clearTimeout(persistProvidersTimeout);
		if (pendingPersistJson !== undefined) {
			sendPersistProviders(pendingPersistJson);
		}
	});

	// Provider names/logos (for the chart section headers) are needed
	// immediately on load, not just once the Streaming popover is opened -
	// it used to only fetch lazily inside that popover's {#await}, so a
	// fresh page load showed "Top 10" with no name until you clicked it.
	onMount(() => {
		loadProviders();
	});

	$effect(() => {
		const joined = selectedProviders.join("|");
		if (joined !== lastPersistedProviders) {
			persistProviders(joined);
		}
		loadCharts();
	});
</script>

<svelte:head>
	<title>Charts</title>
</svelte:head>

<div class="content">
	<div class="inner">
		<PageTitle title="Charts">
			<div class="pagetitle-filters">
				<FilterPopover label="Streaming" active={selectedProviders.length > 0}>
					<ChartsProviderFilter bind:active={selectedProviders} onChange={() => {}} />
				</FilterPopover>
			</div>
		</PageTitle>

		{#if selectedProviders.length === 0}
			<p class="hint">
				Select one or more streaming providers above to see what's currently
				popular on them.
			</p>
		{:else if loading && charts.length === 0}
			<Spinner />
		{:else if loadError}
			<Error
				pretty="Failed to load charts!"
				error={loadError}
				onRetry={loadCharts}
			/>
		{:else}
			{#each selectedProviders as providerShortName (providerShortName)}
				{@const chart = charts.find((c) => c.provider === providerShortName)}
				{@const provider = providerFor(providerShortName)}
				{#if chart && chart.items.length > 0}
					<HorizontalList
						title={provider?.clearName ?? "Top 10"}
						iconUrl={provider?.icon}
					>
						{#each chart.items as item, i (`${i}-${item.type}-${item.ids.tmdb}`)}
							<Poster
								media={item}
								bind:watched={chart.items[i].watched}
								small
								rank={item.rank}
								rankMovement={item.movement}
							/>
						{/each}
					</HorizontalList>
				{/if}
			{/each}
		{/if}
	</div>
</div>

<style lang="scss">
	.content {
		display: flex;
		width: 100%;
		justify-content: center;

		.inner {
			width: 100%;
			max-width: 1800px;
		}
	}

	.pagetitle-filters {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 14px;
		margin-left: auto;
	}

	.hint {
		margin: 40px 15px;
		color: $text-color-accent;
	}
</style>

<script lang="ts">
	import { onDestroy } from "svelte";
	import { req } from "@/lib/util/api";
	import Poster from "@/lib/poster/Poster.svelte";
	import HorizontalList from "@/lib/HorizontalList.svelte";
	import PageTitle from "@/lib/generic/PageTitle.svelte";
	import FilterPopover from "@/lib/generic/FilterPopover.svelte";
	import Spinner from "@/lib/Spinner.svelte";
	import Error from "@/lib/Error.svelte";
	import ProviderFilter from "../discover/ProviderFilter.svelte";
	import { SearchType, type ProviderChart } from "@/types";
	import { store } from "@/store.svelte";

	interface Provider {
		providerId: number;
		providerName: string;
		logoPath: string;
	}

	function parseProviders(raw: string | undefined): number[] {
		if (!raw) return [];
		return raw
			.split("|")
			.map((id) => Number(id))
			.filter((id) => !Number.isNaN(id));
	}

	// store.userSettings is guaranteed populated before this page can mount
	// (the root layout awaits it before rendering routes), so this can
	// restore synchronously rather than via onMount - avoids a first-render
	// window where selectedProviders is empty and would overwrite the saved
	// value the moment the persist effect runs.
	let selectedProviders: number[] = $state(
		parseProviders(store.userSettings?.chartsProviders),
	);
	let providers: Provider[] = $state([]);
	let charts: ProviderChart[] = $state([]);
	let loading = $state(false);
	let loadError: unknown = $state();

	async function loadProviders() {
		providers = await req.get<Provider[]>(`/content/watch-providers`, {
			params: { type: "movie" },
		});
	}

	function providerFor(id: number) {
		return providers.find((p) => p.providerId === id);
	}

	function logoUrl(logoPath: string) {
		return `https://image.tmdb.org/t/p/w45${logoPath}`;
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
					selectedProviders.indexOf(a.providerId) -
					selectedProviders.indexOf(b.providerId),
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
					{#await loadProviders() then}
						<ProviderFilter
							discoverType={SearchType.movie}
							bind:active={selectedProviders}
							onChange={() => {}}
						/>
					{/await}
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
			{#each selectedProviders as providerId (providerId)}
				{@const chart = charts.find((c) => c.providerId === providerId)}
				{@const provider = providerFor(providerId)}
				{#if chart && chart.items.length > 0}
					<HorizontalList
						title={provider?.providerName ?? "Top 10"}
						iconUrl={provider?.logoPath ? logoUrl(provider.logoPath) : undefined}
					>
						{#each chart.items as item, i (`${i}-${item.type}-${item.ids.tmdb}`)}
							<Poster
								media={item}
								bind:watched={chart.items[i].watched}
								small
								rank={i + 1}
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

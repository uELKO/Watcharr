<!-- Streaming-provider picker for the Charts page. Uses JustWatch's own
     provider list (their 3-letter short names, eg "nfx" for Netflix) since
     that's what the /discover/charts endpoint expects - not the same list
     or IDs as Discover's TMDB-based ProviderFilter. Same multi-select
     checklist pattern as GenreFilter/ProviderFilter. -->
<script lang="ts">
	import { req } from "@/lib/util/api";
	import type { JustWatchProvider } from "@/types";

	interface Props {
		/** Selected JustWatch provider short names. */
		active?: string[];
		onChange: () => void;
	}

	let { active = $bindable([]), onChange }: Props = $props();

	let providers: JustWatchProvider[] = $state([]);
	let loaded = $state(false);

	async function loadProviders() {
		if (loaded) return;
		providers = await req.get<JustWatchProvider[]>(`/discover/charts/providers`);
		providers.sort((a, b) => a.clearName.localeCompare(b.clearName));
		loaded = true;
	}

	function toggle(shortName: string) {
		if (active.includes(shortName)) {
			active = active.filter((p) => p !== shortName);
		} else {
			active = [...active, shortName];
		}
		onChange();
	}

	$effect(() => {
		loadProviders();
	});
</script>

<div class="provider-filter">
	{#if active.length > 0}
		<button
			type="button"
			class="plain reset"
			onclick={() => {
				active = [];
				onChange();
			}}
		>
			Reset
		</button>
	{/if}
	{#if providers.length > 0}
		<ul>
			{#each providers as p (p.shortName)}
				<li>
					<label>
						<input
							type="checkbox"
							checked={active.includes(p.shortName)}
							onchange={() => toggle(p.shortName)}
						/>
						{#if p.icon}
							<img src={p.icon} alt="" />
						{/if}
						{p.clearName}
					</label>
				</li>
			{/each}
		</ul>
	{/if}
</div>

<style lang="scss">
	.provider-filter {
		display: flex;
		flex-flow: column;
		gap: 4px;

		.reset {
			align-self: flex-end;
			width: min-content;
			white-space: nowrap;
			font-size: 12px;
			text-decoration: underline;
			color: $text-color;
		}

		ul {
			list-style: none;
			display: flex;
			flex-flow: column;
			gap: 2px;
			max-height: 220px;
			overflow-y: auto;
			margin: 0;
			padding: 2px;

			label {
				display: flex;
				align-items: center;
				gap: 8px;
				padding: 3px 2px;
				font-size: 13px;
				cursor: pointer;
				border-radius: 4px;

				&:hover {
					background-color: rgba(255, 255, 255, 0.06);
				}

				img {
					width: 20px;
					height: 20px;
					border-radius: 4px;
					flex: 0 0 auto;
				}

				input[type="checkbox"] {
					flex: 0 0 auto;
					width: 16px;
					height: 16px;
					padding: 0;
					margin: 0;
					border: 2px solid $text-color;
					border-radius: 3px;
					accent-color: $accent-color;
				}
			}
		}
	}
</style>

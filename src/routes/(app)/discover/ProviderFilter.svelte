<!-- Streaming-provider picker: a multi-select checklist (match any of these
     providers), same pattern as GenreFilter, but with each provider's own
     TMDB logo instead of just text. The parent gates access to this whole
     filter (disables its trigger button, with a tooltip explaining why) for
     wrong types and for Trending, so this component doesn't render its own
     disabled state - `disabled` here only stops it from fetching/rendering
     when the parent's trigger couldn't be opened anyway. -->
<script lang="ts">
	import { req } from "@/lib/util/api";
	import { SearchType } from "@/types";

	interface Props {
		discoverType: SearchType | undefined;
		disabled?: boolean;
		/** Selected TMDB watch provider ids. */
		active?: number[];
		onChange: () => void;
	}

	let {
		discoverType,
		disabled = false,
		active = $bindable([]),
		onChange,
	}: Props = $props();

	let wrongType = $derived(
		discoverType !== SearchType.movie && discoverType !== SearchType.show,
	);
	let effectivelyDisabled = $derived(disabled || wrongType);

	interface Provider {
		providerId: number;
		providerName: string;
		logoPath: string;
	}

	let providers: Provider[] = $state([]);
	let loadedFor: string | undefined = $state();

	async function loadProviders() {
		if (wrongType) {
			providers = [];
			loadedFor = undefined;
			return;
		}
		if (loadedFor === discoverType) {
			return;
		}
		const tmdbType = discoverType === SearchType.movie ? "movie" : "tv";
		providers = await req.get<Provider[]>(`/content/watch-providers`, {
			params: { type: tmdbType },
		});
		providers.sort((a, b) => a.providerName.localeCompare(b.providerName));
		loadedFor = discoverType;
		// Provider ids are shared across movie/tv, but only bother clearing
		// if the id doesn't exist in the freshly loaded list either.
		if (active.length > 0) {
			const known = new Set(providers.map((p) => p.providerId));
			const stillValid = active.filter((id) => known.has(id));
			if (stillValid.length !== active.length) {
				active = stillValid;
				onChange();
			}
		}
	}

	function toggle(id: number) {
		if (active.includes(id)) {
			active = active.filter((p) => p !== id);
		} else {
			active = [...active, id];
		}
		onChange();
	}

	function logoUrl(logoPath: string) {
		return `https://image.tmdb.org/t/p/w45${logoPath}`;
	}

	$effect(() => {
		if (discoverType) {
			loadProviders();
		}
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
	{#if !effectivelyDisabled && providers.length > 0}
		<ul>
			{#each providers as p (p.providerId)}
				<li>
					<label>
						<input
							type="checkbox"
							checked={active.includes(p.providerId)}
							onchange={() => toggle(p.providerId)}
						/>
						{#if p.logoPath}
							<img src={logoUrl(p.logoPath)} alt="" />
						{/if}
						{p.providerName}
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

				// Same override as GenreFilter: norm.scss's global `input`
				// rule otherwise blows the checkbox up.
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

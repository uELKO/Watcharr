<!-- Genre picker: a multi-select checklist (JustWatch-style "match any of
     these genres"), not a single-select dropdown. Only meaningful for
     movie/show discover types (TMDB genre lists are per movie/tv) - the
     parent gates access to this whole filter (disables its trigger button)
     when that's not the case, so this component doesn't need to render its
     own disabled state. Works for every filter mode including Trending: the
     server filters trending results itself using the genre_ids TMDB already
     includes on them, since /trending has no genre query param like
     /discover does.

     Each genre cycles through three states on click, JustWatch-style:
     neutral -> include (green check) -> exclude (red cross) -> neutral. -->
<script lang="ts">
	import { req } from "@/lib/util/api";
	import { SearchType } from "@/types";
	import Icon from "@/lib/Icon.svelte";

	interface Props {
		discoverType: SearchType | undefined;
		/** Included TMDB genre ids. */
		active?: number[];
		/** Excluded TMDB genre ids. */
		excluded?: number[];
		onChange: () => void;
	}

	let {
		discoverType,
		active = $bindable([]),
		excluded = $bindable([]),
		onChange,
	}: Props = $props();

	let wrongType = $derived(
		discoverType !== SearchType.movie && discoverType !== SearchType.show,
	);

	let genres: { id: number; name: string }[] = $state([]);
	let loadedFor: string | undefined = $state();

	async function loadGenres() {
		if (wrongType) {
			genres = [];
			loadedFor = undefined;
			return;
		}
		if (loadedFor === discoverType) {
			return;
		}
		const tmdbType = discoverType === SearchType.movie ? "movie" : "tv";
		genres = await req.get<{ id: number; name: string }[]>(`/content/genres`, {
			params: { type: tmdbType },
		});
		loadedFor = discoverType;
		// The genre list (and ids) differ between movie/show, so only drop
		// selected ids that don't actually exist in the freshly loaded list -
		// don't blindly clear everything just because we (re)fetched (this
		// component remounts every time its FilterPopover panel closes and
		// reopens, which would otherwise wipe the selection on every toggle
		// even when discoverType never changed).
		const known = new Set(genres.map((g) => g.id));
		let changed = false;
		if (active.length > 0) {
			const stillValid = active.filter((id) => known.has(id));
			if (stillValid.length !== active.length) {
				active = stillValid;
				changed = true;
			}
		}
		if (excluded.length > 0) {
			const stillValid = excluded.filter((id) => known.has(id));
			if (stillValid.length !== excluded.length) {
				excluded = stillValid;
				changed = true;
			}
		}
		if (changed) onChange();
	}

	function stateFor(id: number): "include" | "exclude" | "none" {
		if (active.includes(id)) return "include";
		if (excluded.includes(id)) return "exclude";
		return "none";
	}

	function cycle(id: number) {
		if (active.includes(id)) {
			active = active.filter((g) => g !== id);
			excluded = [...excluded, id];
		} else if (excluded.includes(id)) {
			excluded = excluded.filter((g) => g !== id);
		} else {
			active = [...active, id];
		}
		onChange();
	}

	$effect(() => {
		if (discoverType) {
			loadGenres();
		}
	});
</script>

<div class="genre-filter">
	{#if active.length > 0 || excluded.length > 0}
		<button
			type="button"
			class="plain reset"
			onclick={() => {
				active = [];
				excluded = [];
				onChange();
			}}
		>
			Reset
		</button>
	{/if}
	{#if genres.length > 0}
		<ul>
			{#each genres as g (g.id)}
				{@const state = stateFor(g.id)}
				<li>
					<button
						type="button"
						class="plain genre-row {state}"
						onclick={() => cycle(g.id)}
					>
						<span class="indicator">
							{#if state === "include"}
								<Icon i="check" wh={12} />
							{:else if state === "exclude"}
								<Icon i="close" wh={12} />
							{/if}
						</span>
						{g.name}
					</button>
				</li>
			{/each}
		</ul>
	{/if}
</div>

<style lang="scss">
	.genre-filter {
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
			max-height: 200px;
			overflow-y: auto;
			margin: 0;
			padding: 2px;

			.genre-row {
				display: flex;
				align-items: center;
				gap: 8px;
				width: 100%;
				padding: 3px 2px;
				font-size: 13px;
				text-align: start;
				border-radius: 4px;

				&:hover {
					background-color: rgba(255, 255, 255, 0.06);
				}

				.indicator {
					display: flex;
					align-items: center;
					justify-content: center;
					flex: 0 0 auto;
					width: 16px;
					height: 16px;
					border: 2px solid $text-color;
					border-radius: 3px;
				}

				&.include .indicator {
					border-color: $success;
					background-color: $success;
					color: $bg-color;
					fill: $bg-color;
				}

				&.exclude .indicator {
					border-color: $error;
					background-color: $error;
					color: $bg-color;
					fill: $bg-color;
				}
			}
		}
	}
</style>

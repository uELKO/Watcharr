<!-- Genre picker: a multi-select checklist (JustWatch-style "match any of
     these genres"), not a single-select dropdown. Only meaningful for
     movie/show discover types (TMDB genre lists are per movie/tv) - the
     parent gates access to this whole filter (disables its trigger button)
     when that's not the case, so this component doesn't need to render its
     own disabled state. Works for every filter mode including Trending: the
     server filters trending results itself using the genre_ids TMDB already
     includes on them, since /trending has no genre query param like
     /discover does. -->
<script lang="ts">
	import { req } from "@/lib/util/api";
	import { SearchType } from "@/types";

	interface Props {
		discoverType: SearchType | undefined;
		/** Selected TMDB genre ids. */
		active?: number[];
		onChange: () => void;
	}

	let { discoverType, active = $bindable([]), onChange }: Props = $props();

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
		if (active.length > 0) {
			const known = new Set(genres.map((g) => g.id));
			const stillValid = active.filter((id) => known.has(id));
			if (stillValid.length !== active.length) {
				active = stillValid;
				onChange();
			}
		}
	}

	function toggle(id: number) {
		if (active.includes(id)) {
			active = active.filter((g) => g !== id);
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
	{#if genres.length > 0}
		<ul>
			{#each genres as g (g.id)}
				<li>
					<label>
						<input
							type="checkbox"
							checked={active.includes(g.id)}
							onchange={() => toggle(g.id)}
						/>
						{g.name}
					</label>
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

				// Override norm.scss's global `input { width: 100%; padding: 7px 10px;
				// border: 2px solid black; }`, which otherwise blows the checkbox up
				// and shoves the label text away from it.
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

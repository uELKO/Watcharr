<!-- Sort field list for the Home filter bar. Each option cycles through
     off -> ascending -> descending -> off on repeated clicks, same behavior
     as the old nav-bar SortMenu, just restyled as a flat list inside a
     FilterPopover panel instead of a Menu dropdown. -->
<script lang="ts">
	import { store } from "@/store.svelte";

	const options: { id: string; value: string }[] = [
		{ id: "DATEADDED", value: "Date Added" },
		{ id: "LASTCHANGED", value: "Last Changed" },
		{ id: "LASTFIN", value: "Last Finished" },
		{ id: "RATING", value: "Rating" },
		{ id: "ALPHA", value: "Alphabetical" },
		{ id: "DATERELEASED", value: "Release Date" },
	];

	function sortClicked(id: string) {
		let mode = "UP";
		if (store.activeSort[0] == id) {
			if (store.activeSort[1] === "UP") {
				mode = "DOWN";
			} else if (store.activeSort[1] === "DOWN") {
				mode = "";
			}
		}
		if (!mode) {
			store.activeSort = [];
			return;
		}
		store.activeSort = [id, mode];
	}

	function directionClass(id: string): string {
		if (store.activeSort[0] !== id) return "";
		return store.activeSort[1] ? store.activeSort[1].toLowerCase() : "";
	}
</script>

<ul class="home-sort-filter">
	{#each options as o (o.id)}
		<li>
			<button
				type="button"
				class="plain {directionClass(o.id)}"
				onclick={() => sortClicked(o.id)}
			>
				{o.value}
			</button>
		</li>
	{/each}
</ul>

<style lang="scss">
	.home-sort-filter {
		list-style: none;
		display: flex;
		flex-flow: column;
		gap: 2px;
		margin: 0;
		padding: 2px;

		button {
			position: relative;
			width: 100%;
			text-align: start;
			padding: 3px 2px 3px 22px;
			font-size: 13px;
			border-radius: 4px;

			&:hover {
				background-color: rgba(255, 255, 255, 0.06);
			}

			&.down::before {
				content: "\2193";
			}

			&.up::before {
				content: "\2191";
			}

			&::before {
				position: absolute;
				top: 2px;
				left: 4px;
				font-family:
					system-ui,
					-apple-system,
					BlinkMacSystemFont;
				font-size: 15px;
			}
		}
	}
</style>

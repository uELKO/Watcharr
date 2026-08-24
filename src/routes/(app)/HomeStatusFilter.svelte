<!-- Watch status checklist for the Home filter bar. -->
<script lang="ts">
	import { store } from "@/store.svelte";

	const statuses: { id: string; value: string }[] = [
		{ id: "planned", value: "Planned" },
		{ id: "watching", value: store.serverFeatures?.games ? "Watching (Playing)" : "Watching" },
		{ id: "finished", value: store.serverFeatures?.games ? "Finished (Played)" : "Finished" },
		{ id: "hold", value: "On Hold" },
		{ id: "dropped", value: "Dropped" },
	];

	function toggle(f: string) {
		if (store.activeFilters.status?.includes(f)) {
			store.activeFilters.status = store.activeFilters.status.filter((a) => a !== f);
		} else {
			store.activeFilters.status = [...store.activeFilters.status, f];
		}
		store.activeFilters = store.activeFilters;
	}
</script>

<ul class="home-checklist-filter">
	{#each statuses as s (s.id)}
		<li>
			<label>
				<input
					type="checkbox"
					checked={store.activeFilters.status.includes(s.id)}
					onchange={() => toggle(s.id)}
				/>
				{s.value}
			</label>
		</li>
	{/each}
</ul>

<style lang="scss">
	.home-checklist-filter {
		list-style: none;
		display: flex;
		flex-flow: column;
		gap: 2px;
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
			white-space: nowrap;

			&:hover {
				background-color: rgba(255, 255, 255, 0.06);
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
</style>

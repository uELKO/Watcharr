<!-- Compact trigger button that reveals a panel of refinement filters
     (checkboxes/dropdowns), for grids that need more than one such
     filter without cluttering the title row (eg Discover's genre /
     provider / hide-watched filters). -->
<script lang="ts">
	import Icon from "../Icon.svelte";
	import { onMount } from "svelte";
	import type { Snippet } from "svelte";

	interface Props {
		/** Shows a small dot on the trigger when any filter inside is active. */
		active?: boolean;
		children: Snippet;
	}

	let { active = false, children }: Props = $props();

	let open = $state(false);
	let rootEl: HTMLDivElement | undefined = $state();

	function onWindowClick(e: MouseEvent) {
		if (open && rootEl && !rootEl.contains(e.target as Node)) {
			open = false;
		}
	}

	function onWindowKeyUp(e: KeyboardEvent) {
		if (open && e.key === "Escape") {
			open = false;
		}
	}

	onMount(() => {
		window.addEventListener("click", onWindowClick, true);
		window.addEventListener("keyup", onWindowKeyUp);
		return () => {
			window.removeEventListener("click", onWindowClick, true);
			window.removeEventListener("keyup", onWindowKeyUp);
		};
	});
</script>

<div class="filter-popover" bind:this={rootEl}>
	<button
		type="button"
		class="trigger"
		onclick={() => (open = !open)}
		aria-expanded={open}
	>
		<Icon i="filter" wh={16} />
		Filters
		{#if active}<span class="dot"></span>{/if}
	</button>
	{#if open}
		<div class="panel">
			{@render children()}
		</div>
	{/if}
</div>

<style lang="scss">
	.filter-popover {
		position: relative;
		width: min-content;

		.trigger {
			display: flex;
			align-items: center;
			gap: 5px;
			width: min-content;
			white-space: nowrap;
			position: relative;
		}

		.dot {
			position: absolute;
			top: 0px;
			right: -2px;
			width: 7px;
			height: 7px;
			border-radius: 50%;
			background: $success;
		}

		.panel {
			position: absolute;
			top: calc(100% + 5px);
			right: 0;
			z-index: 40;
			display: flex;
			flex-flow: column;
			gap: 10px;
			padding: 10px 12px;
			min-width: 220px;
			border-radius: 5px;
			background-color: $bg-color;
			border: 2px solid $text-color;
			box-shadow: 0px 2px 6px rgba(0, 0, 0, 0.4);
		}
	}
</style>

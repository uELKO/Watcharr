<!-- Compact trigger button that reveals a panel for one refinement filter
     category (eg Discover's Year / Genre / Rating / Provider), styled as a
     horizontal row of "Category ⌄" buttons (JustWatch-style) rather than
     one combined "Filters" button holding everything. -->
<script lang="ts">
	import Icon from "../Icon.svelte";
	import tooltip from "../actions/tooltip";
	import { onMount } from "svelte";
	import type { Snippet } from "svelte";

	interface Props {
		label: string;
		/** Shows a small dot on the trigger when a filter inside is active. */
		active?: boolean;
		disabled?: boolean;
		/** Tooltip shown on the trigger when disabled. */
		disabledReason?: string;
		children: Snippet;
	}

	let {
		label,
		active = false,
		disabled = false,
		disabledReason = "",
		children,
	}: Props = $props();

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

	$effect(() => {
		if (disabled) {
			open = false;
		}
	});

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
		class="trigger plain"
		{disabled}
		onclick={() => (open = !open)}
		aria-expanded={open}
		use:tooltip={{ text: disabledReason, pos: "bot", condition: disabled && !!disabledReason }}
	>
		{label}
		<Icon i="chevron" wh={14} facing={open ? "up" : "down"} />
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
			gap: 4px;
			width: min-content;
			white-space: nowrap;
			position: relative;
			font-size: 14px;

			&:disabled {
				opacity: 0.5;
				cursor: not-allowed;
			}
		}

		.dot {
			position: absolute;
			top: 0px;
			right: -8px;
			width: 7px;
			height: 7px;
			border-radius: 50%;
			background: $success;
		}

		.panel {
			position: absolute;
			top: calc(100% + 5px);
			left: 0;
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

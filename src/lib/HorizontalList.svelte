<script lang="ts">
	interface Props {
		title: string | undefined;
		/**
		 * Centers items instead of spreading them out (space-between) when
		 * there are few enough to not fill the row width - eg Up Next with
		 * just one or two shows in progress. Uses `safe center` so it falls
		 * back to the normal left-start/scrollable behavior once there are
		 * enough items to overflow, rather than centering an overflowing row
		 * and leaving it scrolled to a clipped middle instead of the start.
		 */
		center?: boolean;
		children?: import("svelte").Snippet;
	}

	let { title, center = false, children }: Props = $props();
</script>

<div>
	{#if title}
		<h2>{title}</h2>
	{/if}
	<ul class:center>
		{@render children?.()}
	</ul>
</div>

<style lang="scss">
	div {
		width: 100%;
		overflow-x: auto;
	}

	h2 {
		font-size: 30px;
		font-weight: bold;
		margin-left: 30px;
		position: sticky;
		left: 0;
	}

	ul {
		display: flex;
		flex-flow: row;
		gap: 10px;
		list-style: none;
		padding: 15px 30px;
		justify-content: space-between;

		&.center {
			justify-content: safe center;
		}
	}
</style>

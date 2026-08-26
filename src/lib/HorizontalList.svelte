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
		/** Small icon shown to the left of the title (eg a streaming provider's logo). */
		iconUrl?: string;
		children?: import("svelte").Snippet;
	}

	let { title, center = false, iconUrl, children }: Props = $props();
</script>

<div>
	{#if title}
		<h2>
			{#if iconUrl}
				<img src={iconUrl} alt="" />
			{/if}
			{title}
		</h2>
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
		display: flex;
		align-items: center;
		gap: 10px;
		font-size: 30px;
		font-weight: bold;
		margin-left: 30px;
		position: sticky;
		left: 0;

		img {
			width: 30px;
			height: 30px;
			border-radius: 6px;
		}
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

<!-- Small always-visible chart-position badge. Deliberately top-right, not
     top-left: PosterCommunityRating (the TMDB rating pill) also lives in
     the top-left corner, and while the two are meant to be mutually
     exclusive (this only renders while !posterActive, the rating pill only
     while active/hovered), putting them in different corners means a stuck-
     open card (posterActive staying true, eg after a click that never gets
     a mouseleave) can never visually overlap the two even if that
     assumption is ever violated. -->
<script lang="ts">
	interface Props {
		rank: number;
		/** Rank movement vs ~7 days ago, if known. */
		movement?: "up" | "down" | "same" | "";
	}

	let { rank, movement = "" }: Props = $props();
</script>

<div
	class="rank-badge"
	title={movement === "up"
		? "Up since last week"
		: movement === "down"
			? "Down since last week"
			: undefined}
>
	{rank}
	{#if movement === "up"}
		<span class="arrow up">▲</span>
	{:else if movement === "down"}
		<span class="arrow down">▼</span>
	{/if}
</div>

<style lang="scss">
	.rank-badge {
		position: absolute;
		top: 6px;
		right: 6px;
		z-index: 5;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 3px;
		min-width: 24px;
		height: 24px;
		padding: 0 6px;
		border-radius: 12px;
		background-color: rgba(0, 0, 0, 0.75);
		color: gold;
		font-weight: bold;
		font-size: 13px;

		.arrow {
			font-size: 9px;

			&.up {
				color: $success;
			}

			&.down {
				color: $error;
			}
		}
	}
</style>

<!-- Small pill showing the TMDB community rating (not the user's own rating,
     which PosterRating already covers), shown in the poster's hover overlay. -->
<script lang="ts">
	interface Props {
		/** Media.rating: vote_average*10 (or already 0-100), same encoding as Title.svelte. */
		rating?: number;
		ratingCount?: number;
	}

	let { rating, ratingCount }: Props = $props();

	const vote = $derived(
		rating ? Math.round((rating > 10 ? rating : rating * 10)) / 10 : 0,
	);
</script>

{#if vote > 0}
	<div
		class="community-rating"
		title={`TMDB Rating: ${vote} out of 10${ratingCount ? ` (${ratingCount} votes)` : ""}`}
	>
		<span class="star">★</span>
		{vote}
	</div>
{/if}

<style lang="scss">
	.community-rating {
		position: absolute;
		top: 0;
		left: 0;
		display: flex;
		align-items: center;
		gap: 3px;
		padding: 3px 8px;
		border-radius: 20px;
		background-color: rgba(0, 0, 0, 0.7);
		color: gold;
		font-size: 12px;
		font-weight: bold;

		.star {
			font-size: 13px;
		}
	}
</style>

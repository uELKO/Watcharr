<script lang="ts">
	import { dateValid } from "../util/date";

	interface Props {
		homepage?: string;
		title?: string;
		releaseDate?: Date;
		endDate?: Date;
		voteAverage?: number;
		voteCount?: number;
		/** From JustWatch's scoring data, when a match was found. */
		imdbScore?: number;
		imdbVotes?: number;
		tomatoMeter?: number;
		certifiedFresh?: boolean;
	}

	let {
		homepage,
		title,
		releaseDate,
		endDate,
		voteAverage,
		voteCount,
		imdbScore,
		imdbVotes,
		tomatoMeter,
		certifiedFresh,
	}: Props = $props();

	// if voteAvg bigger than 10, it is out of 100, so no need to * by 10
	const vote = $derived(
		voteAverage
			? Math.round(voteAverage > 10 ? voteAverage : voteAverage * 10) / 10
			: 0,
	);
	const titleSafe = $derived(title ? title : "Unknown Title");
	const releaseYear = $derived(
		dateValid(releaseDate) ? releaseDate.getFullYear() : undefined,
	);
	const endYear = $derived(
		dateValid(endDate) ? endDate.getFullYear() : undefined,
	);
</script>

<span class="title-container">
	<span class="title">
		{#if homepage}
			<a href={homepage} rel="external" target="_blank">{titleSafe}</a>
		{:else}
			<span class="t">{titleSafe}</span>
		{/if}
		{#if releaseYear}
			<span class="year">
				<!--First span ends on the line with the #if so there's no whitespace-->
				<span title={releaseDate?.toLocaleDateString()}>{releaseYear}</span
				>{#if endYear && endYear != releaseYear}
					<span title={endDate?.toLocaleDateString()}>-{endYear}</span>
				{/if}
			</span>
		{/if}
	</span>
	<span class="ratings">
		<span
			class="rating"
			title={`TMDB Rating: ${vote} out of 10 (based on ${voteCount ?? 0} votes)`}
		>
			<span>*</span>
			{vote}
		</span>
		{#if imdbScore || tomatoMeter}
			<span class="external-ratings">
				{#if imdbScore}
					<span
						class="external-rating imdb"
						title={`IMDb Rating: ${imdbScore} out of 10 (based on ${imdbVotes ?? 0} votes)`}
					>
						IMDb {imdbScore}
					</span>
				{/if}
				{#if tomatoMeter}
					<span
						class="external-rating rt"
						title={`Rotten Tomatoes: ${tomatoMeter}%${certifiedFresh ? " - Certified Fresh" : ""}`}
					>
						🍅 {tomatoMeter}%
					</span>
				{/if}
			</span>
		{/if}
	</span>
</span>

<style lang="scss">
	.title-container {
		display: flex;
		gap: 10px;

		.title {
			a,
			span.t {
				color: white;
				text-decoration: none;
				font-size: 30px;
				font-weight: bold;
				padding-right: 3px;
			}

			span.year {
				font-size: 20px;
				color: rgba($color: #fff, $alpha: 0.7);
			}
		}

		.ratings {
			display: flex;
			flex-flow: column;
			align-items: flex-end;
			gap: 4px;
			margin-left: auto;
		}

		.rating {
			display: flex;
			align-items: start;
			justify-content: center;
			gap: 5px;
			font-size: 22px;
			color: gold;
			font-weight: bolder;

			span {
				font-family: "Rampart One";
				-webkit-text-stroke: 1px gold;
				font-size: 40px;
				line-height: 0.7;
				margin-top: 7px;
			}
		}

		.external-ratings {
			display: flex;
			gap: 8px;
			font-size: 12px;
			font-weight: bold;

			.external-rating {
				white-space: nowrap;

				&.imdb {
					color: #f5c518;
				}

				&.rt {
					color: $text-color-accent;
				}
			}
		}
	}
</style>

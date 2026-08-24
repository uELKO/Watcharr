<!-- "Up Next" row: next unwatched episode per in-progress (Watching) show,
     with a quick "mark watched" action. Complements the homepage's
     "Watching" section (which shows where you left off) by showing what to
     watch next. Uses the exact same PosterEpisodeBadge/PosterProgressBar/
     PosterRating/PosterStatus components as Poster.svelte so both the look
     and the hover menu are identical, not just similar. The status picker's
     FINISHED option is repurposed to mean "mark this next episode watched"
     (the point of this row) rather than the show as a whole; every other
     option (PLANNED/WATCHING/HOLD/DROPPED/DELETE) updates the show exactly
     like it would from any other poster. -->
<script lang="ts">
	import { goto } from "$app/navigation";
	import { resolve } from "$app/paths";
	import { req } from "@/lib/util/api";
	import { notify } from "@/lib/util/notify";
	import HorizontalList from "@/lib/HorizontalList.svelte";
	import Icon from "@/lib/Icon.svelte";
	import PosterEpisodeBadge from "@/lib/poster/PosterEpisodeBadge.svelte";
	import PosterProgressBar from "@/lib/poster/PosterProgressBar.svelte";
	import PosterRating from "@/lib/poster/PosterRating.svelte";
	import PosterStatus from "@/lib/poster/PosterStatus.svelte";
	import { store } from "@/store.svelte";
	import type { WatchedStatus } from "@/types";

	interface UpNextItem {
		watchedId: number;
		tmdbId: number;
		showTitle: string;
		posterPath: string;
		seasonNumber: number;
		episodeNumber: number;
		seasonEpisodeCount: number;
		episodeName: string;
		stillPath: string;
		airDate: string;
		remainingEpisodes?: number;
		watchProgress?: number;
		rating?: number;
	}

	interface Props {
		/** Ran after any action here changes watched data, so the rest of the page (eg the Watching section) can refresh instead of going stale. */
		onUpdated?: () => void;
	}

	let { onUpdated = undefined }: Props = $props();

	let items = $state<UpNextItem[]>([]);
	let loading = $state(true);
	let busyId = $state<number | undefined>(undefined);
	let activeId = $state<number | undefined>(undefined);

	async function load() {
		try {
			// Pass the same sort as the watched list so Up Next matches its order.
			const qp = store.sortAndFiltersForQueryParams as {
				sort?: string;
				sortDir?: string;
			};
			items = await req.get<UpNextItem[]>("/watched/upnext", {
				params: { sort: qp?.sort, sortDir: qp?.sortDir },
			});
		} catch (err) {
			console.error("UpNext: load failed", err);
		} finally {
			loading = false;
		}
	}

	async function markWatched(item: UpNextItem) {
		busyId = item.watchedId;
		const nid = notify({
			text: `Marking ${item.showTitle} S${item.seasonNumber}E${item.episodeNumber}`,
			type: "loading",
		});
		try {
			await req.post("/watched/episode", {
				watchedId: item.watchedId,
				seasonNumber: item.seasonNumber,
				episodeNumber: item.episodeNumber,
				status: "FINISHED",
			});
			notify({ id: nid, text: "Marked as watched!", type: "success" });
			await load(); // refresh: the card advances to the next episode (or drops off)
			onUpdated?.();
		} catch (err) {
			console.error("UpNext: markWatched failed", err);
			notify({ id: nid, text: "Failed to mark as watched!", type: "error" });
		} finally {
			busyId = undefined;
		}
	}

	async function updateShowStatus(item: UpNextItem, status: WatchedStatus) {
		busyId = item.watchedId;
		const nid = notify({ text: `Updating ${item.showTitle}`, type: "loading" });
		try {
			await req.put(`/watched/${item.watchedId}`, { status });
			notify({ id: nid, text: "Updated!", type: "success" });
			await load(); // the show leaves the Up Next row once it's no longer Watching
			onUpdated?.();
		} catch (err) {
			console.error("UpNext: updateShowStatus failed", err);
			notify({ id: nid, text: "Failed to update!", type: "error" });
		} finally {
			busyId = undefined;
		}
	}

	async function removeShow(item: UpNextItem) {
		busyId = item.watchedId;
		const nid = notify({ text: `Removing ${item.showTitle}`, type: "loading" });
		try {
			await req.delete(`/watched/${item.watchedId}`);
			notify({ id: nid, text: "Removed!", type: "success" });
			await load();
			onUpdated?.();
		} catch (err) {
			console.error("UpNext: removeShow failed", err);
			notify({ id: nid, text: "Failed to remove!", type: "error" });
		} finally {
			busyId = undefined;
		}
	}

	async function updateRating(item: UpNextItem, rating: number) {
		busyId = item.watchedId;
		try {
			await req.put(`/watched/${item.watchedId}`, { rating });
			await load();
			onUpdated?.();
		} catch (err) {
			console.error("UpNext: updateRating failed", err);
			notify({ text: "Failed to update rating!", type: "error" });
		} finally {
			busyId = undefined;
		}
	}

	function handleStatusClick(item: UpNextItem, type: WatchedStatus | "DELETE") {
		if (type === "FINISHED") {
			markWatched(item);
		} else if (type === "DELETE") {
			removeShow(item);
		} else {
			updateShowStatus(item, type);
		}
	}

	function posterImg(item: UpNextItem) {
		if (!item.posterPath) return "";
		return `https://image.tmdb.org/t/p/w500${item.posterPath}`;
	}

	function episodeBadgeText(item: UpNextItem) {
		return `S${item.seasonNumber}E${item.episodeNumber}`;
	}

	// German format: "DD.MM." for the current year, "DD.MM.YYYY" otherwise.
	function formatGermanDate(d: Date) {
		const day = d.getDate().toString().padStart(2, "0");
		const month = (d.getMonth() + 1).toString().padStart(2, "0");
		if (d.getFullYear() === new Date().getFullYear()) {
			return `${day}.${month}.`;
		}
		return `${day}.${month}.${d.getFullYear()}`;
	}

	// Load on mount and whenever the sort changes, so Up Next stays in the
	// same order as the watched list.
	$effect(() => {
		void store.activeSort;
		load();
	});
</script>

{#if !loading && items.length > 0}
	<HorizontalList title="Up Next" center>
		{#each items as item (item.watchedId)}
			<li
				class={activeId === item.watchedId ? "active" : ""}
				onmouseenter={() => (activeId = item.watchedId)}
				onmouseleave={() => (activeId = undefined)}
				onclick={() => (activeId = item.watchedId)}
			>
				<div class="container">
					{#if posterImg(item)}
						<img src={posterImg(item)} alt="" />
					{:else}
						<div class="noimg"><Icon i="reel" wh={30} /></div>
					{/if}
					{#if item.airDate && activeId !== item.watchedId}
						{@const future = new Date(item.airDate) > new Date()}
						<div
							class="air-badge"
							class:future
							title={future ? "Airs on this date" : "TV broadcast date"}
						>
							{future ? "📺 " : ""}{formatGermanDate(new Date(item.airDate))}
						</div>
					{/if}
					{#if activeId !== item.watchedId}
						<PosterEpisodeBadge
							text={episodeBadgeText(item)}
							remaining={item.remainingEpisodes}
						/>
					{/if}
					{#if item.watchProgress !== undefined && activeId !== item.watchedId}
						<PosterProgressBar progress={item.watchProgress} />
					{/if}
					<div class="inner">
						<a
							onclick={(e) => {
								e.preventDefault();
								if (activeId === item.watchedId) {
									goto(resolve(`/tv/${item.tmdbId}`));
								}
							}}
							href={resolve(`/tv/${item.tmdbId}`)}
						>
							<h2>{item.showTitle}</h2>
							<span>
								{episodeBadgeText(item)}{item.seasonEpisodeCount
									? `/${item.seasonEpisodeCount}`
									: ""}{item.episodeName ? ` · ${item.episodeName}` : ""}
							</span>
						</a>
						<div class="buttons">
							<PosterRating
								rating={item.rating}
								handleStarClick={(r) => updateRating(item, r)}
								disableInteraction={busyId === item.watchedId}
							/>
							<PosterStatus
								status="WATCHING"
								handleStatusClick={(t) => handleStatusClick(item, t)}
								disableInteraction={busyId === item.watchedId}
								btnTooltip="Mark next episode watched"
							/>
						</div>
					</div>
				</div>
			</li>
		{/each}
	</HorizontalList>
{/if}

<style lang="scss">
	li {
		list-style: none;

		&.active {
			cursor: pointer;
		}

		&:not(.active) {
			.container .inner,
			.container .inner .buttons {
				pointer-events: none !important;
			}
		}
	}

	.container {
		display: flex;
		flex-flow: column;
		background-color: rgb(48, 45, 45);
		overflow: hidden;
		border-radius: 5px;
		width: 170px;
		min-width: 170px;
		position: relative;
		aspect-ratio: 170000/256367;
		transition: transform 150ms ease;

		img {
			width: 100%;
			height: 100%;
			object-fit: cover;
		}

		.noimg {
			width: 100%;
			height: 100%;
			display: flex;
			align-items: center;
			justify-content: center;
			background-color: rgba(0, 0, 0, 0.2);
		}

		// Same visual language as PosterStatusBadge/PosterEpisodeBadge.
		.air-badge {
			position: absolute;
			top: 6px;
			right: 6px;
			z-index: 5;
			padding: 3px 10px;
			border-radius: 20px;
			background-color: rgba(0, 0, 0, 0.7);
			color: $text-color-accent;
			font-size: 11px;
			font-weight: bold;
			letter-spacing: 0.3px;
			white-space: nowrap;

			&.future {
				color: $rating-color;
			}
		}

		.inner {
			position: absolute;
			opacity: 0;
			display: flex;
			flex-flow: column;
			top: 0;
			height: 100%;
			width: 100%;
			padding: 10px;
			background-color: transparent;
			transition: opacity 150ms cubic-bezier(0.19, 1, 0.22, 1);

			& > a {
				height: 100%;
				overflow: auto;
			}

			h2 {
				font-family:
					sans-serif,
					system-ui,
					-apple-system,
					BlinkMacSystemFont;
				font-size: 18px;
				color: white;
				word-wrap: break-word;
			}

			span {
				color: white;
				margin: 5px 0 5px 0;
				// Matches Poster's `.small` variant (used in the Watching row this
				// card sits alongside), not its default 9px.
				font-size: 11px;
				display: -webkit-box;
				-webkit-line-clamp: 5;
				-webkit-box-orient: vertical;
				hyphens: auto;
				overflow: hidden;
			}

			.buttons {
				display: flex;
				flex-flow: row;
				margin-top: auto;
				gap: 10px;
				height: 35px;
			}
		}
	}

	li.active .container {
		// Matches Poster's `.small` variant (1.3 is Poster's default/non-small
		// scale) - this card sits in the same compact row as the Watching
		// section's small posters, so it should grow the same amount.
		transform: scale(1.1);
		z-index: 99;

		img {
			filter: blur(4px) grayscale(80%);
			mix-blend-mode: multiply;
		}

		.inner {
			opacity: 1;
		}
	}
</style>

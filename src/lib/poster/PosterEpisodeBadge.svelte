<!-- Small always-visible badge showing the last/current watched
     season+episode for a show (eg "S2E5"), for grids like the homepage's
     "Watching" section where that's the most useful thing to see at a glance.
     Optionally paired with a "+N" pill for episodes still left to watch. -->
<script lang="ts">
	interface Props {
		text: string;
		remaining?: number;
	}

	let { text, remaining }: Props = $props();

	// Purely cosmetic, display-only: "S2E5" -> "S2 E5". Doesn't touch the
	// shared server-side format (server/feature/watched/watchedutil), which
	// other parts of the app (and its tests) rely on staying "S2E5".
	let displayText = $derived(text.replace(/^(S\d+)(E\d+)$/i, "$1 $2"));
</script>

<div class="episode-badge-ctr">
	<div class="episode-badge">{displayText}</div>
	{#if remaining}
		<div class="remaining-badge" title="{remaining} episode{remaining === 1 ? '' : 's'} left">
			+{remaining}
		</div>
	{/if}
</div>

<style lang="scss">
	.episode-badge-ctr {
		position: absolute;
		bottom: 6px;
		left: 6px;
		z-index: 5;
		display: flex;
		align-items: center;
		gap: 4px;
		pointer-events: none;
	}

	.episode-badge,
	.remaining-badge {
		padding: 3px 10px;
		border-radius: 20px;
		background-color: rgba(0, 0, 0, 0.7);
		color: white;
		font-size: 11px;
		font-weight: bold;
		letter-spacing: 0.3px;
	}

	.remaining-badge {
		padding: 3px 8px;
		color: $text-color-accent;
	}
</style>

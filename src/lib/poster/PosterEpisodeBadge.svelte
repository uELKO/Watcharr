<!-- Small always-visible badge showing the last/current watched
     season+episode for a show (eg "S2E5"), for grids like the homepage's
     "Watching" section where that's the most useful thing to see at a glance. -->
<script lang="ts">
	interface Props {
		text: string;
	}

	let { text }: Props = $props();

	// Purely cosmetic, display-only: "S2E5" -> "S2 E5". Doesn't touch the
	// shared server-side format (server/feature/watched/watchedutil), which
	// other parts of the app (and its tests) rely on staying "S2E5".
	let displayText = $derived(text.replace(/^(S\d+)(E\d+)$/i, "$1 $2"));
</script>

<div class="episode-badge">{displayText}</div>

<style lang="scss">
	.episode-badge {
		position: absolute;
		bottom: 6px;
		left: 6px;
		z-index: 5;
		padding: 3px 10px;
		border-radius: 20px;
		background-color: rgba(0, 0, 0, 0.7);
		color: white;
		font-size: 11px;
		font-weight: bold;
		letter-spacing: 0.3px;
		pointer-events: none;
	}
</style>

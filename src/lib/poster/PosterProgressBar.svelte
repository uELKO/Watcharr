<!-- Thin watch-progress bar along the bottom edge of a poster, for grids
     like the homepage's "Watching" section. -->
<script lang="ts">
	interface Props {
		/** 0-100 */
		progress: number;
	}

	let { progress }: Props = $props();

	let clamped = $derived(Math.max(0, Math.min(100, progress)));
</script>

<div class="progress-track" title="{clamped}% watched">
	<div class="progress-fill" style="width: {clamped}%"></div>
</div>

<style lang="scss">
	.progress-track {
		position: absolute;
		bottom: 0;
		left: 0;
		width: 100%;
		height: 4px;
		z-index: 5;
		background-color: rgba(0, 0, 0, 0.5);
		pointer-events: none;
	}

	.progress-fill {
		height: 100%;
		background-color: $success;
	}
</style>

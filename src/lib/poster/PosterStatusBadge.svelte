<!-- Small always-visible badge showing watched status, for grids where
     the full ExtraDetails hover overlay isn't appropriate (eg person filmography). -->
<script lang="ts">
	import type { WatchedStatus } from "@/types";
	import Icon from "../Icon.svelte";
	import tooltip from "../actions/tooltip";
	import { toUnderstandableStatus, watchedStatuses } from "../util/helpers";

	interface Props {
		status: WatchedStatus;
	}

	let { status }: Props = $props();
</script>

<div
	class="status-badge"
	use:tooltip={{ text: toUnderstandableStatus(status, false), pos: "bot" }}
>
	<Icon i={watchedStatuses[status]} wh={14} />
</div>

<style lang="scss">
	.status-badge {
		position: absolute;
		top: 6px;
		right: 6px;
		z-index: 5;
		display: flex;
		align-items: center;
		justify-content: center;
		width: 24px;
		height: 24px;
		border-radius: 50%;
		background-color: rgba(0, 0, 0, 0.65);
		color: white;
		fill: white;
		pointer-events: auto;
	}
</style>

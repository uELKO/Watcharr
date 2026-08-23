<!-- Flat single-select option list rendered directly inside a FilterPopover
     panel (eg Discover's Year / Rating filters). Deliberately not a nested
     <DropDown> - that renders its own trigger + popup, which stacked inside
     an already-open FilterPopover panel looked/behaved like two popups for
     one filter. -->
<script lang="ts">
	import type { DropDownItem } from "@/types";

	interface Props {
		options: DropDownItem[];
		active?: string | number;
		onChange: () => void;
	}

	let { options, active = $bindable(0), onChange }: Props = $props();
</script>

<ul class="single-select-filter">
	{#each options as o (o.id)}
		<li>
			<label>
				<input
					type="radio"
					name="single-select-filter"
					checked={active === o.id}
					onchange={() => {
						active = o.id;
						onChange();
					}}
				/>
				{o.value}
			</label>
		</li>
	{/each}
</ul>

<style lang="scss">
	.single-select-filter {
		list-style: none;
		display: flex;
		flex-flow: column;
		gap: 2px;
		max-height: 200px;
		overflow-y: auto;
		margin: 0;
		padding: 2px;

		label {
			display: flex;
			align-items: center;
			gap: 8px;
			padding: 3px 2px;
			font-size: 13px;
			cursor: pointer;
			border-radius: 4px;

			&:hover {
				background-color: rgba(255, 255, 255, 0.06);
			}

			// Override norm.scss's global `input { width: 100%; padding: 7px 10px;
			// border: 2px solid black; }`, which otherwise blows the radio up
			// and shoves the label text away from it.
			input[type="radio"] {
				flex: 0 0 auto;
				width: 16px;
				height: 16px;
				padding: 0;
				margin: 0;
				accent-color: $accent-color;
			}
		}
	}
</style>

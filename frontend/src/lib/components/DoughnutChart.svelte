<script lang="ts">
	import { onMount } from 'svelte';
	import { Chart, DoughnutController, ArcElement, Tooltip, Legend } from 'chart.js';

	Chart.register(DoughnutController, ArcElement, Tooltip, Legend);

	interface Props {
		labels: string[];
		data: number[];
		backgroundColor: string[];
		height?: number;
	}

	let { labels, data, backgroundColor, height = 300 }: Props = $props();

	let canvas: HTMLCanvasElement;
	let chart: Chart | null = null;

	onMount(() => {
		chart = new Chart(canvas, {
			type: 'doughnut',
			data: {
				labels,
				datasets: [{ data, backgroundColor }]
			},
			options: {
				responsive: true,
				maintainAspectRatio: false,
				plugins: { legend: { position: 'right' } }
			}
		});

		return () => {
			chart?.destroy();
		};
	});

	$effect(() => {
		if (chart) {
			chart.data.labels = labels;
			chart.data.datasets[0].data = data;
			chart.data.datasets[0].backgroundColor = backgroundColor;
			chart.update();
		}
	});
</script>

<div style="height: {height}px">
	<canvas bind:this={canvas}></canvas>
</div>

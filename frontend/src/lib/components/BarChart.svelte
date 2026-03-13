<script lang="ts">
	import { onMount } from 'svelte';
	import {
		Chart,
		BarController,
		BarElement,
		CategoryScale,
		LinearScale,
		Tooltip,
		Legend
	} from 'chart.js';

	Chart.register(BarController, BarElement, CategoryScale, LinearScale, Tooltip, Legend);

	interface Props {
		labels: string[];
		datasets: Array<{
			label: string;
			data: number[];
			backgroundColor: string;
		}>;
		height?: number;
	}

	let { labels, datasets, height = 300 }: Props = $props();

	let canvas: HTMLCanvasElement;
	let chart: Chart | null = null;

	onMount(() => {
		chart = new Chart(canvas, {
			type: 'bar',
			data: { labels, datasets },
			options: {
				responsive: true,
				maintainAspectRatio: false,
				plugins: { legend: { position: 'top' } },
				scales: {
					y: { beginAtZero: true }
				}
			}
		});

		return () => {
			chart?.destroy();
		};
	});

	// Update chart when data changes
	$effect(() => {
		if (chart) {
			chart.data.labels = labels;
			chart.data.datasets = datasets.map((ds, i) => ({
				...chart!.data.datasets[i],
				...ds
			}));
			chart.update();
		}
	});
</script>

<div style="height: {height}px">
	<canvas bind:this={canvas}></canvas>
</div>

<script lang="ts">
	import { getHelpTopics } from '$lib/data/help-content';
	import { helpDrawer, closeHelp } from '$lib/data/help-state.svelte';
	import { browser } from '$app/environment';

	let drawerEl: HTMLDivElement | undefined = $state();

	let topic = $derived(
		helpDrawer.topicId ? getHelpTopics(helpDrawer.taxConstants)[helpDrawer.topicId] : null
	);

	function handleKeydown(event: KeyboardEvent) {
		if (!helpDrawer.open) return;

		if (event.key === 'Escape') {
			event.preventDefault();
			closeHelp();
			return;
		}

		if (event.key === 'Tab' && drawerEl) {
			const focusable = drawerEl.querySelectorAll<HTMLElement>(
				'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
			);
			if (focusable.length === 0) return;

			const first = focusable[0];
			const last = focusable[focusable.length - 1];

			if (event.shiftKey) {
				if (document.activeElement === first) {
					event.preventDefault();
					last.focus();
				}
			} else {
				if (document.activeElement === last) {
					event.preventDefault();
					first.focus();
				}
			}
		}
	}

	$effect(() => {
		if (helpDrawer.open && browser && drawerEl) {
			const closeBtn = drawerEl.querySelector<HTMLElement>('button');
			closeBtn?.focus();
		}
	});
</script>

<svelte:window onkeydown={handleKeydown} />

{#if helpDrawer.open && topic}
	<!-- Mobile backdrop -->
	<div
		class="fixed inset-0 z-50 bg-black/60 lg:hidden"
		role="presentation"
		onclick={closeHelp}
	></div>

	<!-- Drawer panel -->
	<div
		bind:this={drawerEl}
		class="fixed inset-y-0 right-0 z-50 w-full max-w-[400px] bg-surface border-l border-border shadow-xl shadow-black/30 flex flex-col"
		role="dialog"
		aria-modal="true"
		aria-labelledby="help-drawer-title"
	>
		<!-- Header -->
		<div class="flex h-12 items-center justify-between border-b border-border px-4 shrink-0">
			<h2 id="help-drawer-title" class="text-sm font-semibold text-primary">{topic.title}</h2>
			<button
				type="button"
				class="rounded-md p-1.5 text-secondary hover:bg-hover hover:text-primary transition-colors"
				onclick={closeHelp}
				aria-label="Zavřít nápovědu"
			>
				<svg
					class="h-4 w-4"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="1.5"
				>
					<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</button>
		</div>

		<!-- Body -->
		<div class="flex-1 overflow-y-auto px-5 py-5 space-y-6">
			<!-- Simple explanation -->
			<section>
				<h3 class="text-xs font-medium uppercase tracking-wider text-muted mb-3">Jednoduše</h3>
				<div class="bg-elevated rounded-lg p-4 space-y-2">
					{#each topic.simple.split('\n\n') as paragraph}
						<p class="text-sm text-primary leading-relaxed">{paragraph}</p>
					{/each}
				</div>
			</section>

			<hr class="border-border" />

			<!-- Legal framework -->
			<section>
				<h3 class="text-xs font-medium uppercase tracking-wider text-muted mb-3">Právní rámec</h3>
				<div class="space-y-2">
					{#each topic.legal.split('\n\n') as paragraph}
						<p class="text-sm text-secondary leading-relaxed">{paragraph}</p>
					{/each}
				</div>
			</section>
		</div>
	</div>
{/if}

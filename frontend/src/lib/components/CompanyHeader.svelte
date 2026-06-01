<script lang="ts">
	import { goto } from '$app/navigation';
	import { currentCompany } from '$lib/stores/currentCompany.svelte';

	let open = $state(false);

	function toggle() {
		open = !open;
	}

	function close() {
		open = false;
	}

	function pick(id: number) {
		currentCompany.select(id);
		close();
	}

	function manage() {
		close();
		goto('/companies');
	}

	function add() {
		close();
		goto('/companies/new');
	}
</script>

<div class="relative inline-block">
	<button
		type="button"
		class="flex items-center gap-2 rounded-md border border-border bg-surface px-3 py-1.5 text-sm font-medium text-primary hover:bg-hover transition-colors"
		onclick={toggle}
		aria-haspopup="listbox"
		aria-expanded={open}
	>
		<span class="truncate max-w-[200px]">{currentCompany.current?.name ?? 'Žádná firma'}</span>
		<svg
			class="h-4 w-4 shrink-0 text-tertiary"
			fill="none"
			viewBox="0 0 24 24"
			stroke="currentColor"
			stroke-width="1.5"
		>
			<path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
		</svg>
	</button>

	{#if open}
		<div role="presentation" class="fixed inset-0 z-40" onclick={close}></div>
		<ul
			class="absolute right-0 z-50 mt-1 w-64 rounded-md border border-border bg-surface py-1 shadow-lg"
			role="listbox"
		>
			{#each currentCompany.companies as c (c.id)}
				<li>
					<button
						type="button"
						class="flex w-full items-center justify-between px-3 py-1.5 text-sm text-primary hover:bg-hover transition-colors"
						onclick={() => pick(c.id)}
						role="option"
						aria-selected={currentCompany.current?.id === c.id}
					>
						<span class="truncate">{c.name}</span>
						{#if currentCompany.current?.id === c.id}
							<span aria-label="aktivní" class="text-accent shrink-0 ml-2">✓</span>
						{/if}
					</button>
				</li>
			{/each}
			<li class="my-1 border-t border-border" role="separator"></li>
			<li>
				<button
					type="button"
					class="block w-full px-3 py-1.5 text-left text-sm text-secondary hover:bg-hover hover:text-primary transition-colors"
					onclick={manage}
				>
					Spravovat firmy →
				</button>
			</li>
			<li>
				<button
					type="button"
					class="block w-full px-3 py-1.5 text-left text-sm text-accent hover:bg-hover transition-colors"
					onclick={add}
				>
					+ Přidat firmu
				</button>
			</li>
		</ul>
	{/if}
</div>

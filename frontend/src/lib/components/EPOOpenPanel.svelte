<script lang="ts">
	import type { EPOFinding } from '$lib/api/client';
	import type { EPOOpenState } from '$lib/utils/epo-open.svelte';

	interface Props {
		epo: EPOOpenState;
		hasXml: boolean;
		/** Label of the page's own "mark as filed" button; omit when the document is already filed. */
		markFiledLabel?: string;
	}

	let { epo, hasXml, markFiledLabel }: Props = $props();

	const findingLabels: Record<string, string> = {
		K: 'Kritická chyba',
		S: 'Chyba struktury',
		E: 'Chyba EPO',
		P: 'Varování',
		N: 'Upozornění',
		I: 'Informace'
	};
	const warningTypes = ['P', 'N', 'I'];

	function isWarning(finding: EPOFinding): boolean {
		return warningTypes.includes(finding.type.trim().toUpperCase());
	}

	function findingLabel(finding: EPOFinding): string {
		return findingLabels[finding.type.trim().toUpperCase()] ?? 'Chyba';
	}

	function findingPlace(finding: EPOFinding): string {
		const parts = [finding.section, finding.field];
		if (finding.row) parts.push(`řádek ${finding.row}`);
		return parts.filter(Boolean).join(', ');
	}
</script>

<div class="mt-3 space-y-3" data-testid="epo-panel">
	{#if !hasXml}
		<p class="text-xs text-muted">Nejdřív vygenerujte XML.</p>
	{:else}
		<p class="text-xs text-muted">
			Formulář se otevře v prohlížeči. Přihlaste se v EPO a podání odešlete. Odkaz platí 30 minut.
		</p>
	{/if}

	{#if epo.loading}
		<div role="status" class="text-sm text-secondary">
			<span class="sr-only">Nacitani...</span>
			<span aria-hidden="true">Kontroluji podání v EPO...</span>
		</div>
	{/if}

	{#if epo.errorMessage}
		<div
			role="alert"
			class="rounded-lg border border-danger/20 bg-danger-bg p-4 text-sm text-danger"
		>
			<p>{epo.errorMessage}</p>
			{#if epo.blockingFindings.length > 0}
				<ul class="mt-2 list-disc space-y-1 pl-5">
					{#each epo.blockingFindings as finding, i (i)}
						<li class={isWarning(finding) ? 'text-warning' : 'text-danger'}>
							<strong>{findingLabel(finding)}:</strong>
							{finding.text}
							{#if findingPlace(finding)}
								<span class="text-xs">({findingPlace(finding)})</span>
							{/if}
						</li>
					{/each}
				</ul>
			{/if}
		</div>
	{/if}

	{#if epo.opened || epo.fallbackUrl}
		<div class="rounded-lg border border-border bg-surface p-4 text-sm text-secondary">
			{#if epo.opened}
				<p>Formulář byl otevřen v prohlížeči.</p>
			{:else if epo.fallbackUrl}
				<p>
					Formulář se nepodařilo otevřít v nové kartě.
					<a
						href={epo.fallbackUrl}
						target="_blank"
						rel="noopener noreferrer"
						class="text-accent underline"
					>
						Otevřít formulář v EPO
					</a>
				</p>
			{/if}

			{#if epo.warnings.length > 0}
				<p class="mt-3 font-medium text-primary">EPO upozorňuje:</p>
				<ul class="mt-1 list-disc space-y-1 pl-5">
					{#each epo.warnings as finding, i (i)}
						<li class="text-warning">
							<strong>{findingLabel(finding)}:</strong>
							{finding.text}
							{#if findingPlace(finding)}
								<span class="text-xs">({findingPlace(finding)})</span>
							{/if}
						</li>
					{/each}
				</ul>
			{/if}

			{#if markFiledLabel}
				<p class="mt-3">
					Až podání v EPO odešlete, označte ho tlačítkem „{markFiledLabel}“ nahoře.
				</p>
			{/if}
		</div>
	{/if}
</div>

<script lang="ts">
	import { onMount } from 'svelte';
	import { epoApi, epoFindingsFromError, type EPOFinding, type EPOKind } from '$lib/api/client';
	import { detectDesktop } from '$lib/utils/download';
	import { prepareExternalOpen } from '$lib/utils/external';
	import Button from '$lib/ui/Button.svelte';

	interface Props {
		kind: EPOKind;
		id: number;
		hasXml: boolean;
		/** Disables the button while another action of the page is running. */
		disabled?: boolean;
		/** Offered after a successful open; omit when the document is already filed. */
		onMarkFiled?: () => void;
	}

	let { kind, id, hasXml, disabled = false, onMarkFiled }: Props = $props();

	const findingLabels: Record<string, string> = {
		K: 'Kritická chyba',
		S: 'Chyba struktury',
		E: 'Chyba EPO',
		P: 'Varování',
		N: 'Upozornění',
		I: 'Informace'
	};
	const warningTypes = ['P', 'N', 'I'];

	let desktop = $state(false);
	let desktopChecked = $state(false);
	let loading = $state(false);
	let errorMessage = $state<string | null>(null);
	let blockingFindings = $state<EPOFinding[]>([]);
	let warnings = $state<EPOFinding[]>([]);
	let opened = $state(false);
	let fallbackUrl = $state<string | null>(null);

	onMount(async () => {
		desktop = await detectDesktop();
		desktopChecked = true;
	});

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

	async function handleOpen() {
		// Must run before the first await so the new tab keeps the click gesture.
		const pending = prepareExternalOpen(desktop);

		loading = true;
		errorMessage = null;
		blockingFindings = [];
		warnings = [];
		opened = false;
		fallbackUrl = null;
		try {
			const result = await epoApi.openForm(kind, id);
			warnings = result.warnings;
			if (await pending.open(result.url)) {
				opened = true;
			} else {
				fallbackUrl = result.url;
			}
		} catch (e) {
			pending.cancel();
			errorMessage = e instanceof Error ? e.message : 'Nepodařilo se otevřít formulář v EPO';
			blockingFindings = epoFindingsFromError(e);
		} finally {
			loading = false;
		}
	}
</script>

<div class="contents">
	<Button
		variant="secondary"
		onclick={handleOpen}
		disabled={!hasXml || disabled || loading || !desktopChecked}
		title={hasXml ? undefined : 'Nejdřív vygenerujte XML'}
	>
		{loading ? 'Otevírám...' : 'Otevřít v EPO'}
	</Button>

	<div class="w-full space-y-3" data-testid="epo-panel">
		{#if !hasXml}
			<p class="text-xs text-muted">Nejdřív vygenerujte XML.</p>
		{:else}
			<p class="text-xs text-muted">
				Formulář se otevře v prohlížeči. Přihlaste se v EPO a podání odešlete. Odkaz platí 30 minut.
			</p>
		{/if}

		{#if loading}
			<div role="status" class="text-sm text-secondary">
				<span class="sr-only">Nacitani...</span>
				<span aria-hidden="true">Kontroluji podání v EPO...</span>
			</div>
		{/if}

		{#if errorMessage}
			<div
				role="alert"
				class="rounded-lg border border-danger/20 bg-danger-bg p-4 text-sm text-danger"
			>
				<p>{errorMessage}</p>
				{#if blockingFindings.length > 0}
					<ul class="mt-2 list-disc space-y-1 pl-5">
						{#each blockingFindings as finding, i (i)}
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

		{#if opened || fallbackUrl}
			<div class="rounded-lg border border-border bg-surface p-4 text-sm text-secondary">
				{#if opened}
					<p>Formulář byl otevřen v prohlížeči.</p>
				{:else if fallbackUrl}
					<p>
						Prohlížeč zablokoval otevření nové karty.
						<a
							href={fallbackUrl}
							target="_blank"
							rel="noopener noreferrer"
							class="text-accent underline"
						>
							Otevřít formulář v EPO
						</a>
					</p>
				{/if}

				{#if warnings.length > 0}
					<p class="mt-3 font-medium text-primary">EPO upozorňuje:</p>
					<ul class="mt-1 list-disc space-y-1 pl-5">
						{#each warnings as finding, i (i)}
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

				{#if onMarkFiled}
					<div class="mt-3 flex flex-wrap items-center gap-3">
						<span>Až podání v EPO odešlete, označte ho jako podané.</span>
						<Button variant="success" size="sm" onclick={onMarkFiled}>Označit jako podané</Button>
					</div>
				{/if}
			</div>
		{/if}
	</div>
</div>

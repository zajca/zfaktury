<script lang="ts">
	import { onMount } from 'svelte';
	import {
		ApiError,
		epoFindingsFromError,
		filingApi,
		isdsCredentialsApi,
		type EPOFinding,
		type FilingKind,
		type FilingPreview,
		type FilingSubmission,
		type ISDSCredentialsOverview,
		type ISDSEnvironment
	} from '$lib/api/client';
	import {
		environmentLabels,
		environmentOrder,
		filingErrorMessage,
		findingIsWarning,
		findingLabel,
		findingPlace,
		formatFileSize
	} from '$lib/utils/filing';
	import { toastSuccess } from '$lib/data/toast-state.svelte';
	import Button from '$lib/ui/Button.svelte';
	import ISDSEnvironmentBadge from './ISDSEnvironmentBadge.svelte';

	interface Props {
		kind: FilingKind;
		id: number;
		hasXml: boolean;
		/** True when the document is already marked as filed. */
		filed: boolean;
		/** Disables the button while another action of the page is running. */
		disabled?: boolean;
		/** Called after a send was accepted, so the history can reload. */
		onsent?: (submission: FilingSubmission) => void;
		/** Called after a send attempt failed, because the history may hold a new unsettled row. */
		onattempt?: () => void;
	}

	let { kind, id, hasXml, filed, disabled = false, onsent, onattempt }: Props = $props();

	let credentials = $state<ISDSCredentialsOverview | null>(null);
	let credentialsLoading = $state(true);
	let credentialsError = $state<string | null>(null);

	let chosenEnvironment = $state<ISDSEnvironment | null>(null);
	let previewLoading = $state(false);
	let previewError = $state<string | null>(null);
	let previewFindings = $state<EPOFinding[]>([]);

	let dialogOpen = $state(false);
	let preview = $state<FilingPreview | null>(null);
	let acceptWarnings = $state(false);
	let sending = $state(false);
	let sendError = $state<string | null>(null);
	// A failed send always needs a new preview: the token is single use.
	let tokenSpent = $state(false);
	let dialogEl: HTMLDivElement | undefined = $state();

	const available = $derived(
		environmentOrder.filter((env) => credentials?.[env]?.configured === true)
	);
	const environment = $derived<ISDSEnvironment | null>(
		available.length === 1 ? available[0] : chosenEnvironment
	);
	const noLogin = $derived(!credentialsLoading && !credentialsError && available.length === 0);

	const disabledReason = $derived.by(() => {
		if (filed) return 'Podání je už označeno jako podané.';
		if (!hasXml) return 'Nejdřív vygenerujte XML.';
		if (credentialsError) return 'Přihlašovací údaje datové schránky se nepodařilo načíst.';
		if (noLogin) return 'Nejsou uloženy přihlašovací údaje datové schránky.';
		if (available.length > 1 && environment === null) return 'Vyberte prostředí.';
		return null;
	});

	const canConfirm = $derived(
		preview !== null &&
			!sending &&
			!tokenSpent &&
			(!preview.requiresAcceptWarnings || acceptWarnings)
	);

	onMount(() => {
		loadCredentials();
	});

	async function loadCredentials() {
		credentialsLoading = true;
		credentialsError = null;
		try {
			credentials = await isdsCredentialsApi.get();
		} catch (e) {
			credentialsError = filingErrorMessage(e);
		} finally {
			credentialsLoading = false;
		}
	}

	async function requestPreview(): Promise<boolean> {
		if (previewLoading || sending || environment === null) return false;
		previewLoading = true;
		previewError = null;
		previewFindings = [];
		try {
			preview = await filingApi.preview(kind, id, environment);
			acceptWarnings = false;
			sendError = null;
			tokenSpent = false;
			dialogOpen = true;
			return true;
		} catch (e) {
			previewError = filingErrorMessage(e);
			previewFindings = e instanceof ApiError ? epoFindingsFromError(e) : [];
			return false;
		} finally {
			previewLoading = false;
		}
	}

	async function refreshPreview() {
		const ok = await requestPreview();
		// A failed preview is shown below the button, together with its findings.
		if (!ok) {
			dialogOpen = false;
			preview = null;
			sendError = null;
			tokenSpent = false;
		}
	}

	function closeDialog() {
		if (sending) return;
		dialogOpen = false;
		preview = null;
		sendError = null;
		tokenSpent = false;
	}

	async function confirmSend() {
		// The flag is set before the first await so a second click cannot send again.
		if (sending || !preview || !canConfirm) return;
		sending = true;
		sendError = null;
		const current = preview;
		try {
			const submission = await filingApi.send(kind, id, {
				environment: current.environment,
				confirmToken: current.confirmToken,
				acceptWarnings: current.requiresAcceptWarnings && acceptWarnings
			});
			dialogOpen = false;
			preview = null;
			toastSuccess('Podání bylo odesláno datovou schránkou');
			onsent?.(submission);
		} catch (e) {
			sendError = filingErrorMessage(e);
			tokenSpent = true;
			onattempt?.();
		} finally {
			sending = false;
		}
	}

	function handleKeydown(event: KeyboardEvent) {
		if (!dialogOpen) return;
		if (event.key === 'Escape') {
			event.preventDefault();
			closeDialog();
			return;
		}
		if (event.key === 'Tab' && dialogEl) {
			const focusable = dialogEl.querySelectorAll<HTMLElement>(
				'button:not([disabled]), [href], input:not([disabled]), select, textarea, [tabindex]:not([tabindex="-1"])'
			);
			if (focusable.length === 0) return;
			const first = focusable[0];
			const last = focusable[focusable.length - 1];
			if (event.shiftKey && document.activeElement === first) {
				event.preventDefault();
				last.focus();
			} else if (!event.shiftKey && document.activeElement === last) {
				event.preventDefault();
				first.focus();
			}
		}
	}

	$effect(() => {
		if (dialogOpen && dialogEl) {
			dialogEl.querySelector<HTMLElement>('button')?.focus();
		}
	});
</script>

<svelte:window onkeydown={handleKeydown} />

{#if available.length > 1}
	<fieldset
		class="w-full rounded-lg border border-border p-3 sm:w-auto"
		data-testid="isds-env-pick"
	>
		<legend class="px-1 text-xs font-medium text-secondary">Prostředí datové schránky</legend>
		<div class="flex flex-wrap gap-4">
			{#each available as env (env)}
				<label class="flex items-center gap-2 text-sm text-primary">
					<input
						type="radio"
						name="isds-environment-{kind}-{id}"
						value={env}
						checked={chosenEnvironment === env}
						onchange={() => (chosenEnvironment = env)}
					/>
					{environmentLabels[env]}
				</label>
			{/each}
		</div>
	</fieldset>
{:else if available.length === 1}
	<span class="inline-flex items-center gap-2 text-xs text-secondary" data-testid="isds-env-single">
		Prostředí:
		<ISDSEnvironmentBadge environment={available[0]} />
	</span>
{/if}

<Button
	variant="primary"
	onclick={requestPreview}
	disabled={disabled || credentialsLoading || previewLoading || disabledReason !== null}
	title={disabledReason ?? undefined}
>
	{previewLoading ? 'Kontroluji podání...' : 'Odeslat datovou schránkou'}
</Button>

<div class="w-full space-y-2 empty:hidden" data-testid="isds-send-notes">
	{#if credentialsLoading}
		<div role="status" class="text-xs text-secondary">
			<span class="sr-only">Nacitani...</span>
			<span aria-hidden="true">Zjišťuji přihlášení k datové schránce...</span>
		</div>
	{/if}

	{#if noLogin}
		<p class="text-xs text-muted" data-testid="isds-no-login">
			Pro odeslání datovou schránkou nejdřív uložte přihlašovací údaje v
			<a href="/settings/firma" class="text-accent underline">nastavení</a>.
		</p>
	{/if}

	{#if filed}
		<p class="text-xs text-muted">Podání je už označeno jako podané, další odeslání není možné.</p>
	{/if}

	{#if credentialsError}
		<div
			role="alert"
			class="rounded-lg border border-danger/20 bg-danger-bg p-3 text-sm text-danger"
		>
			{credentialsError}
		</div>
	{/if}

	{#if previewError && !dialogOpen}
		<div
			role="alert"
			class="rounded-lg border border-danger/20 bg-danger-bg p-4 text-sm text-danger"
		>
			<p>{previewError}</p>
			{#if previewFindings.length > 0}
				<ul class="mt-2 list-disc space-y-1 pl-5">
					{#each previewFindings as finding, i (i)}
						<li class={findingIsWarning(finding) ? 'text-warning' : 'text-danger'}>
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
</div>

{#if dialogOpen && preview}
	<div class="fixed inset-0 z-50 bg-black/60" role="presentation" onclick={closeDialog}></div>
	<div
		bind:this={dialogEl}
		class="fixed inset-0 z-50 flex items-center justify-center p-4"
		role="dialog"
		aria-modal="true"
		aria-labelledby="isds-send-title"
	>
		<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
		<div
			class="max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-xl border border-border bg-surface p-6 shadow-xl"
			onclick={(e) => e.stopPropagation()}
		>
			<div class="flex flex-wrap items-center justify-between gap-2">
				<h2 id="isds-send-title" class="text-lg font-semibold text-primary">
					Odeslat datovou schránkou
				</h2>
				<ISDSEnvironmentBadge environment={preview.environment} prominent />
			</div>

			<dl class="mt-4 space-y-3 text-sm" data-testid="isds-preview">
				<div>
					<dt class="text-xs font-medium uppercase tracking-wider text-muted">Příjemce</dt>
					<dd class="mt-0.5 text-primary">
						<span class="font-medium">{preview.recipientName}</span>
						<span class="font-mono text-secondary">({preview.recipientBoxId})</span>
					</dd>
				</div>
				<div>
					<dt class="text-xs font-medium uppercase tracking-wider text-muted">Soubor</dt>
					<dd class="mt-0.5 text-primary">
						{preview.fileName}
						<span class="text-secondary">({formatFileSize(preview.sizeBytes)})</span>
					</dd>
				</div>
				<div>
					<dt class="text-xs font-medium uppercase tracking-wider text-muted">Předmět zprávy</dt>
					<dd class="mt-0.5 text-primary">{preview.annotation}</dd>
				</div>
			</dl>

			{#if preview.findings.length > 0}
				<div
					class="mt-4 rounded-lg border border-warning/30 bg-warning-bg p-3 text-sm"
					data-testid="isds-findings"
				>
					<p class="font-medium text-primary">Kontrola podání upozorňuje:</p>
					<ul class="mt-1 list-disc space-y-1 pl-5">
						{#each preview.findings as finding, i (i)}
							<li class={findingIsWarning(finding) ? 'text-warning' : 'text-danger'}>
								<strong>{findingLabel(finding)}:</strong>
								{finding.text}
								{#if findingPlace(finding)}
									<span class="text-xs">({findingPlace(finding)})</span>
								{/if}
							</li>
						{/each}
					</ul>
				</div>
			{/if}

			{#if preview.requiresAcceptWarnings}
				<label class="mt-4 flex items-start gap-2 text-sm text-primary">
					<input type="checkbox" class="mt-0.5" bind:checked={acceptWarnings} disabled={sending} />
					<span>Odeslat i přesto</span>
				</label>
			{/if}

			{#if sendError}
				<div
					role="alert"
					class="mt-4 rounded-lg border border-danger/20 bg-danger-bg p-3 text-sm text-danger"
				>
					{sendError}
				</div>
			{/if}

			<div class="mt-6 flex flex-wrap justify-end gap-3">
				<Button variant="secondary" onclick={closeDialog} disabled={sending}>
					{tokenSpent ? 'Zavřít' : 'Zrušit'}
				</Button>
				{#if tokenSpent}
					<Button variant="primary" onclick={refreshPreview} disabled={previewLoading || sending}>
						{previewLoading ? 'Kontroluji podání...' : 'Zobrazit náhled znovu'}
					</Button>
				{:else}
					<Button variant="primary" onclick={confirmSend} disabled={!canConfirm}>
						{sending ? 'Odesílám...' : 'Odeslat'}
					</Button>
				{/if}
			</div>
		</div>
	</div>
{/if}

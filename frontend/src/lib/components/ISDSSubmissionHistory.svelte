<script lang="ts">
	import { onMount } from 'svelte';
	import { filingApi, type FilingKind, type FilingSubmission } from '$lib/api/client';
	import { downloadFile } from '$lib/utils/download';
	import { formatDateTime } from '$lib/utils/date';
	import {
		canRefresh,
		filingErrorMessage,
		needsResolution,
		submissionStatusLabels,
		submissionStatusVariant
	} from '$lib/utils/filing';
	import { toastSuccess } from '$lib/data/toast-state.svelte';
	import Badge from '$lib/ui/Badge.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';
	import ConfirmDialog from '$lib/ui/ConfirmDialog.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import Input from '$lib/ui/Input.svelte';
	import ISDSEnvironmentBadge from './ISDSEnvironmentBadge.svelte';

	interface Props {
		kind: FilingKind;
		id: number;
		/** Called when a refresh shows the document is now filed, so the page reloads it. */
		ondocumentfiled?: () => void;
	}

	let { kind, id, ondocumentfiled }: Props = $props();

	let submissions = $state<FilingSubmission[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);

	// Per-row state, keyed by submission id.
	let rowBusy = $state<Record<number, 'refresh' | 'receipt' | 'resolve' | undefined>>({});
	let rowError = $state<Record<number, string | null>>({});
	let attachOpen = $state<Record<number, boolean>>({});
	let dmIdInput = $state<Record<number, string>>({});
	let notSentTarget = $state<number | null>(null);
	let showNotSentConfirm = $state(false);

	onMount(() => {
		reload();
	});

	/** Reloads the list; the page calls it after a send. */
	export async function reload(): Promise<void> {
		loading = true;
		loadError = null;
		try {
			submissions = await filingApi.submissions(kind, id);
		} catch (e) {
			loadError = filingErrorMessage(e);
		} finally {
			loading = false;
		}
	}

	/** Re-reads the list without the loading state; a failed read keeps the rows shown. */
	async function syncList(): Promise<void> {
		try {
			submissions = await filingApi.submissions(kind, id);
		} catch {
			// The row's own error stays visible.
		}
	}

	function replaceRow(updated: FilingSubmission) {
		submissions = submissions.map((s) => (s.id === updated.id ? updated : s));
	}

	async function handleRefresh(sub: FilingSubmission) {
		if (rowBusy[sub.id]) return;
		rowBusy[sub.id] = 'refresh';
		rowError[sub.id] = null;
		try {
			const result = await filingApi.refresh(sub.id);
			replaceRow(result.submission);
			if (result.documentFiled) {
				toastSuccess('Podání bylo doručeno a je označeno jako podané');
				ondocumentfiled?.();
			}
		} catch (e) {
			rowError[sub.id] = filingErrorMessage(e);
			// A refused refresh can leave the stored row changed, so show what the server has now.
			await syncList();
		} finally {
			rowBusy[sub.id] = undefined;
		}
	}

	async function handleReceipt(sub: FilingSubmission) {
		if (rowBusy[sub.id]) return;
		rowBusy[sub.id] = 'receipt';
		rowError[sub.id] = null;
		try {
			await downloadFile(filingApi.receiptUrl(sub.id), `dorucenka-${sub.dmId || sub.id}.zfo`);
		} catch {
			rowError[sub.id] = 'Doručenku se nepodařilo stáhnout.';
		} finally {
			rowBusy[sub.id] = undefined;
		}
	}

	function toggleAttach(sub: FilingSubmission) {
		attachOpen[sub.id] = !attachOpen[sub.id];
		rowError[sub.id] = null;
	}

	function onDmIdInput(sub: FilingSubmission, event: Event) {
		const input = event.target as HTMLInputElement;
		const digits = input.value.replace(/\D/g, '');
		input.value = digits;
		dmIdInput[sub.id] = digits;
	}

	async function resolve(
		sub: FilingSubmission,
		resolution: Parameters<typeof filingApi.resolve>[1]
	) {
		if (rowBusy[sub.id]) return;
		rowBusy[sub.id] = 'resolve';
		rowError[sub.id] = null;
		try {
			const updated = await filingApi.resolve(sub.id, resolution);
			replaceRow(updated);
			attachOpen[sub.id] = false;
			dmIdInput[sub.id] = '';
			toastSuccess(
				resolution.action === 'attach'
					? 'Číslo zprávy bylo uloženo'
					: 'Odeslání bylo označeno jako neodeslané'
			);
		} catch (e) {
			rowError[sub.id] = filingErrorMessage(e);
		} finally {
			rowBusy[sub.id] = undefined;
		}
	}

	function handleAttach(sub: FilingSubmission) {
		const dmId = (dmIdInput[sub.id] ?? '').trim();
		if (!/^\d+$/.test(dmId)) {
			rowError[sub.id] = 'Zadejte číslo zprávy, tvoří ho jen číslice.';
			return;
		}
		void resolve(sub, { action: 'attach', dmId });
	}

	function askNotSent(sub: FilingSubmission) {
		notSentTarget = sub.id;
		showNotSentConfirm = true;
	}

	function confirmNotSent() {
		const target = submissions.find((s) => s.id === notSentTarget);
		showNotSentConfirm = false;
		notSentTarget = null;
		if (target) void resolve(target, { action: 'notSent' });
	}
</script>

<section class="mt-6" aria-labelledby="isds-history-heading" data-testid="isds-history">
	<Card>
		<h2 id="isds-history-heading" class="text-base font-semibold text-primary">
			Odeslání datovou schránkou
		</h2>

		<ErrorAlert error={loadError} class="mt-3" />

		{#if loading}
			<div role="status" class="mt-3 text-sm text-secondary">
				<span class="sr-only">Nacitani...</span>
				<span aria-hidden="true">Načítám historii odeslání...</span>
			</div>
		{:else if submissions.length === 0 && !loadError}
			<p class="mt-3 text-sm text-muted">Toto podání zatím nebylo odesláno datovou schránkou.</p>
		{:else}
			<ul class="mt-3 divide-y divide-border-subtle">
				{#each submissions as sub (sub.id)}
					<li class="py-4" data-testid="isds-submission-{sub.id}">
						<div class="flex flex-wrap items-center gap-2">
							<span class="text-sm font-medium text-primary">{formatDateTime(sub.createdAt)}</span>
							<ISDSEnvironmentBadge environment={sub.environment} />
							<Badge variant={submissionStatusVariant[sub.status]}>
								{submissionStatusLabels[sub.status]}
							</Badge>
						</div>

						<dl class="mt-2 grid gap-x-6 gap-y-1 text-sm sm:grid-cols-2">
							<div>
								<dt class="inline text-muted">Příjemce:</dt>
								<dd class="inline text-primary">
									{sub.recipientName}
									<span class="font-mono text-secondary">({sub.recipientBoxId})</span>
								</dd>
							</div>
							<div>
								<dt class="inline text-muted">Číslo zprávy:</dt>
								<dd class="inline font-mono text-primary">{sub.dmId || '-'}</dd>
							</div>
							<div>
								<dt class="inline text-muted">Doručeno:</dt>
								<dd class="inline text-primary">{formatDateTime(sub.deliveredAt)}</dd>
							</div>
						</dl>

						{#if needsResolution(sub.status)}
							<div
								class="mt-3 rounded-lg border border-warning/30 bg-warning-bg p-3 text-sm text-primary"
							>
								{#if sub.status === 'sending'}
									<p>
										Zpráva se právě odesílá, nebo se odesílání přerušilo. Pokud se stav během
										několika minut nezmění, zkontrolujte odeslané zprávy v datové schránce. Podání
										se znovu neodešle, dokud výsledek nepotvrdíte.
									</p>
								{:else}
									<p>
										Výsledek tohoto odeslání není znám, zpráva mohla odejít, ale nemusela.
										Zkontrolujte odeslané zprávy v datové schránce. Podání se znovu neodešle, dokud
										výsledek nepotvrdíte.
									</p>
								{/if}
								<div class="mt-3 flex flex-wrap gap-2">
									<Button
										variant="secondary"
										size="sm"
										onclick={() => toggleAttach(sub)}
										disabled={rowBusy[sub.id] !== undefined}
									>
										Zprávu jsem našel
									</Button>
									<Button
										variant="secondary"
										size="sm"
										onclick={() => askNotSent(sub)}
										disabled={rowBusy[sub.id] !== undefined}
									>
										Nic neodešlo
									</Button>
								</div>
								{#if attachOpen[sub.id]}
									<form
										class="mt-3 flex flex-wrap items-end gap-2"
										onsubmit={(e) => {
											e.preventDefault();
											handleAttach(sub);
										}}
									>
										<div class="min-w-48 flex-1">
											<label
												for="isds-dmid-{sub.id}"
												class="block text-xs font-medium text-secondary"
											>
												Číslo nalezené zprávy
											</label>
											<Input
												id="isds-dmid-{sub.id}"
												type="text"
												inputmode="numeric"
												autocomplete="off"
												value={dmIdInput[sub.id] ?? ''}
												oninput={(e) => onDmIdInput(sub, e)}
												class="mt-1"
											/>
										</div>
										<Button
											type="submit"
											variant="primary"
											size="sm"
											disabled={rowBusy[sub.id] !== undefined || !(dmIdInput[sub.id] ?? '')}
										>
											{rowBusy[sub.id] === 'resolve' ? 'Ukládám...' : 'Uložit číslo zprávy'}
										</Button>
									</form>
								{/if}
							</div>
						{/if}

						<div class="mt-3 flex flex-wrap gap-2">
							{#if canRefresh(sub.status)}
								<Button
									variant="secondary"
									size="sm"
									onclick={() => handleRefresh(sub)}
									disabled={rowBusy[sub.id] !== undefined}
								>
									{rowBusy[sub.id] === 'refresh' ? 'Zjišťuji...' : 'Zjistit stav'}
								</Button>
							{/if}
							{#if sub.status === 'delivered' && sub.hasReceipt}
								<Button
									variant="secondary"
									size="sm"
									onclick={() => handleReceipt(sub)}
									disabled={rowBusy[sub.id] !== undefined}
								>
									{rowBusy[sub.id] === 'receipt' ? 'Stahuji...' : 'Stáhnout doručenku'}
								</Button>
							{/if}
						</div>

						<ErrorAlert error={rowError[sub.id] ?? null} class="mt-3" />
					</li>
				{/each}
			</ul>
		{/if}
	</Card>
</section>

<ConfirmDialog
	bind:open={showNotSentConfirm}
	title="Potvrdit, že nic neodešlo"
	message="Potvrďte jen tehdy, když jste v odeslaných zprávách datové schránky zprávu nenašli. Podání pak bude možné odeslat znovu."
	confirmLabel="Nic neodešlo"
	variant="warning"
	onconfirm={confirmNotSent}
	oncancel={() => {
		showNotSentConfirm = false;
		notSentTarget = null;
	}}
/>

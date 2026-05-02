<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import {
		employmentApi,
		incomeTaxApi,
		type EmploymentCertificate,
		type EmploymentDocumentKind
	} from '$lib/api/client';
	import { toastError, toastSuccess } from '$lib/data/toast-state.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import EmploymentCertificateEditor from '$lib/components/EmploymentCertificateEditor.svelte';

	const yearParam = page.url.searchParams.get('year');
	let selectedYear = $state(yearParam ? Number(yearParam) : new Date().getFullYear() - 1);

	let loading = $state(true);
	let error = $state<string | null>(null);
	let uploading = $state(false);
	let working = $state(false);

	let certificates = $state<EmploymentCertificate[]>([]);

	// Editor state
	let editorOpen = $state(false);
	let editorDraft = $state<Partial<EmploymentCertificate>>({});

	// File pickers
	let advanceFileInput = $state<HTMLInputElement | null>(null);
	let withholdingFileInput = $state<HTMLInputElement | null>(null);

	async function loadData() {
		loading = true;
		error = null;
		try {
			const certs = await employmentApi.listCertificates(selectedYear);
			certificates = certs ?? [];
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se načíst Potvrzení.';
		} finally {
			loading = false;
		}
	}

	let mounted = false;
	onMount(() => {
		loadData();
		mounted = true;
	});

	$effect(() => {
		selectedYear;
		if (!mounted) return;
		loadData();
	});

	function pickFile(kind: EmploymentDocumentKind) {
		const input = kind === 'advance' ? advanceFileInput : withholdingFileInput;
		if (!input) return;
		input.value = '';
		input.click();
	}

	async function uploadAndExtract(file: File, kind: EmploymentDocumentKind) {
		uploading = true;
		try {
			const doc = await employmentApi.uploadDocument(selectedYear, kind, file);
			const draft = await employmentApi.extractDocument(doc.id);
			editorDraft = draft;
			editorOpen = true;
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se zpracovat soubor.');
		} finally {
			uploading = false;
		}
	}

	function onAdvanceFilePicked(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		if (file) uploadAndExtract(file, 'advance');
	}

	function onWithholdingFilePicked(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		if (file) uploadAndExtract(file, 'withholding');
	}

	function openManual(certType: 'advance' | 'withholding') {
		editorDraft = {
			year: selectedYear,
			certificate_type: certType,
			contract_type: 'dpc'
		};
		editorOpen = true;
	}

	function editCertificate(cert: EmploymentCertificate) {
		editorDraft = { ...cert };
		editorOpen = true;
	}

	async function deleteCertificate(id: number) {
		if (!confirm('Opravdu chcete toto Potvrzení smazat?')) return;
		working = true;
		try {
			await employmentApi.deleteCertificate(id);
			await loadData();
			await maybeRecomputeReturn();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Mazání selhalo.');
		} finally {
			working = false;
		}
	}

	async function confirmCertificate(cert: EmploymentCertificate) {
		working = true;
		try {
			await employmentApi.confirmCertificate(cert.id);
			toastSuccess('Potvrzení potvrzeno.');
			await loadData();
			await maybeRecomputeReturn();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Potvrzení selhalo.');
		} finally {
			working = false;
		}
	}

	async function maybeRecomputeReturn() {
		try {
			const returns = await incomeTaxApi.list(selectedYear);
			if (returns && returns.length > 0) {
				await incomeTaxApi.recalculate(returns[0].id);
			}
		} catch {
			// Recompute is best-effort -- a missing return is not an error.
		}
	}

	async function onEditorSaved(_cert: EmploymentCertificate, confirmed: boolean) {
		editorOpen = false;
		toastSuccess(confirmed ? 'Potvrzení uloženo a potvrzeno.' : 'Potvrzení uloženo jako koncept.');
		await loadData();
		if (confirmed) await maybeRecomputeReturn();
	}

	function fmt(n: number): string {
		return n.toLocaleString('cs-CZ') + ' Kč';
	}

	function contractLabel(t: string): string {
		switch (t) {
			case 'dpc':
				return 'DPČ';
			case 'dpp':
				return 'DPP';
			case 'hpp':
				return 'HPP';
			default:
				return 'Jiné';
		}
	}

	function certTypeLabel(t: string): string {
		return t === 'withholding' ? 'Srážkové (vzor 12)' : 'Zálohové (vzor 33)';
	}

	function statusLabel(status: string): string {
		return status === 'confirmed' ? 'Potvrzeno' : 'Koncept';
	}

	function statusColor(status: string): string {
		return status === 'confirmed' ? 'bg-success-bg text-success' : 'bg-elevated text-secondary';
	}
</script>

<svelte:head>
	<title>Závislá činnost (§6) {selectedYear} - ZFaktury</title>
</svelte:head>

<input
	bind:this={advanceFileInput}
	type="file"
	class="sr-only"
	accept=".pdf,.png,.jpg,.jpeg,.webp"
	onchange={onAdvanceFilePicked}
	tabindex="-1"
	aria-hidden="true"
/>
<input
	bind:this={withholdingFileInput}
	type="file"
	class="sr-only"
	accept=".pdf,.png,.jpg,.jpeg,.webp"
	onchange={onWithholdingFilePicked}
	tabindex="-1"
	aria-hidden="true"
/>

<div class="mx-auto max-w-6xl">
	<a href="/tax" class="text-sm text-secondary hover:text-primary">&larr; Zpět na daně</a>

	<h1 class="mt-2 text-xl font-semibold text-primary">
		Závislá činnost (§6) -- {selectedYear}
		<HelpTip topic="zavisla-cinnost-s6" />
	</h1>

	<div class="mt-4 flex items-center gap-3">
		<Button
			variant="ghost"
			size="sm"
			onclick={() => {
				selectedYear--;
			}}
			aria-label="Předchozí rok"
		>
			&larr;
		</Button>
		<span class="min-w-[4rem] text-center text-xl font-semibold text-primary tabular-nums">
			{selectedYear}
		</span>
		<Button
			variant="ghost"
			size="sm"
			onclick={() => {
				selectedYear++;
			}}
			aria-label="Následující rok"
		>
			&rarr;
		</Button>
	</div>

	<ErrorAlert {error} class="mt-4" />

	<!-- Upload tiles -->
	<div class="mt-6 grid grid-cols-1 gap-4 md:grid-cols-3">
		<Card>
			<h2 class="text-sm font-semibold text-primary">
				Nahrát zálohové Potvrzení (vzor 33) <HelpTip topic="potvrzeni-zalohove" />
			</h2>
			<p class="mt-2 text-xs text-tertiary">
				PDF nebo fotka. Aplikace přečte částky a otevře editor s předvyplněnými poli.
			</p>
			<div class="mt-3" data-testid="upload-advance-wrapper">
				<Button
					variant="primary"
					size="sm"
					onclick={() => pickFile('advance')}
					disabled={uploading}
				>
					{uploading ? 'Zpracovávám...' : 'Vybrat soubor'}
				</Button>
			</div>
		</Card>

		<Card>
			<h2 class="text-sm font-semibold text-primary">
				Nahrát srážkové Potvrzení (vzor 12) <HelpTip topic="potvrzeni-srazkove" />
			</h2>
			<p class="mt-2 text-xs text-tertiary">
				Pro DPP/DPČ se srážkovou daní. Sami se rozhodnete, jestli zahrnete do přiznání.
			</p>
			<div class="mt-3" data-testid="upload-withholding-wrapper">
				<Button
					variant="primary"
					size="sm"
					onclick={() => pickFile('withholding')}
					disabled={uploading}
				>
					{uploading ? 'Zpracovávám...' : 'Vybrat soubor'}
				</Button>
			</div>
		</Card>

		<Card>
			<h2 class="text-sm font-semibold text-primary">Zadat ručně</h2>
			<p class="mt-2 text-xs text-tertiary">
				Pokud Potvrzení nemáte v digitální podobě nebo chcete data zadat sami.
			</p>
			<div class="mt-3 flex gap-2" data-testid="manual-buttons-wrapper">
				<Button variant="secondary" size="sm" onclick={() => openManual('advance')}>
					Zálohové
				</Button>
				<Button variant="secondary" size="sm" onclick={() => openManual('withholding')}>
					Srážkové
				</Button>
			</div>
		</Card>
	</div>

	<!--
		OCR provider availability advisory (RFC-016 Migration Plan #2).
		Backend currently does not expose a public flag for OCR config presence,
		so we surface a graceful UX hint instead of hard-gating the upload tiles.
	-->
	<div
		class="mt-3 flex items-start gap-2 rounded border border-border bg-info-bg p-3 text-xs text-info"
		role="status"
		data-testid="ocr-config-advisory"
	>
		<svg
			class="mt-0.5 h-4 w-4 shrink-0"
			fill="none"
			viewBox="0 0 24 24"
			stroke="currentColor"
			stroke-width="1.5"
			aria-hidden="true"
		>
			<path
				stroke-linecap="round"
				stroke-linejoin="round"
				d="M11.25 11.25l.041-.02a.75.75 0 011.063.852l-.708 2.836a.75.75 0 001.063.853l.041-.021M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9-3.75h.008v.008H12V8.25z"
			/>
		</svg>
		<div>
			OCR vyžaduje aktivní AI poskytovatel v <code class="font-mono">config.toml [ocr]</code>. Pokud
			OCR nefunguje nebo není nakonfigurované, použijte tlačítko „Zadat ručně".
		</div>
	</div>

	<!-- Certificates table -->
	<div class="mt-8">
		<h2 class="text-base font-semibold text-primary">Potvrzení za rok {selectedYear}</h2>
		{#if loading}
			<LoadingSpinner class="mt-6 p-12" />
		{:else if certificates.length === 0}
			<div
				class="mt-4 rounded-lg border border-border bg-surface p-8 text-center text-sm text-tertiary"
				data-testid="empty-state"
			>
				<p>Zatím žádná Potvrzení.</p>
				<p class="mt-1">Nahrajte PDF nebo zadejte data ručně.</p>
			</div>
		{:else}
			<div class="mt-4 overflow-hidden rounded-lg border border-border bg-surface">
				<table class="w-full text-sm">
					<thead class="bg-elevated text-xs uppercase text-tertiary">
						<tr>
							<th class="px-3 py-2 text-left">Zaměstnavatel</th>
							<th class="px-3 py-2 text-left">Období</th>
							<th class="px-3 py-2 text-left">Typ smlouvy</th>
							<th class="px-3 py-2 text-left">Typ Potvrzení</th>
							<th class="px-3 py-2 text-right">ř. 31 (úhrn)</th>
							<th class="px-3 py-2 text-right">ř. 84 (zálohy)</th>
							<th class="px-3 py-2 text-right">ř. 87 (srážka)</th>
							<th class="px-3 py-2 text-left">Stav</th>
							<th class="px-3 py-2 text-right">Akce</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-border">
						{#each certificates as cert (cert.id)}
							<tr data-testid="certificate-row">
								<td class="px-3 py-2 text-primary">{cert.employer_name}</td>
								<td class="px-3 py-2 text-secondary">
									{cert.period_from} – {cert.period_to}
								</td>
								<td class="px-3 py-2 text-secondary">{contractLabel(cert.contract_type)}</td>
								<td class="px-3 py-2 text-secondary">{certTypeLabel(cert.certificate_type)}</td>
								<td class="px-3 py-2 text-right tabular-nums">{fmt(cert.gross_income_czk)}</td>
								<td class="px-3 py-2 text-right tabular-nums">
									{cert.certificate_type === 'advance'
										? fmt(cert.advance_tax_withheld_czk - cert.annual_settlement_refund_czk)
										: '—'}
								</td>
								<td class="px-3 py-2 text-right tabular-nums">
									{cert.certificate_type === 'withholding' && cert.include_withholding_in_dap
										? fmt(cert.withheld_final_tax_czk)
										: '—'}
								</td>
								<td class="px-3 py-2">
									<span
										class="inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium {statusColor(
											cert.status
										)}"
									>
										{statusLabel(cert.status)}
									</span>
								</td>
								<td class="px-3 py-2 text-right">
									<div class="flex justify-end gap-2">
										<Button
											variant="ghost"
											size="sm"
											onclick={() => editCertificate(cert)}
											disabled={working}
										>
											Upravit
										</Button>
										{#if cert.status !== 'confirmed'}
											<Button
												variant="success"
												size="sm"
												onclick={() => confirmCertificate(cert)}
												disabled={working}
											>
												Potvrdit
											</Button>
										{/if}
										<Button
											variant="danger"
											size="sm"
											onclick={() => deleteCertificate(cert.id)}
											disabled={working}
										>
											Smazat
										</Button>
									</div>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</div>
</div>

<EmploymentCertificateEditor
	bind:open={editorOpen}
	year={selectedYear}
	draft={editorDraft}
	onclose={() => {
		editorOpen = false;
	}}
	onsaved={onEditorSaved}
/>

<script lang="ts">
	import { nativeDownload } from '$lib/actions/download';
	import type { InvestmentDocument } from '$lib/api/client';
	import { investmentsApi } from '$lib/api/client';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';
	import Select from '$lib/ui/Select.svelte';

	interface Props {
		documents: InvestmentDocument[];
		uploadPlatform: string;
		uploading: boolean;
		saving: boolean;
		extractingId: number | null;
		onUploadPlatformChange: (value: string) => void;
		onUpload: (kind: 'statement' | 'data') => void;
		onExtract: (id: number) => void;
		onDelete: (id: number) => void;
	}

	let {
		documents,
		uploadPlatform,
		uploading,
		saving,
		extractingId,
		onUploadPlatformChange,
		onUpload,
		onExtract,
		onDelete
	}: Props = $props();

	const platformLabels: Record<string, string> = {
		portu: 'Portu',
		zonky: 'Zonky',
		trading212: 'Trading 212',
		revolut: 'Revolut',
		other: 'Jiný'
	};

	const statusLabels: Record<string, { text: string; class: string }> = {
		pending: { text: 'Čeká na zpracování', class: 'bg-warning-bg text-warning' },
		extracted: { text: 'Extrahováno', class: 'bg-success-bg text-success' },
		failed: { text: 'Chyba', class: 'bg-danger-bg text-danger' },
		skipped: { text: 'Příloha (bez OCR)', class: 'bg-info-bg text-info' }
	};

	function formatFileSize(bytes: number): string {
		if (bytes < 1024) return `${bytes} B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
		return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	}
</script>

<Card>
	{#if extractingId !== null}
		<div
			class="mb-4 flex items-start gap-2 rounded border border-border bg-info-bg p-3 text-sm text-info"
			role="status"
		>
			<span
				class="mt-0.5 inline-block h-4 w-4 shrink-0 animate-spin rounded-full border-2 border-current border-t-transparent"
				aria-hidden="true"
			></span>
			<div>
				<div class="font-medium">Probíhá AI extrakce dat z dokumentu...</div>
				<div class="text-xs">U větších brokerských výpisů to může trvat 1-3 minuty.</div>
			</div>
		</div>
	{/if}

	<div class="flex flex-wrap items-center justify-between gap-2">
		<h2 class="text-base font-semibold text-primary">Nahrané dokumenty</h2>
		<div class="flex flex-wrap items-center gap-2">
			<Select
				value={uploadPlatform}
				onchange={(e: Event) => {
					onUploadPlatformChange((e.currentTarget as HTMLSelectElement).value);
				}}
			>
				{#each Object.entries(platformLabels) as [key, label]}
					<option value={key}>{label}</option>
				{/each}
			</Select>
			<Button
				variant="primary"
				size="sm"
				onclick={() => onUpload('statement')}
				disabled={uploading}
				title="Výpis z brokerského účtu pro AI extrakci (PDF, JPG, PNG)"
			>
				{uploading ? 'Nahrává se...' : 'Nahrát výpis (OCR)'}
			</Button>
			<Button
				variant="secondary"
				size="sm"
				onclick={() => onUpload('data')}
				disabled={uploading}
				title="Datový export (XLSX, CSV, ZIP) — jen příloha k DPFO, bez OCR"
			>
				{uploading ? 'Nahrává se...' : 'Nahrát data (XLSX/CSV)'}
			</Button>
		</div>
	</div>

	{#if documents.length > 0}
		<div class="mt-4 overflow-x-auto">
			<table class="w-full text-sm">
				<thead>
					<tr class="border-b border-border text-left text-xs text-tertiary">
						<th class="pb-2 pr-4">Název souboru</th>
						<th class="pb-2 pr-4">Platforma</th>
						<th class="pb-2 pr-4">Stav</th>
						<th class="pb-2 pr-4">Velikost</th>
						<th class="pb-2 text-right">Akce</th>
					</tr>
				</thead>
				<tbody>
					{#each documents as doc (doc.id)}
						<tr class="border-b border-border-subtle">
							<td class="py-2 pr-4">
								<a
									href={investmentsApi.downloadDocumentUrl(doc.id)}
									class="text-accent hover:underline"
									target="_blank"
									use:nativeDownload={doc.filename}>{doc.filename}</a
								>
							</td>
							<td class="py-2 pr-4 text-tertiary">{platformLabels[doc.platform] ?? doc.platform}</td
							>
							<td class="py-2 pr-4">
								{#if statusLabels[doc.extraction_status]}
									{@const status = statusLabels[doc.extraction_status]}
									<span
										class="inline-flex rounded-full px-2 py-0.5 text-xs font-medium {status.class}"
									>
										{status.text}
									</span>
								{:else}
									<span
										class="inline-flex rounded-full bg-surface px-2 py-0.5 text-xs font-medium text-tertiary"
									>
										{doc.extraction_status}
									</span>
								{/if}
								{#if doc.extraction_error}
									<span class="ml-1 text-xs text-danger" title={doc.extraction_error}>!</span>
								{/if}
							</td>
							<td class="py-2 pr-4 text-tertiary">{formatFileSize(doc.size)}</td>
							<td class="py-2 text-right">
								<div class="flex justify-end gap-1">
									{#if doc.kind !== 'data' && doc.extraction_status !== 'extracted'}
										<Button
											variant="secondary"
											size="sm"
											onclick={() => onExtract(doc.id)}
											disabled={saving}
										>
											{#if extractingId === doc.id}
												<span
													class="inline-block h-3 w-3 animate-spin rounded-full border-2 border-current border-t-transparent"
													role="status"
													aria-hidden="true"
												></span>
												<span class="sr-only">Probíhá extrakce</span>
												Extrahuje se...
											{:else}
												Extrahovat
											{/if}
										</Button>
									{/if}
									<Button
										variant="danger"
										size="sm"
										onclick={() => onDelete(doc.id)}
										disabled={saving}>Smazat</Button
									>
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{:else}
		<p class="mt-4 text-sm text-tertiary">
			Žádné nahrané dokumenty. Nahrajte výpisy z investičních platforem pro automatickou extrakci
			dat.
		</p>
	{/if}
</Card>

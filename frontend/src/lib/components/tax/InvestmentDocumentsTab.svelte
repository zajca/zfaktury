<script lang="ts">
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
		onUploadPlatformChange: (value: string) => void;
		onUpload: () => void;
		onExtract: (id: number) => void;
		onDelete: (id: number) => void;
	}

	let {
		documents,
		uploadPlatform,
		uploading,
		saving,
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
		failed: { text: 'Chyba', class: 'bg-danger-bg text-danger' }
	};

	function formatFileSize(bytes: number): string {
		if (bytes < 1024) return `${bytes} B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
		return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	}
</script>

<Card>
	<div class="flex items-center justify-between">
		<h2 class="text-base font-semibold text-primary">Nahrané dokumenty</h2>
		<div class="flex items-center gap-2">
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
			<Button variant="primary" size="sm" onclick={onUpload} disabled={uploading}>
				{uploading ? 'Nahrává se...' : 'Nahrát dokument'}
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
									target="_blank">{doc.filename}</a
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
									{#if doc.extraction_status !== 'extracted'}
										<Button
											variant="secondary"
											size="sm"
											onclick={() => onExtract(doc.id)}
											disabled={saving}>Extrahovat</Button
										>
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

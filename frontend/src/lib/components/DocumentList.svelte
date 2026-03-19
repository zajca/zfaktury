<script lang="ts">
	import { nativeDownload } from '$lib/actions/download';
	import { documentsApi, type ExpenseDocument } from '$lib/api/client';
	import { formatDate } from '$lib/utils/date';

	interface Props {
		documents: ExpenseDocument[];
		ondelete?: (id: number) => void;
		onocr?: (docId: number) => void;
	}

	let { documents, ondelete, onocr }: Props = $props();

	let confirmDeleteId = $state<number | null>(null);
	let deleting = $state(false);

	function formatFileSize(bytes: number): string {
		if (bytes < 1024) return `${bytes} B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
		return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	}

	function handleDeleteClick(id: number) {
		confirmDeleteId = id;
	}

	function cancelDelete() {
		confirmDeleteId = null;
	}

	async function confirmDelete() {
		if (confirmDeleteId == null) return;

		const id = confirmDeleteId;
		deleting = true;

		try {
			await documentsApi.delete(id);
			confirmDeleteId = null;
			ondelete?.(id);
		} catch {
			// Error is handled silently; the document remains in the list
		} finally {
			deleting = false;
		}
	}
</script>

{#if documents.length === 0}
	<p class="py-4 text-center text-sm text-muted">Žádné dokumenty</p>
{:else}
	<ul class="divide-y divide-border">
		{#each documents as doc (doc.id)}
			<li class="flex items-center justify-between gap-3 py-3">
				<div class="min-w-0 flex-1">
					<p class="truncate text-sm font-medium text-primary">{doc.filename}</p>
					<p class="text-xs text-muted">
						{formatFileSize(doc.size)} — {formatDate(doc.created_at)}
					</p>
				</div>

				<div class="flex shrink-0 items-center gap-1.5">
					<a
						href={documentsApi.getDownloadUrl(doc.id)}
						target="_blank"
						rel="noopener noreferrer"
						class="rounded-md px-2.5 py-1.5 text-xs text-secondary hover:bg-hover hover:text-primary transition-colors"
						title="Stáhnout"
						use:nativeDownload={doc.filename}
					>
						<svg
							class="h-4 w-4"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="1.5"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5M16.5 12L12 16.5m0 0L7.5 12m4.5 4.5V3"
							/>
						</svg>
					</a>

					{#if onocr}
						<button
							type="button"
							onclick={() => onocr?.(doc.id)}
							class="rounded-md px-2.5 py-1.5 text-xs text-secondary hover:bg-hover hover:text-primary transition-colors"
							title="Spustit OCR"
						>
							OCR
						</button>
					{/if}

					<button
						type="button"
						onclick={() => handleDeleteClick(doc.id)}
						class="rounded-md px-2.5 py-1.5 text-xs text-danger hover:bg-danger-bg transition-colors"
						title="Smazat"
					>
						<svg
							class="h-4 w-4"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="1.5"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0"
							/>
						</svg>
					</button>
				</div>
			</li>
		{/each}
	</ul>
{/if}

{#if confirmDeleteId !== null}
	<div
		role="presentation"
		class="fixed inset-0 z-40 bg-black/50"
		onclick={cancelDelete}
		onkeydown={() => {}}
	></div>
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-4"
		role="dialog"
		aria-modal="true"
	>
		<div class="w-full max-w-sm rounded-xl border border-border bg-surface p-6 shadow-xl">
			<p class="mb-4 text-sm text-primary">Opravdu chcete smazat tento dokument?</p>
			<div class="flex justify-end gap-2">
				<button
					type="button"
					onclick={cancelDelete}
					class="rounded-lg border border-border px-4 py-2 text-sm text-secondary hover:bg-hover transition-colors"
				>
					Zrušit
				</button>
				<button
					type="button"
					onclick={confirmDelete}
					disabled={deleting}
					class="rounded-lg bg-danger px-4 py-2 text-sm text-white hover:bg-danger/90 transition-colors disabled:opacity-50"
				>
					{deleting ? 'Mažu...' : 'Smazat'}
				</button>
			</div>
		</div>
	</div>
{/if}

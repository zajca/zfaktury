<script lang="ts">
	import { onMount } from 'svelte';
	import { nativeDownload } from '$lib/actions/download';
	import { backupApi, type BackupRecord, type BackupStatus } from '$lib/api/client';
	import Card from '$lib/ui/Card.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Badge from '$lib/ui/Badge.svelte';
	import ConfirmDialog from '$lib/ui/ConfirmDialog.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';
	import { toastSuccess, toastError } from '$lib/data/toast-state.svelte';

	let loading = $state(true);
	let error = $state<string | null>(null);
	let backups = $state<BackupRecord[]>([]);
	let status = $state<BackupStatus | null>(null);
	let creating = $state(false);
	let deleteId = $state<number | null>(null);
	let showDeleteConfirm = $state(false);

	async function loadData() {
		loading = true;
		error = null;
		try {
			const [backupList, backupStatus] = await Promise.all([
				backupApi.list(),
				backupApi.getStatus()
			]);
			backups = backupList;
			status = backupStatus;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodarilo se nacist data zaloh';
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		loadData();
	});

	async function createBackup() {
		creating = true;
		try {
			await backupApi.create();
			toastSuccess('Zaloha byla uspesne vytvorena');
			await loadData();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodarilo se vytvorit zalohu');
		} finally {
			creating = false;
		}
	}

	function handleDelete(id: number) {
		deleteId = id;
		showDeleteConfirm = true;
	}

	async function confirmDelete() {
		if (!deleteId) return;
		showDeleteConfirm = false;
		try {
			await backupApi.delete(deleteId);
			toastSuccess('Zaloha byla smazana');
			await loadData();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodarilo se smazat zalohu');
		} finally {
			deleteId = null;
		}
	}

	function formatBytes(bytes: number): string {
		if (bytes === 0) return '0 B';
		const units = ['B', 'KB', 'MB', 'GB'];
		const i = Math.floor(Math.log(bytes) / Math.log(1024));
		const value = bytes / Math.pow(1024, i);
		return `${value.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
	}

	function formatDuration(ms: number): string {
		if (ms < 1000) return `${ms} ms`;
		return `${(ms / 1000).toFixed(1)} s`;
	}

	function formatDate(dateStr: string): string {
		return new Date(dateStr).toLocaleString('cs-CZ');
	}

	function statusVariant(s: BackupRecord['status']): 'warning' | 'success' | 'danger' {
		switch (s) {
			case 'running':
				return 'warning';
			case 'completed':
				return 'success';
			case 'failed':
				return 'danger';
		}
	}

	function statusLabel(s: BackupRecord['status']): string {
		switch (s) {
			case 'running':
				return 'Probiha...';
			case 'completed':
				return 'Dokonceno';
			case 'failed':
				return 'Chyba';
		}
	}
</script>

<svelte:head>
	<title>Zalohy - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-5xl">
	<PageHeader
		title="Zalohy"
		description="Sprava zaloh databaze a souboru"
		backHref="/settings"
		backLabel="Zpet na nastaveni"
	>
		{#snippet actions()}
			<Button variant="primary" onclick={createBackup} disabled={creating || status?.is_running}>
				{#if creating}
					Vytvarim...
				{:else}
					Vytvorit zalohu
				{/if}
			</Button>
		{/snippet}
	</PageHeader>

	<ErrorAlert {error} class="mt-4" />

	{#if loading}
		<LoadingSpinner class="mt-8" />
	{:else}
		{#if status}
			<Card class="mt-6">
				<h2 class="text-base font-semibold text-primary">Stav</h2>
				<div class="mt-3 grid grid-cols-1 gap-4 sm:grid-cols-3">
					<div>
						<p class="text-xs font-medium text-muted">Aktualni stav</p>
						<p class="mt-1">
							{#if status.is_running}
								<Badge variant="warning">Probiha zalohovani</Badge>
							{:else}
								<Badge variant="success">Necinna</Badge>
							{/if}
						</p>
					</div>
					<div>
						<p class="text-xs font-medium text-muted">Posledni zaloha</p>
						<p class="mt-1 text-sm text-primary">
							{#if status.last_backup}
								{formatDate(status.last_backup.created_at)}
								<Badge variant={statusVariant(status.last_backup.status)} class="ml-1">
									{statusLabel(status.last_backup.status)}
								</Badge>
							{:else}
								<span class="text-tertiary">Zadna zaloha</span>
							{/if}
						</p>
					</div>
					<div>
						<p class="text-xs font-medium text-muted">Dalsi planovana</p>
						<p class="mt-1 text-sm text-primary">
							{#if status.next_scheduled}
								{formatDate(status.next_scheduled)}
							{:else}
								<span class="text-tertiary">Nenastaveno</span>
							{/if}
						</p>
					</div>
				</div>
			</Card>
		{/if}

		<div class="mt-6 overflow-x-auto rounded-lg border border-border bg-surface">
			<table class="min-w-full divide-y divide-border">
				<thead class="bg-elevated">
					<tr>
						<th
							class="px-4 py-2.5 text-left text-xs font-medium uppercase tracking-wider text-muted"
						>
							Datum
						</th>
						<th
							class="px-4 py-2.5 text-left text-xs font-medium uppercase tracking-wider text-muted"
						>
							Soubor
						</th>
						<th
							class="px-4 py-2.5 text-left text-xs font-medium uppercase tracking-wider text-muted"
						>
							Velikost
						</th>
						<th
							class="px-4 py-2.5 text-left text-xs font-medium uppercase tracking-wider text-muted"
						>
							Trvani
						</th>
						<th
							class="px-4 py-2.5 text-left text-xs font-medium uppercase tracking-wider text-muted"
						>
							Uloziste
						</th>
						<th
							class="px-4 py-2.5 text-left text-xs font-medium uppercase tracking-wider text-muted"
						>
							Stav
						</th>
						<th
							class="px-4 py-2.5 text-right text-xs font-medium uppercase tracking-wider text-muted"
						>
							Akce
						</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-border-subtle">
					{#each backups as backup (backup.id)}
						<tr class="hover:bg-hover">
							<td class="whitespace-nowrap px-4 py-2.5 text-sm text-primary">
								{formatDate(backup.created_at)}
							</td>
							<td class="whitespace-nowrap px-4 py-2.5 text-sm font-mono text-secondary">
								{backup.filename}
							</td>
							<td class="whitespace-nowrap px-4 py-2.5 text-sm text-tertiary">
								{formatBytes(backup.size_bytes)}
							</td>
							<td class="whitespace-nowrap px-4 py-2.5 text-sm text-tertiary">
								{formatDuration(backup.duration_ms)}
							</td>
							<td class="whitespace-nowrap px-4 py-2.5">
								<Badge variant={backup.destination === 's3' ? 'info' : 'default'}>
									{backup.destination === 's3' ? 'S3' : 'Lokalni'}
								</Badge>
							</td>
							<td class="whitespace-nowrap px-4 py-2.5">
								<Badge variant={statusVariant(backup.status)}>
									{statusLabel(backup.status)}
								</Badge>
								{#if backup.error_message}
									<p class="mt-1 text-xs text-danger">{backup.error_message}</p>
								{/if}
							</td>
							<td class="whitespace-nowrap px-4 py-2.5 text-right text-sm">
								{#if backup.status === 'completed'}
									<a
										href={backupApi.downloadUrl(backup.id)}
										class="text-accent-text hover:text-accent mr-3"
										download
										use:nativeDownload={backup.filename}
									>
										Stahnout
									</a>
								{/if}
								<button onclick={() => handleDelete(backup.id)} class="text-danger hover:underline">
									Smazat
								</button>
							</td>
						</tr>
					{/each}
					{#if backups.length === 0}
						<tr>
							<td colspan="7" class="px-4 py-8 text-center text-sm text-tertiary">
								Zadne zalohy
							</td>
						</tr>
					{/if}
				</tbody>
			</table>
		</div>

		<Card class="mt-6">
			<h2 class="text-base font-semibold text-primary">Synchronizace dat</h2>
			<p class="mt-2 text-sm text-secondary">
				ZFaktury nepodporuje vestavenou synchronizaci. Pro pristup z vice zarizeni doporucujeme:
			</p>
			<ul class="mt-3 list-inside list-disc space-y-1.5 text-sm text-secondary">
				<li>
					Nastavit <code class="rounded bg-elevated px-1 py-0.5 text-xs">ZFAKTURY_DATA_DIR</code> na sdileny
					adresar (Syncthing, NAS, Nextcloud)
				</li>
				<li>Zajistit ze bezi vzdy jen jedna instance (automaticky chraneno zamkem)</li>
				<li>
					Pouzit <code class="rounded bg-elevated px-1 py-0.5 text-xs">journal_mode = "delete"</code
					> v konfiguraci pro sitove filesystemy
				</li>
			</ul>
		</Card>
	{/if}
</div>

<ConfirmDialog
	bind:open={showDeleteConfirm}
	title="Smazat zalohu"
	message="Opravdu chcete smazat tuto zalohu? Soubor bude trvale odstranen."
	confirmLabel="Smazat"
	onconfirm={confirmDelete}
	oncancel={() => (showDeleteConfirm = false)}
/>

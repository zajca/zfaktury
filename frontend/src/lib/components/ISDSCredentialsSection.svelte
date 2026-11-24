<script lang="ts">
	import { onMount } from 'svelte';
	import {
		isdsCredentialsApi,
		type ISDSCredentialsOverview,
		type ISDSEnvironment,
		type ISDSLoginTestResult
	} from '$lib/api/client';
	import { environmentLabels, environmentOrder, filingErrorMessage } from '$lib/utils/filing';
	import Card from '$lib/ui/Card.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Input from '$lib/ui/Input.svelte';
	import ConfirmDialog from '$lib/ui/ConfirmDialog.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import { toastSuccess, toastError } from '$lib/data/toast-state.svelte';

	const environments: { id: ISDSEnvironment; label: string }[] = environmentOrder
		.slice()
		.reverse()
		.map((id) => ({ id, label: environmentLabels[id] }));

	let overview = $state<Partial<ISDSCredentialsOverview>>({});
	let loading = $state(true);
	let loadError = $state<string | null>(null);

	let usernames = $state<Record<ISDSEnvironment, string>>({ test: '', production: '' });
	let passwords = $state<Record<ISDSEnvironment, string>>({ test: '', production: '' });
	let formError = $state<Record<ISDSEnvironment, string | null>>({ test: null, production: null });
	let busy = $state<ISDSEnvironment | null>(null);
	let testing = $state<ISDSEnvironment | null>(null);
	let testResult = $state<Record<ISDSEnvironment, ISDSLoginTestResult | null>>({
		test: null,
		production: null
	});
	let testError = $state<Record<ISDSEnvironment, string | null>>({ test: null, production: null });
	let deleteTarget = $state<ISDSEnvironment | null>(null);
	let showDeleteConfirm = $state(false);

	onMount(() => {
		loadStatus();
	});

	async function loadStatus() {
		loading = true;
		loadError = null;
		try {
			overview = await isdsCredentialsApi.get();
		} catch (e) {
			loadError = e instanceof Error ? e.message : 'Nepodařilo se načíst přihlašovací údaje';
		} finally {
			loading = false;
		}
	}

	async function handleSave(env: ISDSEnvironment) {
		busy = env;
		formError[env] = null;
		try {
			const status = await isdsCredentialsApi.save({
				environment: env,
				username: usernames[env],
				password: passwords[env]
			});
			overview = { ...overview, [env]: status };
			testResult[env] = null;
			testError[env] = null;
			usernames[env] = '';
			passwords[env] = '';
			toastSuccess('Přihlašovací údaje uloženy do systémové klíčenky');
		} catch (e) {
			formError[env] = e instanceof Error ? e.message : 'Nepodařilo se uložit přihlašovací údaje';
		} finally {
			busy = null;
		}
	}

	async function handleTest(env: ISDSEnvironment) {
		if (testing !== null) return;
		testing = env;
		testError[env] = null;
		testResult[env] = null;
		try {
			testResult[env] = await isdsCredentialsApi.test(env);
		} catch (e) {
			testError[env] = filingErrorMessage(e);
		} finally {
			testing = null;
		}
	}

	function askDelete(env: ISDSEnvironment) {
		deleteTarget = env;
		showDeleteConfirm = true;
	}

	async function confirmDelete() {
		const env = deleteTarget;
		showDeleteConfirm = false;
		deleteTarget = null;
		if (!env) return;
		busy = env;
		try {
			await isdsCredentialsApi.remove(env);
			overview = { ...overview, [env]: { configured: false, username: '' } };
			testResult[env] = null;
			testError[env] = null;
			toastSuccess('Přihlašovací údaje smazány');
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se smazat přihlašovací údaje');
		} finally {
			busy = null;
		}
	}
</script>

<section class="mt-6 pb-8" aria-labelledby="isds-heading">
	<Card>
		<h2 id="isds-heading" class="text-base font-semibold text-primary">Datová schránka</h2>
		<p class="mt-1 text-sm text-tertiary">
			Přihlašovací údaje pověřené osoby, kterou ZFaktury použije k odesílání podání datovou
			schránkou.
		</p>

		<div class="mt-4 rounded-lg border border-warning/30 bg-warning-bg p-4 text-sm text-primary">
			<p class="font-medium">Použijte pověřenou osobu s minimem oprávnění</p>
			<p class="mt-1">
				Pověřené osobě nechte jen oprávnění „vytvářet a odesílat zprávy“ a „seznam odeslaných zpráv
				a doručenky“. Účet, který smí číst doručené zprávy, způsobí doručení přihlášením a tím
				spustí zákonné lhůty.
			</p>
			<p class="mt-2">
				Heslo se ukládá do systémové klíčenky a už se nikdy nezobrazí. Mimo klíčenku se neukládá
				nikde.
			</p>
		</div>

		<ErrorAlert error={loadError} class="mt-4" />

		{#if loading}
			<LoadingSpinner class="mt-4" />
		{:else}
			<div class="mt-4 grid gap-6 lg:grid-cols-2">
				{#each environments as env (env.id)}
					{@const status = overview[env.id]}
					<form
						class="rounded-lg border border-border p-4"
						autocomplete="off"
						onsubmit={(e) => {
							e.preventDefault();
							handleSave(env.id);
						}}
					>
						<h3 class="text-sm font-semibold text-primary">{env.label}</h3>
						<p class="mt-1 text-sm" data-testid="isds-status-{env.id}">
							{#if status?.configured}
								<span class="text-success">Přihlášení uloženo</span>
								<span class="text-tertiary">– uživatel {status.username}</span>
							{:else}
								<span class="text-tertiary">Přihlášení není uloženo</span>
							{/if}
						</p>

						<ErrorAlert error={formError[env.id]} class="mt-3" />

						<div class="mt-3 space-y-3">
							<div>
								<label for="isds-username-{env.id}" class="block text-sm font-medium text-secondary"
									>Uživatelské jméno</label
								>
								<Input
									id="isds-username-{env.id}"
									type="text"
									autocomplete="off"
									value={usernames[env.id]}
									oninput={(e) => (usernames[env.id] = (e.target as HTMLInputElement).value)}
									class="mt-1"
								/>
							</div>
							<div>
								<label for="isds-password-{env.id}" class="block text-sm font-medium text-secondary"
									>Heslo</label
								>
								<Input
									id="isds-password-{env.id}"
									type="password"
									autocomplete="off"
									value={passwords[env.id]}
									oninput={(e) => (passwords[env.id] = (e.target as HTMLInputElement).value)}
									class="mt-1"
								/>
							</div>
						</div>

						<div class="mt-4 flex flex-wrap gap-2">
							<Button
								type="submit"
								variant="primary"
								disabled={busy !== null || !usernames[env.id] || !passwords[env.id]}
							>
								{status?.configured ? 'Přepsat přihlášení' : 'Uložit přihlášení'}
							</Button>
							{#if status?.configured}
								<Button
									variant="secondary"
									disabled={busy !== null || testing !== null}
									onclick={() => handleTest(env.id)}
								>
									{testing === env.id ? 'Zkouším přihlášení...' : 'Vyzkoušet přihlášení'}
								</Button>
							{/if}
							{#if status?.configured || status?.username}
								<Button variant="danger" disabled={busy !== null} onclick={() => askDelete(env.id)}>
									Smazat přihlášení
								</Button>
							{/if}
						</div>

						<ErrorAlert error={testError[env.id]} class="mt-3" />

						{#if testResult[env.id]}
							{@const result = testResult[env.id]!}
							<div class="mt-3 space-y-3 text-sm" data-testid="isds-test-result-{env.id}">
								{#if result.warning}
									<div
										role="alert"
										class="rounded-lg border border-warning/40 bg-warning-bg p-3 font-medium text-primary"
									>
										{result.warning}
									</div>
								{/if}
								<dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1">
									<dt class="text-muted">Číslo schránky</dt>
									<dd class="font-mono text-primary">{result.boxId}</dd>
									<dt class="text-muted">Typ schránky</dt>
									<dd class="text-primary">{result.boxType}</dd>
									<dt class="text-muted">Majitel</dt>
									<dd class="text-primary">{result.ownerName}</dd>
									<dt class="text-muted">Smí odesílat</dt>
									<dd class={result.canCreateAndSend ? 'text-success' : 'text-danger'}>
										{result.canCreateAndSend ? 'Ano' : 'Ne'}
									</dd>
									<dt class="text-muted">Smí číst doručené</dt>
									<dd class={result.canReadMessages ? 'text-warning' : 'text-primary'}>
										{result.canReadMessages ? 'Ano' : 'Ne'}
									</dd>
								</dl>
							</div>
						{/if}
					</form>
				{/each}
			</div>
		{/if}
	</Card>
</section>

<ConfirmDialog
	bind:open={showDeleteConfirm}
	title="Smazat přihlášení datové schránky"
	message="Opravdu chcete smazat uložené přihlášení z klíčenky? Odesílání datovou schránkou přestane pro dané prostředí fungovat."
	confirmLabel="Smazat"
	onconfirm={confirmDelete}
	oncancel={() => {
		showDeleteConfirm = false;
		deleteTarget = null;
	}}
/>

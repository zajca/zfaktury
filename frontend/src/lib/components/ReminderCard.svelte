<script lang="ts">
	import { onMount } from 'svelte';
	import { remindersApi, ApiError, type PaymentReminder } from '$lib/api/client';
	import { formatDateTime } from '$lib/utils/date';
	import Card from '$lib/ui/Card.svelte';
	import Badge from '$lib/ui/Badge.svelte';
	import Button from '$lib/ui/Button.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';

	interface Props {
		invoiceId: number;
		invoiceStatus: string;
	}

	let { invoiceId, invoiceStatus }: Props = $props();

	let reminders: PaymentReminder[] = $state([]);
	let loading = $state(true);
	let sending = $state(false);
	let error: string | null = $state(null);
	let expanded = $state(false);

	async function loadReminders() {
		try {
			loading = true;
			error = null;
			reminders = await remindersApi.listReminders(invoiceId);
		} catch (e) {
			if (e instanceof ApiError) {
				error = e.message;
			} else {
				error = 'Nepodařilo se načíst upomínky';
			}
		} finally {
			loading = false;
		}
	}

	async function sendReminder() {
		try {
			sending = true;
			error = null;
			await remindersApi.sendReminder(invoiceId);
			await loadReminders();
		} catch (e) {
			if (e instanceof ApiError) {
				if (e.message.includes('no email') || e.message.includes('nemá email')) {
					error = 'Zákazník nemá email';
				} else if (e.message.includes('not overdue') || e.message.includes('není po splatnosti')) {
					error = 'Faktura není po splatnosti';
				} else {
					error = e.message;
				}
			} else {
				error = 'Nepodařilo se odeslat upomínku';
			}
		} finally {
			sending = false;
		}
	}

	function toggleExpanded() {
		expanded = !expanded;
	}

	onMount(() => {
		loadReminders();
	});
</script>

<Card padding={false}>
	<button
		type="button"
		class="flex w-full items-center justify-between p-5 text-left hover:bg-hover transition-colors"
		onclick={toggleExpanded}
		data-testid="reminder-header"
	>
		<div class="flex items-center gap-2">
			<h3 class="text-sm font-medium text-primary">Upomínky</h3>
			{#if !loading}
				<Badge variant={reminders.length > 0 ? 'warning' : 'muted'}>
					{reminders.length}
				</Badge>
			{/if}
		</div>
		<svg
			class="h-4 w-4 text-muted transition-transform {expanded ? 'rotate-180' : ''}"
			fill="none"
			viewBox="0 0 24 24"
			stroke="currentColor"
			stroke-width="2"
		>
			<path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
		</svg>
	</button>

	{#if expanded}
		<div class="border-t border-border px-5 pb-5" data-testid="reminder-content">
			<ErrorAlert error={error} class="mt-4" />

			{#if loading}
				<LoadingSpinner class="py-6" />
			{:else}
				{#if reminders.length === 0}
					<p class="py-4 text-sm text-muted" data-testid="empty-state">Žádné odeslané upomínky</p>
				{:else}
					<div class="mt-4 space-y-3" data-testid="reminder-list">
						{#each reminders as reminder (reminder.id)}
							<div class="rounded-lg border border-border bg-elevated p-3" data-testid="reminder-item">
								<div class="flex items-center justify-between">
									<span class="text-sm font-medium text-primary">
										Upomínka #{reminder.reminder_number}
									</span>
									<span class="text-xs text-muted" data-testid="reminder-date">
										{formatDateTime(reminder.sent_at)}
									</span>
								</div>
								<p class="mt-1 text-xs text-secondary">{reminder.sent_to}</p>
								<p class="mt-1 text-xs text-muted">{reminder.subject}</p>
							</div>
						{/each}
					</div>
				{/if}

				{#if invoiceStatus === 'overdue'}
					<div class="mt-4">
						<Button
							variant="primary"
							size="sm"
							onclick={sendReminder}
							disabled={sending}
						>
							{#if sending}
								Odesílám...
							{:else}
								Odeslat upomínku
							{/if}
						</Button>
					</div>
				{/if}
			{/if}
		</div>
	{/if}
</Card>

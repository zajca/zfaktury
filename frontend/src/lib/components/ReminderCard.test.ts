import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';
import ReminderCard from './ReminderCard.svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

const sampleReminders = [
	{
		id: 1,
		invoice_id: 42,
		reminder_number: 1,
		sent_at: '2026-03-08T10:00:00Z',
		sent_to: 'jan@example.com',
		subject: 'Upominka - Faktura 2026001',
		body_preview: 'Dobry den...',
		created_at: '2026-03-08T10:00:00Z'
	},
	{
		id: 2,
		invoice_id: 42,
		reminder_number: 2,
		sent_at: '2026-03-10T14:00:00Z',
		sent_to: 'jan@example.com',
		subject: 'Druha upominka - Faktura 2026001',
		body_preview: 'Dobry den...',
		created_at: '2026-03-10T14:00:00Z'
	}
];

beforeEach(() => {
	mockFetch.mockReset();
});

afterEach(() => {
	cleanup();
});

describe('ReminderCard', () => {
	it('loads and displays reminders on mount', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleReminders));

		render(ReminderCard, { props: { invoiceId: 42, invoiceStatus: 'overdue' } });

		// Wait for loading to finish and badge to appear
		await waitFor(() => {
			expect(screen.getByText('2')).toBeInTheDocument();
		});

		// Verify the API call
		const [url] = mockFetch.mock.calls[0];
		expect(url).toBe('/api/v1/invoices/42/reminders');

		// Expand to see content
		await fireEvent.click(screen.getByTestId('reminder-header'));

		// Should show both reminders
		const items = screen.getAllByTestId('reminder-item');
		expect(items).toHaveLength(2);

		expect(screen.getByText('Upomínka #1')).toBeInTheDocument();
		expect(screen.getByText('Upomínka #2')).toBeInTheDocument();
		expect(screen.getAllByText('jan@example.com')).toHaveLength(2);
		expect(screen.getByText('Upominka - Faktura 2026001')).toBeInTheDocument();
	});

	it('shows send button only for overdue invoices', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse([]));

		render(ReminderCard, { props: { invoiceId: 42, invoiceStatus: 'overdue' } });

		// Wait for loading to finish
		await waitFor(() => {
			expect(screen.getByText('0')).toBeInTheDocument();
		});

		await fireEvent.click(screen.getByTestId('reminder-header'));

		expect(screen.getByText('Odeslat upomínku')).toBeInTheDocument();
	});

	it('hides send button for non-overdue statuses', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse([]));

		render(ReminderCard, { props: { invoiceId: 42, invoiceStatus: 'sent' } });

		// Wait for loading to finish
		await waitFor(() => {
			expect(screen.getByText('0')).toBeInTheDocument();
		});

		await fireEvent.click(screen.getByTestId('reminder-header'));

		expect(screen.queryByText('Odeslat upomínku')).not.toBeInTheDocument();
	});

	it('sends reminder and reloads list', async () => {
		// Initial load - empty list
		mockFetch.mockResolvedValueOnce(jsonResponse([]));

		render(ReminderCard, { props: { invoiceId: 42, invoiceStatus: 'overdue' } });

		// Wait for loading to finish
		await waitFor(() => {
			expect(screen.getByText('0')).toBeInTheDocument();
		});

		await fireEvent.click(screen.getByTestId('reminder-header'));

		// Mock send response + reload response
		const newReminder = sampleReminders[0];
		mockFetch.mockResolvedValueOnce(jsonResponse(newReminder));
		mockFetch.mockResolvedValueOnce(jsonResponse([newReminder]));

		await fireEvent.click(screen.getByText('Odeslat upomínku'));

		// Wait for send + reload to complete
		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledTimes(3);
		});

		// Verify the send call
		const [sendUrl, sendOptions] = mockFetch.mock.calls[1];
		expect(sendUrl).toBe('/api/v1/invoices/42/remind');
		expect(sendOptions.method).toBe('POST');

		// After reload, should show 1 reminder
		await waitFor(() => {
			expect(screen.getByText('Upomínka #1')).toBeInTheDocument();
		});
	});

	it('shows error message on API failure', async () => {
		// Initial load succeeds
		mockFetch.mockResolvedValueOnce(jsonResponse([]));

		render(ReminderCard, { props: { invoiceId: 42, invoiceStatus: 'overdue' } });

		// Wait for loading to finish
		await waitFor(() => {
			expect(screen.getByText('0')).toBeInTheDocument();
		});

		await fireEvent.click(screen.getByTestId('reminder-header'));

		// Send fails with no-email error
		mockFetch.mockResolvedValueOnce(
			jsonResponse({ error: 'customer has no email' }, 422)
		);

		await fireEvent.click(screen.getByText('Odeslat upomínku'));

		await waitFor(() => {
			expect(screen.getByRole('alert')).toBeInTheDocument();
		});
	});

	it('handles empty reminder list', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse([]));

		render(ReminderCard, { props: { invoiceId: 42, invoiceStatus: 'overdue' } });

		// Wait for loading to finish, badge should show 0
		await waitFor(() => {
			expect(screen.getByText('0')).toBeInTheDocument();
		});

		// Expand to see empty state
		await fireEvent.click(screen.getByTestId('reminder-header'));

		expect(screen.getByTestId('empty-state')).toBeInTheDocument();
		expect(screen.getByText('Žádné odeslané upomínky')).toBeInTheDocument();
	});

	it('toggles visibility on header click (collapsible behavior)', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleReminders));

		render(ReminderCard, { props: { invoiceId: 42, invoiceStatus: 'overdue' } });

		// Wait for loading to finish
		await waitFor(() => {
			expect(screen.getByText('2')).toBeInTheDocument();
		});

		// Initially collapsed
		expect(screen.queryByTestId('reminder-content')).not.toBeInTheDocument();

		// Click to expand
		await fireEvent.click(screen.getByTestId('reminder-header'));
		expect(screen.getByTestId('reminder-content')).toBeInTheDocument();

		// Click again to collapse
		await fireEvent.click(screen.getByTestId('reminder-header'));
		expect(screen.queryByTestId('reminder-content')).not.toBeInTheDocument();
	});
});

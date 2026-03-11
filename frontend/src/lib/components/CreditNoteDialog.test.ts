import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';
import CreditNoteDialog from './CreditNoteDialog.svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

const sampleInvoice = {
	id: 99,
	sequence_id: 1,
	invoice_number: 'D-2026-001',
	type: 'credit_note' as const,
	status: 'draft' as const,
	issue_date: '2026-03-11',
	due_date: '2026-03-25',
	contact_id: 1,
	total_amount: -10000,
	total_vat: -2100,
	total_with_vat: -12100,
	currency: 'CZK',
	note: '',
	created_at: '2026-03-11T10:00:00Z',
	updated_at: '2026-03-11T10:00:00Z'
};

beforeEach(() => {
	mockFetch.mockReset();
});

afterEach(() => {
	cleanup();
});

describe('CreditNoteDialog', () => {
	it('renders dialog with form fields', () => {
		render(CreditNoteDialog, {
			props: { invoiceId: 1, onclose: vi.fn(), onsuccess: vi.fn() }
		});

		expect(screen.getByRole('heading', { name: 'Vytvořit dobropis' })).toBeInTheDocument();
		expect(screen.getByLabelText('Důvod dobropisu')).toBeInTheDocument();
		expect(screen.getByRole('button', { name: 'Zrušit' })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: 'Vytvořit dobropis' })).toBeInTheDocument();
	});

	it('submits credit note and calls onsuccess', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleInvoice));
		const onsuccess = vi.fn();

		render(CreditNoteDialog, {
			props: { invoiceId: 42, onclose: vi.fn(), onsuccess }
		});

		const input = screen.getByLabelText('Důvod dobropisu') as HTMLInputElement;
		// Remove required to bypass native validation in jsdom
		input.removeAttribute('required');
		await fireEvent.input(input, { target: { value: 'Reklamace zboží' } });

		const submitBtn = screen.getByRole('button', { name: 'Vytvořit dobropis' });
		await fireEvent.click(submitBtn);

		await waitFor(() => {
			expect(onsuccess).toHaveBeenCalledWith(sampleInvoice);
		});

		expect(mockFetch).toHaveBeenCalledWith(
			expect.stringContaining('/invoices/42/credit-note'),
			expect.objectContaining({
				method: 'POST',
				body: JSON.stringify({ reason: 'Reklamace zboží' })
			})
		);
	});

	it('displays error on API failure', async () => {
		mockFetch.mockRejectedValueOnce(new Error('Server error'));
		const onsuccess = vi.fn();

		render(CreditNoteDialog, {
			props: { invoiceId: 1, onclose: vi.fn(), onsuccess }
		});

		const input = screen.getByLabelText('Důvod dobropisu') as HTMLInputElement;
		input.removeAttribute('required');
		await fireEvent.input(input, { target: { value: 'Test reason' } });

		const submitBtn = screen.getByRole('button', { name: 'Vytvořit dobropis' });
		await fireEvent.click(submitBtn);

		await waitFor(() => {
			const alert = screen.getByRole('alert');
			expect(alert).toHaveTextContent('Server error');
		});

		expect(onsuccess).not.toHaveBeenCalled();
	});

	it('calls onclose when cancel clicked', async () => {
		const onclose = vi.fn();

		render(CreditNoteDialog, {
			props: { invoiceId: 1, onclose, onsuccess: vi.fn() }
		});

		const cancelBtn = screen.getByRole('button', { name: 'Zrušit' });
		await fireEvent.click(cancelBtn);

		expect(onclose).toHaveBeenCalled();
	});

	it('calls onclose when backdrop clicked', async () => {
		const onclose = vi.fn();

		render(CreditNoteDialog, {
			props: { invoiceId: 1, onclose, onsuccess: vi.fn() }
		});

		const backdrop = document.querySelector('[role="presentation"]')!;
		await fireEvent.click(backdrop);

		expect(onclose).toHaveBeenCalled();
	});

	it('shows validation error when reason is empty', async () => {
		render(CreditNoteDialog, {
			props: { invoiceId: 1, onclose: vi.fn(), onsuccess: vi.fn() }
		});

		const input = screen.getByLabelText('Důvod dobropisu') as HTMLInputElement;
		input.removeAttribute('required');

		// Submit with empty reason
		const form = input.closest('form')!;
		await fireEvent.submit(form);

		await waitFor(() => {
			const alert = screen.getByRole('alert');
			expect(alert).toHaveTextContent('Důvod dobropisu je povinný');
		});

		expect(mockFetch).not.toHaveBeenCalled();
	});
});

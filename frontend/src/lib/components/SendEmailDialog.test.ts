import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';
import SendEmailDialog from './SendEmailDialog.svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

beforeEach(() => {
	mockFetch.mockReset();
});

afterEach(() => {
	cleanup();
});

const defaultProps = {
	invoiceId: 42,
	invoiceNumber: 'FV-2026-001',
	customerEmail: 'zakaznik@example.com',
	onclose: vi.fn(),
	onsuccess: vi.fn()
};

describe('SendEmailDialog', () => {
	it('renders with pre-filled fields', () => {
		render(SendEmailDialog, { props: { ...defaultProps } });

		const toInput = screen.getByLabelText('Příjemce') as HTMLInputElement;
		expect(toInput.value).toBe('zakaznik@example.com');

		const subjectInput = screen.getByLabelText('Předmět') as HTMLInputElement;
		expect(subjectInput.value).toBe('Faktura FV-2026-001');

		const bodyTextarea = screen.getByLabelText('Text emailu') as HTMLTextAreaElement;
		expect(bodyTextarea.value).toContain('fakturu FV-2026-001');
		expect(bodyTextarea.value).toContain('Dobrý den');
		expect(bodyTextarea.value).toContain('S pozdravem');
	});

	it('renders with empty email when customerEmail is not provided', () => {
		render(SendEmailDialog, {
			props: {
				invoiceId: 42,
				invoiceNumber: 'FV-2026-001',
				onclose: vi.fn(),
				onsuccess: vi.fn()
			}
		});

		const toInput = screen.getByLabelText('Příjemce') as HTMLInputElement;
		expect(toInput.value).toBe('');
	});

	it('sends email on submit', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ status: 'sent' }));
		const onsuccess = vi.fn();

		render(SendEmailDialog, { props: { ...defaultProps, onsuccess } });

		// Remove required attrs to bypass HTML5 validation in jsdom
		document.querySelectorAll('[required]').forEach((el) => el.removeAttribute('required'));

		const submitBtn = screen.getByText('Odeslat');
		await fireEvent.click(submitBtn);

		await waitFor(() => {
			expect(onsuccess).toHaveBeenCalled();
		});

		expect(mockFetch).toHaveBeenCalledWith(
			'/api/v1/invoices/42/send-email',
			expect.objectContaining({
				method: 'POST',
				body: JSON.stringify({
					to: 'zakaznik@example.com',
					subject: 'Faktura FV-2026-001',
					body: 'Dobrý den,\n\nv příloze zasíláme fakturu FV-2026-001.\n\nS pozdravem'
				})
			})
		);
	});

	it('displays error on failure', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ error: 'SMTP error' }, 500));
		const onsuccess = vi.fn();

		render(SendEmailDialog, { props: { ...defaultProps, onsuccess } });

		document.querySelectorAll('[required]').forEach((el) => el.removeAttribute('required'));

		const submitBtn = screen.getByText('Odeslat');
		await fireEvent.click(submitBtn);

		await waitFor(() => {
			const alert = screen.getByRole('alert');
			expect(alert).toBeInTheDocument();
		});

		expect(onsuccess).not.toHaveBeenCalled();
	});

	it('calls onclose on cancel', async () => {
		const onclose = vi.fn();

		render(SendEmailDialog, { props: { ...defaultProps, onclose } });

		const cancelBtn = screen.getByText('Zrušit');
		await fireEvent.click(cancelBtn);

		expect(onclose).toHaveBeenCalled();
	});

	it('calls onclose on backdrop click', async () => {
		const onclose = vi.fn();

		render(SendEmailDialog, { props: { ...defaultProps, onclose } });

		const backdrop = screen.getByRole('presentation');
		await fireEvent.click(backdrop);

		expect(onclose).toHaveBeenCalled();
	});

	it('shows dialog title', () => {
		render(SendEmailDialog, { props: { ...defaultProps } });

		expect(screen.getByText('Odeslat fakturu emailem')).toBeInTheDocument();
	});

	it('has proper dialog role', () => {
		render(SendEmailDialog, { props: { ...defaultProps } });

		const dialog = screen.getByRole('dialog');
		expect(dialog).toBeInTheDocument();
		expect(dialog).toHaveAttribute('aria-modal', 'true');
	});
});

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

const defaultDefaults = {
	attach_pdf: true,
	attach_isdoc: false,
	subject: 'Faktura FV-2026-001',
	body: 'Dobrý den,\n\nv příloze zasíláme fakturu FV-2026-001.\n\nS pozdravem'
};

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

function renderWithDefaults(propsOverride = {}, defaultsOverride = {}) {
	// First call is getDefaults, chain subsequent calls as needed.
	mockFetch.mockResolvedValueOnce(jsonResponse({ ...defaultDefaults, ...defaultsOverride }));
	return render(SendEmailDialog, { props: { ...defaultProps, ...propsOverride } });
}

describe('SendEmailDialog', () => {
	it('renders with pre-filled fields after loading defaults', async () => {
		renderWithDefaults();

		await waitFor(() => {
			const subjectInput = screen.getByLabelText('Předmět') as HTMLInputElement;
			expect(subjectInput.value).toBe('Faktura FV-2026-001');
		});

		const toInput = screen.getByLabelText('Příjemce') as HTMLInputElement;
		expect(toInput.value).toBe('zakaznik@example.com');

		const bodyTextarea = screen.getByLabelText('Text emailu') as HTMLTextAreaElement;
		expect(bodyTextarea.value).toContain('fakturu FV-2026-001');
		expect(bodyTextarea.value).toContain('Dobrý den');
		expect(bodyTextarea.value).toContain('S pozdravem');
	});

	it('renders with empty email when customerEmail is not provided', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(defaultDefaults));
		render(SendEmailDialog, {
			props: {
				invoiceId: 42,
				invoiceNumber: 'FV-2026-001',
				onclose: vi.fn(),
				onsuccess: vi.fn()
			}
		});

		await waitFor(() => {
			expect((screen.getByLabelText('Předmět') as HTMLInputElement).value).toBe(
				'Faktura FV-2026-001'
			);
		});

		const toInput = screen.getByLabelText('Příjemce') as HTMLInputElement;
		expect(toInput.value).toBe('');
	});

	it('sends email with attachment flags on submit', async () => {
		renderWithDefaults();

		await waitFor(() => {
			expect((screen.getByLabelText('Předmět') as HTMLInputElement).value).toBe(
				'Faktura FV-2026-001'
			);
		});

		// Mock the send-email call
		mockFetch.mockResolvedValueOnce(jsonResponse({ status: 'sent' }));
		const onsuccess = defaultProps.onsuccess;

		document.querySelectorAll('[required]').forEach((el) => el.removeAttribute('required'));

		const submitBtn = screen.getByText('Odeslat');
		await fireEvent.click(submitBtn);

		await waitFor(() => {
			expect(onsuccess).toHaveBeenCalled();
		});

		// Second call should be send-email
		const sendCall = mockFetch.mock.calls[1];
		expect(sendCall[0]).toBe('/api/v1/invoices/42/send-email');
		const sentBody = JSON.parse(sendCall[1].body);
		expect(sentBody.attach_pdf).toBe(true);
		expect(sentBody.attach_isdoc).toBe(false);
		expect(sentBody.to).toBe('zakaznik@example.com');
		expect(sentBody.subject).toBe('Faktura FV-2026-001');
	});

	it('displays error on failure', async () => {
		renderWithDefaults();

		await waitFor(() => {
			expect((screen.getByLabelText('Předmět') as HTMLInputElement).value).toBe(
				'Faktura FV-2026-001'
			);
		});

		mockFetch.mockResolvedValueOnce(jsonResponse({ error: 'SMTP error' }, 500));

		document.querySelectorAll('[required]').forEach((el) => el.removeAttribute('required'));

		const submitBtn = screen.getByText('Odeslat');
		await fireEvent.click(submitBtn);

		await waitFor(() => {
			const alert = screen.getByRole('alert');
			expect(alert).toBeInTheDocument();
		});
	});

	it('calls onclose on cancel', async () => {
		const onclose = vi.fn();
		renderWithDefaults({ onclose });

		const cancelBtn = screen.getByText('Zrušit');
		await fireEvent.click(cancelBtn);

		expect(onclose).toHaveBeenCalled();
	});

	it('calls onclose on backdrop click', async () => {
		const onclose = vi.fn();
		renderWithDefaults({ onclose });

		const backdrop = screen.getByRole('presentation');
		await fireEvent.click(backdrop);

		expect(onclose).toHaveBeenCalled();
	});

	it('shows dialog title', () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(defaultDefaults));
		render(SendEmailDialog, { props: { ...defaultProps } });

		expect(screen.getByText('Odeslat fakturu emailem')).toBeInTheDocument();
	});

	it('has proper dialog role', () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(defaultDefaults));
		render(SendEmailDialog, { props: { ...defaultProps } });

		const dialog = screen.getByRole('dialog');
		expect(dialog).toBeInTheDocument();
		expect(dialog).toHaveAttribute('aria-modal', 'true');
	});

	it('shows attachment checkboxes', async () => {
		renderWithDefaults();

		await waitFor(() => {
			expect(screen.getByLabelText('Přiložit PDF')).toBeInTheDocument();
		});

		const pdfCheckbox = screen.getByLabelText('Přiložit PDF') as HTMLInputElement;
		const isdocCheckbox = screen.getByLabelText('Přiložit ISDOC') as HTMLInputElement;

		expect(pdfCheckbox.checked).toBe(true);
		expect(isdocCheckbox.checked).toBe(false);
	});

	it('loads defaults with both attachments enabled', async () => {
		renderWithDefaults({}, { attach_isdoc: true });

		await waitFor(() => {
			const isdocCheckbox = screen.getByLabelText('Přiložit ISDOC') as HTMLInputElement;
			expect(isdocCheckbox.checked).toBe(true);
		});
	});

	it('falls back to hardcoded defaults when API fails', async () => {
		mockFetch.mockRejectedValueOnce(new Error('Network error'));
		render(SendEmailDialog, { props: { ...defaultProps } });

		await waitFor(() => {
			const subjectInput = screen.getByLabelText('Předmět') as HTMLInputElement;
			expect(subjectInput.value).toBe('Faktura FV-2026-001');
		});
	});
});

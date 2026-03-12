import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';
import Page from './+page.svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

const sampleSettings = {
	email_attach_pdf: 'true',
	email_attach_isdoc: 'false',
	email_subject_template: 'Faktura {invoice_number}',
	email_body_template: 'Dobrý den,\n\nv příloze zasíláme fakturu {invoice_number}.\n\nS pozdravem'
};

beforeEach(() => {
	vi.useFakeTimers();
	mockFetch.mockReset();
	mockFetch.mockResolvedValue(jsonResponse(sampleSettings));
});

afterEach(() => {
	cleanup();
	vi.useRealTimers();
});

describe('Settings Email Page', () => {
	it('loads settings on mount', async () => {
		render(Page);
		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				expect.stringContaining('/api/v1/settings'),
				expect.any(Object)
			);
		});
	});

	it('renders email template fields after loading', async () => {
		render(Page);
		await waitFor(() => {
			expect(document.querySelector('#email_attach_pdf')).toBeInTheDocument();
		});
		expect(document.querySelector('#email_attach_isdoc')).toBeInTheDocument();
		expect(document.querySelector('#email_subject_template')).toBeInTheDocument();
		expect(document.querySelector('#email_body_template')).toBeInTheDocument();
	});

	it('shows email settings values', async () => {
		render(Page);
		await waitFor(() => {
			const pdfCheckbox = document.querySelector('#email_attach_pdf') as HTMLInputElement;
			expect(pdfCheckbox).toBeInTheDocument();
			expect(pdfCheckbox.checked).toBe(true);
		});
		const isdocCheckbox = document.querySelector('#email_attach_isdoc') as HTMLInputElement;
		expect(isdocCheckbox.checked).toBe(false);
	});

	it('save calls settingsApi.update (PUT /api/v1/settings)', async () => {
		render(Page);
		await waitFor(() => {
			expect(document.querySelector('#email_attach_pdf')).toBeInTheDocument();
		});

		mockFetch.mockResolvedValue(jsonResponse(sampleSettings));

		const form = document.querySelector('form')!;
		await fireEvent.submit(form);

		await waitFor(() => {
			const putCall = mockFetch.mock.calls.find(
				(call: any[]) =>
					typeof call[0] === 'string' &&
					call[0].includes('/api/v1/settings') &&
					call[1]?.method === 'PUT'
			);
			expect(putCall).toBeDefined();
		});
	});

	it('success message appears after save', async () => {
		render(Page);
		await waitFor(() => {
			expect(document.querySelector('#email_attach_pdf')).toBeInTheDocument();
		});

		mockFetch.mockResolvedValue(jsonResponse(sampleSettings));

		const form = document.querySelector('form')!;
		await fireEvent.submit(form);

		await waitFor(() => {
			expect(screen.getByText('Nastavení bylo uloženo.')).toBeInTheDocument();
		});
	});

	it('error state on load failure', async () => {
		mockFetch.mockRejectedValue(new Error('Network error'));

		render(Page);
		await waitFor(() => {
			expect(screen.getByText('Network error')).toBeInTheDocument();
		});
	});
});

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';
import Page from './+page.svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

const sampleImportResponseNoOCR = {
	expense: { id: 42, description: '', amount: 0, issue_date: '2026-03-10' },
	document: { id: 1, filename: 'receipt.pdf' }
};

const sampleImportResponseWithOCR = {
	expense: { id: 42, description: '', amount: 0, issue_date: '2026-03-10' },
	document: { id: 1, filename: 'receipt.pdf' },
	ocr: {
		vendor_name: 'Test Vendor s.r.o.',
		vendor_ico: '12345678',
		vendor_dic: 'CZ12345678',
		invoice_number: 'FV-2024-001',
		issue_date: '2024-01-15',
		due_date: '2024-02-15',
		total_amount: 12100,
		vat_amount: 2100,
		vat_rate_percent: 21,
		currency_code: 'CZK',
		description: 'Test expense',
		items: [],
		confidence: 0.92
	}
};

function createTestFile(name = 'receipt.pdf', type = 'application/pdf', sizeMB = 1): File {
	const content = new Uint8Array(sizeMB * 1024 * 1024);
	return new File([content], name, { type });
}

beforeEach(() => {
	mockFetch.mockReset();
});

afterEach(() => {
	cleanup();
});

describe('Expenses import page', () => {
	it('renders drop zone in idle state', () => {
		render(Page);

		expect(screen.getByText('Pretahni soubor sem')).toBeInTheDocument();
		expect(screen.getByText('nebo klikni pro vyber')).toBeInTheDocument();
		expect(screen.getByText('PDF, JPG, PNG, WebP (max 20 MB)')).toBeInTheDocument();
		expect(screen.getByText('Import z dokladu')).toBeInTheDocument();
	});

	it('renders back link to expenses', () => {
		render(Page);

		const backLink = screen.getByText('Zpet na naklady');
		expect(backLink).toBeInTheDocument();
		expect(backLink.closest('a')?.getAttribute('href')).toBe('/expenses');
	});

	it('renders file input with correct accept attribute', () => {
		render(Page);

		const fileInput = document.getElementById('file-input') as HTMLInputElement;
		expect(fileInput).toBeTruthy();
		expect(fileInput.type).toBe('file');
		expect(fileInput.accept).toBe('.pdf,.jpg,.jpeg,.png,.webp');
	});

	it('shows processing state and redirects when no OCR', async () => {
		vi.useFakeTimers();
		mockFetch.mockResolvedValue(jsonResponse(sampleImportResponseNoOCR));

		render(Page);

		const fileInput = document.getElementById('file-input') as HTMLInputElement;
		const file = createTestFile();

		await fireEvent.change(fileInput, { target: { files: [file] } });

		// Flush the async import response
		await vi.advanceTimersByTimeAsync(10);

		// Should call import endpoint
		expect(mockFetch).toHaveBeenCalled();
		const url = mockFetch.mock.calls[0][0] as string;
		expect(url).toContain('/api/v1/expenses/import');
		expect(mockFetch.mock.calls[0][1].method).toBe('POST');

		// Advance past the 3000ms setTimeout redirect
		await vi.advanceTimersByTimeAsync(3000);

		// Should redirect to expense detail
		const { goto } = await import('$app/navigation');
		expect(goto).toHaveBeenCalledWith('/expenses/42');

		vi.useRealTimers();
	});

	it('shows OCR review dialog when OCR result is present', async () => {
		mockFetch.mockResolvedValue(jsonResponse(sampleImportResponseWithOCR));

		render(Page);

		const fileInput = document.getElementById('file-input') as HTMLInputElement;
		const file = createTestFile();

		await fireEvent.change(fileInput, { target: { files: [file] } });

		// Should show OCR review dialog
		await waitFor(() => {
			expect(screen.getByText('OCR - Kontrola dat')).toBeInTheDocument();
		});

		// Should show OCR data in the dialog
		expect(screen.getByDisplayValue('Test Vendor s.r.o.')).toBeInTheDocument();
		expect(screen.getByDisplayValue('FV-2024-001')).toBeInTheDocument();
	});

	it('shows error message on failed upload', async () => {
		mockFetch.mockResolvedValue(jsonResponse({ error: 'Import failed' }, 500));

		render(Page);

		const fileInput = document.getElementById('file-input') as HTMLInputElement;
		const file = createTestFile();

		await fireEvent.change(fileInput, { target: { files: [file] } });

		// Should show error and return to idle state (drop zone visible again)
		await waitFor(() => {
			expect(screen.getByText('Pretahni soubor sem')).toBeInTheDocument();
		});
	});

	it('shows validation error for unsupported file type', async () => {
		render(Page);

		const fileInput = document.getElementById('file-input') as HTMLInputElement;
		const file = new File(['content'], 'document.txt', { type: 'text/plain' });

		await fireEvent.change(fileInput, { target: { files: [file] } });

		await waitFor(() => {
			expect(
				screen.getByText('Nepodporovany format souboru. Povolene: PDF, JPG, PNG, WebP.')
			).toBeInTheDocument();
		});

		// Should NOT call fetch
		expect(mockFetch).not.toHaveBeenCalled();
	});

	it('shows validation error for file exceeding 20 MB', async () => {
		render(Page);

		const fileInput = document.getElementById('file-input') as HTMLInputElement;
		const file = createTestFile('big.pdf', 'application/pdf', 21);

		await fireEvent.change(fileInput, { target: { files: [file] } });

		await waitFor(() => {
			expect(
				screen.getByText('Soubor je prilis velky. Maximum je 20 MB.')
			).toBeInTheDocument();
		});

		expect(mockFetch).not.toHaveBeenCalled();
	});

	it('OCR confirm saves data and redirects', async () => {
		// First call: import document (returns OCR)
		// Second call: update expense with OCR data
		mockFetch
			.mockResolvedValueOnce(jsonResponse(sampleImportResponseWithOCR))
			.mockResolvedValueOnce(jsonResponse({ id: 42, description: 'Test expense' }));

		render(Page);

		const fileInput = document.getElementById('file-input') as HTMLInputElement;
		const file = createTestFile();

		await fireEvent.change(fileInput, { target: { files: [file] } });

		// Wait for OCR dialog
		await waitFor(() => {
			expect(screen.getByText('OCR - Kontrola dat')).toBeInTheDocument();
		});

		// Click confirm button
		const confirmBtn = screen.getByText('Potvrdit a vyplnit');
		await fireEvent.click(confirmBtn);

		// Should call update endpoint
		await waitFor(() => {
			const updateCall = mockFetch.mock.calls.find(
				(call: unknown[]) => typeof call[0] === 'string' && call[0].includes('/api/v1/expenses/42')
			);
			expect(updateCall).toBeDefined();
		});

		// Should redirect to expense detail
		const { goto } = await import('$app/navigation');
		await waitFor(() => {
			expect(goto).toHaveBeenCalledWith('/expenses/42');
		});
	});

	it('OCR cancel redirects to expense detail', async () => {
		mockFetch.mockResolvedValue(jsonResponse(sampleImportResponseWithOCR));

		render(Page);

		const fileInput = document.getElementById('file-input') as HTMLInputElement;
		const file = createTestFile();

		await fireEvent.change(fileInput, { target: { files: [file] } });

		// Wait for OCR dialog
		await waitFor(() => {
			expect(screen.getByText('OCR - Kontrola dat')).toBeInTheDocument();
		});

		// Click cancel button
		const cancelBtn = screen.getByText('Zrusit');
		await fireEvent.click(cancelBtn);

		// Should redirect to expense detail (manual editing)
		const { goto } = await import('$app/navigation');
		await waitFor(() => {
			expect(goto).toHaveBeenCalledWith('/expenses/42');
		});
	});
});

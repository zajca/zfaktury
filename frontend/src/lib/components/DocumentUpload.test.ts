import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';
import DocumentUpload from './DocumentUpload.svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

function createFile(name: string, size: number, type: string): File {
	const buffer = new ArrayBuffer(size);
	return new File([buffer], name, { type });
}

const sampleDoc = {
	id: 1,
	expense_id: 42,
	filename: 'receipt.pdf',
	content_type: 'application/pdf',
	size: 1024,
	created_at: '2026-03-10T10:00:00Z'
};

beforeEach(() => {
	mockFetch.mockReset();
});

afterEach(() => {
	cleanup();
});

describe('DocumentUpload', () => {
	it('renders drop zone with correct text', () => {
		render(DocumentUpload, { props: { expenseId: 42 } });

		expect(screen.getByText('Přetáhněte soubor nebo klikněte pro výběr')).toBeInTheDocument();
		expect(screen.getByText(/PDF, JPG, PNG/)).toBeInTheDocument();
		expect(screen.getByText(/max 20 MB/)).toBeInTheDocument();
	});

	it('shows error for files over 20MB', async () => {
		render(DocumentUpload, { props: { expenseId: 42 } });

		const bigFile = createFile('huge.pdf', 21 * 1024 * 1024, 'application/pdf');
		const input = document.querySelector('input[type="file"]') as HTMLInputElement;

		// Simulate file selection by firing change event with files
		Object.defineProperty(input, 'files', { value: [bigFile], writable: false });
		await fireEvent.change(input);

		await waitFor(() => {
			expect(screen.getByRole('alert')).toHaveTextContent(
				'Soubor je příliš velký. Maximální velikost je 20 MB.'
			);
		});
	});

	it('shows error for invalid file types', async () => {
		render(DocumentUpload, { props: { expenseId: 42 } });

		const invalidFile = createFile('doc.txt', 100, 'text/plain');
		const input = document.querySelector('input[type="file"]') as HTMLInputElement;

		Object.defineProperty(input, 'files', { value: [invalidFile], writable: false });
		await fireEvent.change(input);

		await waitFor(() => {
			expect(screen.getByRole('alert')).toHaveTextContent(
				'Nepodporovaný typ souboru. Povolené jsou PDF, JPG a PNG.'
			);
		});
	});

	it('calls documentsApi.upload on valid file selection', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleDoc));

		render(DocumentUpload, { props: { expenseId: 42 } });

		const validFile = createFile('receipt.pdf', 1024, 'application/pdf');
		const input = document.querySelector('input[type="file"]') as HTMLInputElement;

		Object.defineProperty(input, 'files', { value: [validFile], writable: false });
		await fireEvent.change(input);

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledTimes(1);
		});

		const [url, options] = mockFetch.mock.calls[0];
		expect(url).toBe('/api/v1/expenses/42/documents');
		expect(options.method).toBe('POST');
		expect(options.body).toBeInstanceOf(FormData);
	});

	it('shows loading state during upload', async () => {
		// Never resolve to keep the upload "in progress"
		mockFetch.mockReturnValue(new Promise(() => {}));

		render(DocumentUpload, { props: { expenseId: 42 } });

		const validFile = createFile('receipt.pdf', 1024, 'application/pdf');
		const input = document.querySelector('input[type="file"]') as HTMLInputElement;

		Object.defineProperty(input, 'files', { value: [validFile], writable: false });
		await fireEvent.change(input);

		await waitFor(() => {
			expect(screen.getByText('Nahrávám...')).toBeInTheDocument();
		});

		expect(screen.getByRole('status')).toBeInTheDocument();
	});

	it('calls onuploaded callback after successful upload', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleDoc));
		const onuploaded = vi.fn();

		render(DocumentUpload, { props: { expenseId: 42, onuploaded } });

		const validFile = createFile('receipt.pdf', 1024, 'application/pdf');
		const input = document.querySelector('input[type="file"]') as HTMLInputElement;

		Object.defineProperty(input, 'files', { value: [validFile], writable: false });
		await fireEvent.change(input);

		await waitFor(() => {
			expect(onuploaded).toHaveBeenCalledTimes(1);
		});

		expect(onuploaded).toHaveBeenCalledWith(sampleDoc);
	});

	it('shows error when upload fails', async () => {
		mockFetch.mockResolvedValueOnce(
			new Response(JSON.stringify({ error: 'Upload failed' }), {
				status: 500,
				statusText: 'Internal Server Error',
				headers: { 'Content-Type': 'application/json' }
			})
		);

		render(DocumentUpload, { props: { expenseId: 42 } });

		const validFile = createFile('receipt.pdf', 1024, 'application/pdf');
		const input = document.querySelector('input[type="file"]') as HTMLInputElement;

		Object.defineProperty(input, 'files', { value: [validFile], writable: false });
		await fireEvent.change(input);

		await waitFor(() => {
			expect(screen.getByRole('alert')).toBeInTheDocument();
		});
	});
});

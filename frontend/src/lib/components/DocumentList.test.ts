import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';
import DocumentList from './DocumentList.svelte';
import type { ExpenseDocument } from '$lib/api/client';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

const sampleDocuments: ExpenseDocument[] = [
	{
		id: 1,
		expense_id: 42,
		filename: 'receipt.pdf',
		content_type: 'application/pdf',
		size: 1536,
		created_at: '2026-03-10T10:00:00Z'
	},
	{
		id: 2,
		expense_id: 42,
		filename: 'photo.jpg',
		content_type: 'image/jpeg',
		size: 2_621_440,
		created_at: '2026-03-11T14:30:00Z'
	}
];

beforeEach(() => {
	mockFetch.mockReset();
});

afterEach(() => {
	cleanup();
});

describe('DocumentList', () => {
	it('renders list of documents with filenames', () => {
		render(DocumentList, { props: { documents: sampleDocuments } });

		expect(screen.getByText('receipt.pdf')).toBeInTheDocument();
		expect(screen.getByText('photo.jpg')).toBeInTheDocument();
	});

	it('shows download links', () => {
		render(DocumentList, { props: { documents: sampleDocuments } });

		const links = screen.getAllByTitle('Stáhnout');
		expect(links).toHaveLength(2);
		expect(links[0]).toHaveAttribute('href', '/api/v1/documents/1/download');
		expect(links[1]).toHaveAttribute('href', '/api/v1/documents/2/download');
	});

	it('delete button triggers confirmation and calls delete API', async () => {
		// Mock the DELETE request (returns 204)
		mockFetch.mockResolvedValueOnce(
			new Response(null, { status: 204, statusText: 'No Content' })
		);
		const ondelete = vi.fn();

		render(DocumentList, { props: { documents: sampleDocuments, ondelete } });

		// Click delete on first document
		const deleteButtons = screen.getAllByTitle('Smazat');
		await fireEvent.click(deleteButtons[0]);

		// Confirmation dialog should appear
		expect(
			screen.getByText('Opravdu chcete smazat tento dokument?')
		).toBeInTheDocument();

		// Confirm deletion - find the confirm button in the dialog (it has different styling)
		const confirmBtns = screen.getAllByRole('button', { name: 'Smazat' });
		// The confirm button is the one in the dialog (last one added)
		const confirmBtn = confirmBtns[confirmBtns.length - 1];
		await fireEvent.click(confirmBtn);

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledTimes(1);
		});

		// Verify DELETE call
		const [url, options] = mockFetch.mock.calls[0];
		expect(url).toBe('/api/v1/documents/1');
		expect(options.method).toBe('DELETE');

		await waitFor(() => {
			expect(ondelete).toHaveBeenCalledWith(1);
		});
	});

	it('cancel button dismisses confirmation dialog', async () => {
		render(DocumentList, { props: { documents: sampleDocuments } });

		const deleteButtons = screen.getAllByTitle('Smazat');
		await fireEvent.click(deleteButtons[0]);

		expect(
			screen.getByText('Opravdu chcete smazat tento dokument?')
		).toBeInTheDocument();

		const cancelBtn = screen.getByRole('button', { name: 'Zrušit' });
		await fireEvent.click(cancelBtn);

		await waitFor(() => {
			expect(
				screen.queryByText('Opravdu chcete smazat tento dokument?')
			).not.toBeInTheDocument();
		});
	});

	it('shows OCR button when onocr prop provided', () => {
		const onocr = vi.fn();
		render(DocumentList, { props: { documents: sampleDocuments, onocr } });

		const ocrButtons = screen.getAllByRole('button', { name: 'OCR' });
		expect(ocrButtons).toHaveLength(2);
	});

	it('does not show OCR button when onocr prop not provided', () => {
		render(DocumentList, { props: { documents: sampleDocuments } });

		expect(screen.queryByRole('button', { name: 'OCR' })).not.toBeInTheDocument();
	});

	it('OCR button calls onocr with document id', async () => {
		const onocr = vi.fn();
		render(DocumentList, { props: { documents: sampleDocuments, onocr } });

		const ocrButtons = screen.getAllByRole('button', { name: 'OCR' });
		await fireEvent.click(ocrButtons[0]);

		expect(onocr).toHaveBeenCalledWith(1);
	});

	it('handles empty document list', () => {
		render(DocumentList, { props: { documents: [] } });

		expect(screen.getByText('Žádné dokumenty')).toBeInTheDocument();
	});

	it('formats file sizes correctly', () => {
		const docs: ExpenseDocument[] = [
			{
				id: 10,
				expense_id: 1,
				filename: 'small.pdf',
				content_type: 'application/pdf',
				size: 512,
				created_at: '2026-01-01T00:00:00Z'
			},
			{
				id: 11,
				expense_id: 1,
				filename: 'medium.pdf',
				content_type: 'application/pdf',
				size: 150 * 1024,
				created_at: '2026-01-01T00:00:00Z'
			},
			{
				id: 12,
				expense_id: 1,
				filename: 'large.jpg',
				content_type: 'image/jpeg',
				size: 5 * 1024 * 1024,
				created_at: '2026-01-01T00:00:00Z'
			}
		];

		render(DocumentList, { props: { documents: docs } });

		expect(screen.getByText(/512 B/)).toBeInTheDocument();
		expect(screen.getByText(/150\.0 KB/)).toBeInTheDocument();
		expect(screen.getByText(/5\.0 MB/)).toBeInTheDocument();
	});
});

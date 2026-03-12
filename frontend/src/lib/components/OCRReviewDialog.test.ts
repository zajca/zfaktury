import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, cleanup } from '@testing-library/svelte';
import OCRReviewDialog from './OCRReviewDialog.svelte';
import type { OCRResult } from '$lib/api/client';

function makeOCRResult(overrides: Partial<OCRResult> = {}): OCRResult {
	return {
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
		confidence: 0.92,
		...overrides
	};
}

beforeEach(() => {});
afterEach(() => {
	cleanup();
});

describe('OCRReviewDialog', () => {
	it('renders with OCR data in form fields', () => {
		const result = makeOCRResult();
		render(OCRReviewDialog, {
			props: { ocrResult: result, onclose: vi.fn(), onconfirm: vi.fn() }
		});

		expect(screen.getByText('OCR - Kontrola dat')).toBeInTheDocument();

		const vendorInput = screen.getByLabelText('Dodavatel') as HTMLInputElement;
		expect(vendorInput.value).toBe('Test Vendor s.r.o.');

		const icoInput = screen.getByLabelText('IČO dodavatele') as HTMLInputElement;
		expect(icoInput.value).toBe('12345678');

		const invoiceNumberInput = screen.getByLabelText('Číslo faktury') as HTMLInputElement;
		expect(invoiceNumberInput.value).toBe('FV-2024-001');

		const issueDateInput = screen.getByLabelText('Datum vystavení') as HTMLInputElement;
		expect(issueDateInput.value).toBe('2024-01-15');

		const dueDateInput = screen.getByLabelText('Datum splatnosti') as HTMLInputElement;
		expect(dueDateInput.value).toBe('2024-02-15');

		const totalInput = screen.getByLabelText('Celková částka') as HTMLInputElement;
		expect(totalInput.value).toBe('12100');

		const vatInput = screen.getByLabelText('DPH') as HTMLInputElement;
		expect(vatInput.value).toBe('2100');

		const currencyInput = screen.getByLabelText('Měna') as HTMLInputElement;
		expect(currencyInput.value).toBe('CZK');
	});

	it('shows confidence percentage with green color for high confidence', () => {
		render(OCRReviewDialog, {
			props: {
				ocrResult: makeOCRResult({ confidence: 0.92 }),
				onclose: vi.fn(),
				onconfirm: vi.fn()
			}
		});

		const badge = screen.getByTestId('confidence');
		expect(badge.textContent).toContain('92');
		expect(badge.className).toContain('text-success');
	});

	it('shows confidence percentage with yellow color for medium confidence', () => {
		render(OCRReviewDialog, {
			props: {
				ocrResult: makeOCRResult({ confidence: 0.65 }),
				onclose: vi.fn(),
				onconfirm: vi.fn()
			}
		});

		const badge = screen.getByTestId('confidence');
		expect(badge.textContent).toContain('65');
		expect(badge.className).toContain('text-warning');
	});

	it('shows confidence percentage with red color for low confidence', () => {
		render(OCRReviewDialog, {
			props: { ocrResult: makeOCRResult({ confidence: 0.3 }), onclose: vi.fn(), onconfirm: vi.fn() }
		});

		const badge = screen.getByTestId('confidence');
		expect(badge.textContent).toContain('30');
		expect(badge.className).toContain('text-danger');
	});

	it('calls onconfirm with edited data when confirm button clicked', async () => {
		const onconfirm = vi.fn();
		const result = makeOCRResult();
		render(OCRReviewDialog, {
			props: { ocrResult: result, onclose: vi.fn(), onconfirm }
		});

		// Edit vendor name
		const vendorInput = screen.getByLabelText('Dodavatel') as HTMLInputElement;
		await fireEvent.input(vendorInput, { target: { value: 'Updated Vendor' } });

		// Edit currency
		const currencyInput = screen.getByLabelText('Měna') as HTMLInputElement;
		await fireEvent.input(currencyInput, { target: { value: 'EUR' } });

		// Click confirm
		const confirmBtn = screen.getByText('Potvrdit a vyplnit');
		await fireEvent.click(confirmBtn);

		expect(onconfirm).toHaveBeenCalledOnce();
		const calledWith = onconfirm.mock.calls[0][0] as OCRResult;
		expect(calledWith.vendor_name).toBe('Updated Vendor');
		expect(calledWith.currency_code).toBe('EUR');
		// Unchanged fields should remain
		expect(calledWith.vendor_ico).toBe('12345678');
	});

	it('calls onclose on cancel button click', async () => {
		const onclose = vi.fn();
		render(OCRReviewDialog, {
			props: { ocrResult: makeOCRResult(), onclose, onconfirm: vi.fn() }
		});

		const cancelBtn = screen.getByText('Zrušit');
		await fireEvent.click(cancelBtn);

		expect(onclose).toHaveBeenCalledOnce();
	});

	it('calls onclose on backdrop click', async () => {
		const onclose = vi.fn();
		render(OCRReviewDialog, {
			props: { ocrResult: makeOCRResult(), onclose, onconfirm: vi.fn() }
		});

		const backdrop = document.querySelector('[role="presentation"]') as HTMLElement;
		expect(backdrop).toBeTruthy();
		await fireEvent.click(backdrop);

		expect(onclose).toHaveBeenCalledOnce();
	});
});

// API client for VAT Control Statements

import { get, post, del, ApiError, type TaxPeriod } from './client';

const API_BASE = '/api/v1';

// --- Types ---

export type { TaxPeriod };

export interface ControlStatementLine {
	id: number;
	section: string; // "A4", "A5", "B2", "B3"
	partner_dic: string;
	document_number: string;
	dppd: string;
	base: number; // halere
	vat: number; // halere
	vat_rate_percent: number;
	invoice_id: number | null;
	expense_id: number | null;
}

export interface ControlStatement {
	id: number;
	period: TaxPeriod;
	filing_type: string;
	lines: ControlStatementLine[] | null;
	has_xml: boolean;
	status: string;
	filed_at: string | null;
	created_at: string;
	updated_at: string;
}

// --- API ---

export const controlStatementApi = {
	list(year?: number) {
		const query = year ? `?year=${year}` : '';
		return get<ControlStatement[]>(`/vat-control-statements${query}`);
	},

	getById(id: number) {
		return get<ControlStatement>(`/vat-control-statements/${id}`);
	},

	create(data: { year: number; month: number; filing_type?: string }) {
		return post<ControlStatement>('/vat-control-statements', data);
	},

	delete(id: number) {
		return del<void>(`/vat-control-statements/${id}`);
	},

	recalculate(id: number) {
		return post<ControlStatement>(`/vat-control-statements/${id}/recalculate`);
	},

	generateXml(id: number) {
		return post<ControlStatement>(`/vat-control-statements/${id}/generate-xml`);
	},

	async downloadXml(id: number): Promise<Blob> {
		const url = `${API_BASE}/vat-control-statements/${id}/xml`;
		const response = await fetch(url, { method: 'GET' });
		if (!response.ok) {
			let body: unknown;
			try { body = await response.json(); } catch { /* ignore */ }
			throw new ApiError(response.status, response.statusText, body);
		}
		return response.blob();
	},

	markFiled(id: number) {
		return post<ControlStatement>(`/vat-control-statements/${id}/mark-filed`);
	}
};

// API client for VIES Summaries

import { get, post, del, ApiError, type TaxPeriod } from './client';

const API_BASE = '/api/v1';

// --- Types ---

export type { TaxPeriod };

export interface VIESSummaryLine {
	id: number;
	partner_dic: string;
	country_code: string;
	total_amount: number; // halere
	service_code: string;
}

export interface VIESSummary {
	id: number;
	period: TaxPeriod;
	filing_type: string;
	lines: VIESSummaryLine[] | null;
	has_xml: boolean;
	status: string;
	filed_at: string | null;
	created_at: string;
	updated_at: string;
}

// --- API ---

export const viesApi = {
	list(year?: number) {
		const query = year ? `?year=${year}` : '';
		return get<VIESSummary[]>(`/vies-summaries${query}`);
	},

	getById(id: number) {
		return get<VIESSummary>(`/vies-summaries/${id}`);
	},

	create(data: { year: number; quarter: number; filing_type?: string }) {
		return post<VIESSummary>('/vies-summaries', data);
	},

	delete(id: number) {
		return del<void>(`/vies-summaries/${id}`);
	},

	recalculate(id: number) {
		return post<VIESSummary>(`/vies-summaries/${id}/recalculate`);
	},

	generateXml(id: number) {
		return post<VIESSummary>(`/vies-summaries/${id}/generate-xml`);
	},

	async downloadXml(id: number): Promise<Blob> {
		const url = `${API_BASE}/vies-summaries/${id}/xml`;
		const response = await fetch(url, { method: 'GET' });
		if (!response.ok) {
			let body: unknown;
			try { body = await response.json(); } catch { /* ignore */ }
			throw new ApiError(response.status, response.statusText, body);
		}
		return response.blob();
	},

	markFiled(id: number) {
		return post<VIESSummary>(`/vies-summaries/${id}/mark-filed`);
	}
};

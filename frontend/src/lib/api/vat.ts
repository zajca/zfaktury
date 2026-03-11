// VAT Returns API client

import { get, post, del, ApiError, type TaxPeriod } from './client';

const API_BASE = '/api/v1';

// --- Types ---

export type { TaxPeriod };

export interface VATReturn {
	id: number;
	period: TaxPeriod;
	filing_type: string;
	output_vat_base_21: number;
	output_vat_amount_21: number;
	output_vat_base_12: number;
	output_vat_amount_12: number;
	output_vat_base_0: number;
	reverse_charge_base_21: number;
	reverse_charge_amount_21: number;
	reverse_charge_base_12: number;
	reverse_charge_amount_12: number;
	input_vat_base_21: number;
	input_vat_amount_21: number;
	input_vat_base_12: number;
	input_vat_amount_12: number;
	total_output_vat: number;
	total_input_vat: number;
	net_vat: number;
	has_xml: boolean;
	status: string;
	filed_at: string | null;
	created_at: string;
	updated_at: string;
}

// --- API ---

export const vatReturnApi = {
	list(year?: number): Promise<VATReturn[]> {
		const query = year ? `?year=${year}` : '';
		return get<VATReturn[]>(`/vat-returns${query}`);
	},

	getById(id: number): Promise<VATReturn> {
		return get<VATReturn>(`/vat-returns/${id}`);
	},

	create(data: { year: number; month?: number; quarter?: number; filing_type?: string }): Promise<VATReturn> {
		return post<VATReturn>('/vat-returns', data);
	},

	delete(id: number): Promise<void> {
		return del(`/vat-returns/${id}`);
	},

	recalculate(id: number): Promise<VATReturn> {
		return post<VATReturn>(`/vat-returns/${id}/recalculate`, {});
	},

	generateXml(id: number): Promise<VATReturn> {
		return post<VATReturn>(`/vat-returns/${id}/generate-xml`, {});
	},

	async downloadXml(id: number): Promise<Blob> {
		const res = await fetch(`${API_BASE}/vat-returns/${id}/xml`);
		if (!res.ok) {
			let body: unknown;
			try { body = await res.json(); } catch { /* ignore */ }
			throw new ApiError(res.status, res.statusText, body);
		}
		return res.blob();
	},

	markFiled(id: number): Promise<VATReturn> {
		return post<VATReturn>(`/vat-returns/${id}/mark-filed`, {});
	}
};

// API client for VAT Control Statements

const API_BASE = '/api/v1';

class ApiError extends Error {
	constructor(
		public status: number,
		public statusText: string,
		public body?: unknown
	) {
		const detail =
			body && typeof body === 'object' && 'error' in body && typeof (body as Record<string, unknown>).error === 'string'
				? (body as Record<string, string>).error
				: null;
		super(detail ?? `API Error ${status}: ${statusText}`);
		this.name = 'ApiError';
	}
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
	const url = `${API_BASE}${path}`;
	const headers: Record<string, string> = {
		'Content-Type': 'application/json',
		...(options?.headers as Record<string, string>)
	};

	const response = await fetch(url, { ...options, headers });

	if (!response.ok) {
		let body: unknown;
		try {
			body = await response.json();
		} catch {
			// response body is not JSON
		}
		throw new ApiError(response.status, response.statusText, body);
	}

	if (response.status === 204) {
		return undefined as T;
	}

	return response.json();
}

function get<T>(path: string): Promise<T> {
	return request<T>(path, { method: 'GET' });
}

function post<T>(path: string, body?: unknown): Promise<T> {
	return request<T>(path, { method: 'POST', body: body ? JSON.stringify(body) : undefined });
}

function del<T>(path: string): Promise<T> {
	return request<T>(path, { method: 'DELETE' });
}

// --- Types ---

export interface TaxPeriod {
	year: number;
	month: number;
	quarter: number;
}

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
			throw new ApiError(response.status, response.statusText);
		}
		return response.blob();
	},

	markFiled(id: number) {
		return post<ControlStatement>(`/vat-control-statements/${id}/mark-filed`);
	}
};

// API client for VIES Summaries

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
			throw new ApiError(response.status, response.statusText);
		}
		return response.blob();
	},

	markFiled(id: number) {
		return post<VIESSummary>(`/vies-summaries/${id}/mark-filed`);
	}
};

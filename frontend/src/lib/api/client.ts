// API client for ZFaktury backend

// --- Domain Types ---

export interface Contact {
	id: number;
	type: 'company' | 'individual';
	name: string;
	ico: string;
	dic: string;
	street: string;
	city: string;
	zip: string;
	country: string;
	email: string;
	phone: string;
	web: string;
	bank_account: string;
	bank_code: string;
	iban: string;
	swift: string;
	payment_terms_days: number;
	tags: string;
	notes: string;
	is_favorite: boolean;
	vat_unreliable_at: string | null;
	created_at: string;
	updated_at: string;
}

export interface InvoiceItem {
	id: number;
	invoice_id: number;
	description: string;
	quantity: number;
	unit: string;
	unit_price: number;
	vat_rate_percent: number;
	vat_amount: number;
	total_amount: number;
	sort_order: number;
}

export interface Invoice {
	id: number;
	sequence_id: number;
	invoice_number: string;
	type: 'regular' | 'proforma' | 'credit_note';
	status: 'draft' | 'sent' | 'paid' | 'overdue' | 'cancelled';
	issue_date: string;
	due_date: string;
	delivery_date: string;
	variable_symbol: string;
	constant_symbol: string;
	customer_id: number;
	customer?: Contact;
	currency_code: string;
	exchange_rate: number;
	payment_method: string;
	bank_account: string;
	bank_code: string;
	iban: string;
	swift: string;
	subtotal_amount: number;
	vat_amount: number;
	total_amount: number;
	paid_amount: number;
	notes: string;
	internal_notes: string;
	sent_at?: string;
	paid_at?: string;
	items: InvoiceItem[];
	created_at: string;
	updated_at: string;
}

export interface Expense {
	id: number;
	vendor_id?: number;
	vendor?: Contact;
	expense_number: string;
	category: string;
	description: string;
	issue_date: string;
	amount: number;
	currency_code: string;
	exchange_rate: number;
	vat_rate_percent: number;
	vat_amount: number;
	is_tax_deductible: boolean;
	business_percent: number;
	payment_method: string;
	document_path?: string;
	notes: string;
	created_at: string;
	updated_at: string;
}

// --- Invoice Sequence Types ---

export interface InvoiceSequence {
	id: number;
	prefix: string;
	next_number: number;
	year: number;
	format_pattern: string;
	preview: string;
}

// --- Expense Category Types ---

export interface ExpenseCategory {
	id: number;
	key: string;
	label_cs: string;
	label_en: string;
	color: string;
	sort_order: number;
	is_default: boolean;
	created_at: string;
}

export interface ListResponse<T> {
	data: T[];
	total: number;
	limit: number;
	offset: number;
}

export interface AresResult {
	ico: string;
	dic: string;
	name: string;
	street: string;
	city: string;
	zip: string;
	country: string;
}

// --- Settings Types ---

export interface Settings {
	[key: string]: string;
}

// --- API Error ---

export class ApiError extends Error {
	constructor(
		public status: number,
		public statusText: string,
		public body?: unknown
	) {
		super(`API Error ${status}: ${statusText}`);
		this.name = 'ApiError';
	}
}

// --- Fetch Wrapper ---

const API_BASE = '/api/v1';

async function request<T>(path: string, options?: RequestInit): Promise<T> {
	const url = `${API_BASE}${path}`;
	const headers: Record<string, string> = {
		'Content-Type': 'application/json',
		...(options?.headers as Record<string, string>)
	};

	const response = await fetch(url, {
		...options,
		headers
	});

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

function post<T>(path: string, body: unknown): Promise<T> {
	return request<T>(path, { method: 'POST', body: JSON.stringify(body) });
}

function put<T>(path: string, body: unknown): Promise<T> {
	return request<T>(path, { method: 'PUT', body: JSON.stringify(body) });
}

function del<T>(path: string): Promise<T> {
	return request<T>(path, { method: 'DELETE' });
}

// --- Contacts API ---

export const contactsApi = {
	list(params?: { limit?: number; offset?: number; search?: string }) {
		const query = new URLSearchParams();
		if (params?.limit) query.set('limit', String(params.limit));
		if (params?.offset) query.set('offset', String(params.offset));
		if (params?.search) query.set('search', params.search);
		const qs = query.toString();
		return get<ListResponse<Contact>>(`/contacts${qs ? `?${qs}` : ''}`);
	},

	getById(id: number) {
		return get<Contact>(`/contacts/${id}`);
	},

	create(data: Partial<Contact>) {
		return post<Contact>('/contacts', data);
	},

	update(id: number, data: Partial<Contact>) {
		return put<Contact>(`/contacts/${id}`, data);
	},

	delete(id: number) {
		return del<void>(`/contacts/${id}`);
	},

	lookupAres(ico: string) {
		return get<AresResult>(`/contacts/ares/${ico}`);
	}
};

// --- Invoices API ---

export const invoicesApi = {
	list(params?: { limit?: number; offset?: number; search?: string; status?: string }) {
		const query = new URLSearchParams();
		if (params?.limit) query.set('limit', String(params.limit));
		if (params?.offset) query.set('offset', String(params.offset));
		if (params?.search) query.set('search', params.search);
		if (params?.status) query.set('status', params.status);
		const qs = query.toString();
		return get<ListResponse<Invoice>>(`/invoices${qs ? `?${qs}` : ''}`);
	},

	getById(id: number) {
		return get<Invoice>(`/invoices/${id}`);
	},

	create(data: Partial<Invoice>) {
		return post<Invoice>('/invoices', data);
	},

	update(id: number, data: Partial<Invoice>) {
		return put<Invoice>(`/invoices/${id}`, data);
	},

	delete(id: number) {
		return del<void>(`/invoices/${id}`);
	},

	send(id: number) {
		return post<Invoice>(`/invoices/${id}/send`, {});
	},

	markPaid(id: number, amount?: number, paidAt?: string) {
		return post<Invoice>(`/invoices/${id}/mark-paid`, { amount, paid_at: paidAt });
	},

	duplicate(id: number) {
		return post<Invoice>(`/invoices/${id}/duplicate`, {});
	},

	getPdfUrl(id: number): string {
		return `${API_BASE}/invoices/${id}/pdf`;
	},

	getQrUrl(id: number): string {
		return `${API_BASE}/invoices/${id}/qr`;
	},

	getIsdocUrl(id: number): string {
		return `${API_BASE}/invoices/${id}/isdoc`;
	},

	async exportIsdocBatch(invoiceIds: number[]): Promise<Blob> {
		const url = `${API_BASE}/invoices/export/isdoc`;
		const response = await fetch(url, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ invoice_ids: invoiceIds })
		});

		if (!response.ok) {
			throw new ApiError(response.status, response.statusText);
		}

		return response.blob();
	}
};

// --- Expenses API ---

export const expensesApi = {
	list(params?: { limit?: number; offset?: number; search?: string }) {
		const query = new URLSearchParams();
		if (params?.limit) query.set('limit', String(params.limit));
		if (params?.offset) query.set('offset', String(params.offset));
		if (params?.search) query.set('search', params.search);
		const qs = query.toString();
		return get<ListResponse<Expense>>(`/expenses${qs ? `?${qs}` : ''}`);
	},

	getById(id: number) {
		return get<Expense>(`/expenses/${id}`);
	},

	create(data: Partial<Expense>) {
		return post<Expense>('/expenses', data);
	},

	update(id: number, data: Partial<Expense>) {
		return put<Expense>(`/expenses/${id}`, data);
	},

	delete(id: number) {
		return del<void>(`/expenses/${id}`);
	}
};

// --- Invoice Sequences API ---

export const sequencesApi = {
	list() {
		return get<InvoiceSequence[]>('/invoice-sequences');
	},

	getById(id: number) {
		return get<InvoiceSequence>(`/invoice-sequences/${id}`);
	},

	create(data: Partial<InvoiceSequence>) {
		return post<InvoiceSequence>('/invoice-sequences', data);
	},

	update(id: number, data: Partial<InvoiceSequence>) {
		return put<InvoiceSequence>(`/invoice-sequences/${id}`, data);
	},

	delete(id: number) {
		return del<void>(`/invoice-sequences/${id}`);
	}
};

// --- Expense Categories API ---

export const categoriesApi = {
	list() {
		return get<ExpenseCategory[]>('/expense-categories');
	},

	create(data: Partial<ExpenseCategory>) {
		return post<ExpenseCategory>('/expense-categories', data);
	},

	update(id: number, data: Partial<ExpenseCategory>) {
		return put<ExpenseCategory>(`/expense-categories/${id}`, data);
	},

	delete(id: number) {
		return del<void>(`/expense-categories/${id}`);
	}
};

// --- Settings API ---

export const settingsApi = {
	getAll() {
		return get<Settings>('/settings');
	},

	update(settings: Settings) {
		return put<Settings>('/settings', settings);
	}
};

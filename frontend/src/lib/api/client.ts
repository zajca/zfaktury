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
	related_invoice_id?: number;
	relation_type?: string;
	related_invoices?: RelatedInvoice[];
	sent_at?: string;
	paid_at?: string;
	items: InvoiceItem[];
	created_at: string;
	updated_at: string;
}

export interface RelatedInvoice {
	id: number;
	invoice_number: string;
	type: string;
	relation_type: string;
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
	tax_reviewed_at?: string;
	created_at: string;
	updated_at: string;
}

export interface ExpenseDocument {
	id: number;
	expense_id: number;
	filename: string;
	content_type: string;
	size: number;
	created_at: string;
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

// --- Recurring Invoice Types ---

export interface RecurringInvoiceItem {
	id: number;
	recurring_invoice_id: number;
	description: string;
	quantity: number;
	unit: string;
	unit_price: number;
	vat_rate_percent: number;
	sort_order: number;
}

export interface RecurringInvoice {
	id: number;
	name: string;
	customer_id: number;
	customer?: Contact;
	frequency: 'weekly' | 'monthly' | 'quarterly' | 'yearly';
	next_issue_date: string;
	end_date?: string;
	currency_code: string;
	exchange_rate: number;
	payment_method: string;
	bank_account: string;
	bank_code: string;
	iban: string;
	swift: string;
	constant_symbol: string;
	notes: string;
	is_active: boolean;
	items: RecurringInvoiceItem[];
	created_at: string;
	updated_at: string;
}

// --- Recurring Expense Types ---

export interface RecurringExpense {
	id: number;
	name: string;
	vendor_id?: number;
	vendor?: Contact;
	category: string;
	description: string;
	amount: number;
	currency_code: string;
	exchange_rate: number;
	vat_rate_percent: number;
	vat_amount: number;
	is_tax_deductible: boolean;
	business_percent: number;
	payment_method: string;
	notes: string;
	frequency: 'weekly' | 'monthly' | 'quarterly' | 'yearly';
	next_issue_date: string;
	end_date?: string;
	is_active: boolean;
	created_at: string;
	updated_at: string;
}

// --- OCR Types ---

export interface OCRItem {
	description: string;
	quantity: number;
	unit_price: number;
	vat_rate_percent: number;
	total_amount: number;
}

export interface OCRResult {
	vendor_name: string;
	vendor_ico: string;
	vendor_dic: string;
	invoice_number: string;
	issue_date: string;
	due_date: string;
	total_amount: number;
	vat_amount: number;
	vat_rate_percent: number;
	currency_code: string;
	description: string;
	items: OCRItem[];
	confidence: number;
}

// --- Settings Types ---

export interface Settings {
	[key: string]: string;
}

// --- API Error ---

// --- Tax Period ---

export interface TaxPeriod {
	year: number;
	month: number;
	quarter: number;
}

export class ApiError extends Error {
	constructor(
		public status: number,
		public statusText: string,
		public body?: unknown
	) {
		const detail =
			body &&
			typeof body === 'object' &&
			'error' in body &&
			typeof (body as Record<string, unknown>).error === 'string'
				? (body as Record<string, string>).error
				: null;
		super(detail ?? `API Error ${status}: ${statusText}`);
		this.name = 'ApiError';
	}
}

// --- Fetch Wrapper ---

const API_BASE = '/api/v1';

export async function request<T>(path: string, options?: RequestInit): Promise<T> {
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

export function get<T>(path: string): Promise<T> {
	return request<T>(path, { method: 'GET' });
}

export function post<T>(path: string, body?: unknown): Promise<T> {
	return request<T>(path, {
		method: 'POST',
		body: body != null ? JSON.stringify(body) : undefined
	});
}

export function put<T>(path: string, body: unknown): Promise<T> {
	return request<T>(path, { method: 'PUT', body: JSON.stringify(body) });
}

export function del<T>(path: string): Promise<T> {
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
	list(params?: { limit?: number; offset?: number; search?: string; status?: string; type?: string }) {
		const query = new URLSearchParams();
		if (params?.limit) query.set('limit', String(params.limit));
		if (params?.offset) query.set('offset', String(params.offset));
		if (params?.search) query.set('search', params.search);
		if (params?.status) query.set('status', params.status);
		if (params?.type) query.set('type', params.type);
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

	settle(id: number) {
		return post<Invoice>(`/invoices/${id}/settle`, {});
	},

	createCreditNote(id: number, data: CreditNoteRequest) {
		return post<Invoice>(`/invoices/${id}/credit-note`, data);
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

	sendEmail(id: number, data: { to: string; subject: string; body: string; attach_pdf: boolean; attach_isdoc: boolean }) {
		return post<{ status: string }>(`/invoices/${id}/send-email`, data);
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

// --- Email Defaults API ---

export interface EmailDefaults {
	attach_pdf: boolean;
	attach_isdoc: boolean;
	subject: string;
	body: string;
}

export const emailApi = {
	getDefaults(invoiceNumber: string) {
		const query = new URLSearchParams({ invoice_number: invoiceNumber });
		return get<EmailDefaults>(`/email/defaults?${query.toString()}`);
	}
};

// --- Expenses API ---

export const expensesApi = {
	list(params?: {
		limit?: number;
		offset?: number;
		search?: string;
		date_from?: string;
		date_to?: string;
		tax_reviewed?: string;
	}) {
		const query = new URLSearchParams();
		if (params?.limit) query.set('limit', String(params.limit));
		if (params?.offset) query.set('offset', String(params.offset));
		if (params?.search) query.set('search', params.search);
		if (params?.date_from) query.set('date_from', params.date_from);
		if (params?.date_to) query.set('date_to', params.date_to);
		if (params?.tax_reviewed) query.set('tax_reviewed', params.tax_reviewed);
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
	},

	markTaxReviewed(ids: number[]) {
		return post<void>('/expenses/review', { ids });
	},

	unmarkTaxReviewed(ids: number[]) {
		return post<void>('/expenses/unreview', { ids });
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

// --- Documents API ---

export const documentsApi = {
	listByExpense(expenseId: number) {
		return get<ExpenseDocument[]>(`/expenses/${expenseId}/documents`);
	},

	getById(id: number) {
		return get<ExpenseDocument>(`/documents/${id}`);
	},

	async upload(expenseId: number, file: File): Promise<ExpenseDocument> {
		const formData = new FormData();
		formData.append('file', file);
		const url = `${API_BASE}/expenses/${expenseId}/documents`;
		const response = await fetch(url, { method: 'POST', body: formData });
		if (!response.ok) {
			let body: unknown;
			try {
				body = await response.json();
			} catch {
				/* ignore */
			}
			throw new ApiError(response.status, response.statusText, body);
		}
		return response.json();
	},

	getDownloadUrl(id: number): string {
		return `${API_BASE}/documents/${id}/download`;
	},

	delete(id: number) {
		return del<void>(`/documents/${id}`);
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

// --- Recurring Invoices API ---

export const recurringInvoicesApi = {
	list() {
		return get<RecurringInvoice[]>('/recurring-invoices');
	},

	getById(id: number) {
		return get<RecurringInvoice>(`/recurring-invoices/${id}`);
	},

	create(data: Partial<RecurringInvoice>) {
		return post<RecurringInvoice>('/recurring-invoices', data);
	},

	update(id: number, data: Partial<RecurringInvoice>) {
		return put<RecurringInvoice>(`/recurring-invoices/${id}`, data);
	},

	delete(id: number) {
		return del<void>(`/recurring-invoices/${id}`);
	},

	generate(id: number) {
		return post<Invoice>(`/recurring-invoices/${id}/generate`, {});
	},

	processDue() {
		return post<{ generated_count: number }>('/recurring-invoices/process-due', {});
	}
};

// --- Recurring Expenses API ---

export const recurringExpensesApi = {
	list(params?: { limit?: number; offset?: number }) {
		const query = new URLSearchParams();
		if (params?.limit) query.set('limit', String(params.limit));
		if (params?.offset) query.set('offset', String(params.offset));
		const qs = query.toString();
		return get<ListResponse<RecurringExpense>>(`/recurring-expenses${qs ? `?${qs}` : ''}`);
	},

	getById(id: number) {
		return get<RecurringExpense>(`/recurring-expenses/${id}`);
	},

	create(data: Partial<RecurringExpense>) {
		return post<RecurringExpense>('/recurring-expenses', data);
	},

	update(id: number, data: Partial<RecurringExpense>) {
		return put<RecurringExpense>(`/recurring-expenses/${id}`, data);
	},

	delete(id: number) {
		return del<void>(`/recurring-expenses/${id}`);
	},

	activate(id: number) {
		return post<void>(`/recurring-expenses/${id}/activate`, {});
	},

	deactivate(id: number) {
		return post<void>(`/recurring-expenses/${id}/deactivate`, {});
	},

	generate(asOfDate?: string) {
		return post<{ generated: number }>('/recurring-expenses/generate', {
			as_of_date: asOfDate || ''
		});
	}
};

// --- OCR API ---

export const ocrApi = {
	processDocument(documentId: number) {
		return post<OCRResult>(`/documents/${documentId}/ocr`, {});
	}
};

// --- Credit Note API (on invoices) ---

export interface CreditNoteRequest {
	items?: Array<{
		description: string;
		quantity: number;
		unit: string;
		unit_price: number;
		vat_rate_percent: number;
		sort_order: number;
	}>;
	reason: string;
}

// --- Exchange Rate API ---

export interface ExchangeRateResult {
	currency_code: string;
	rate: number;
	date: string;
}

export const exchangeRateApi = {
	getRate(currency: string, date?: string) {
		const query = new URLSearchParams({ currency });
		if (date) query.set('date', date);
		return get<ExchangeRateResult>(`/exchange-rate?${query.toString()}`);
	}
};

// --- Status History API ---

export interface InvoiceStatusChange {
	id: number;
	invoice_id: number;
	old_status: string;
	new_status: string;
	changed_at: string;
	note: string;
}

export const statusHistoryApi = {
	getHistory(invoiceId: number) {
		return get<InvoiceStatusChange[]>(`/invoices/${invoiceId}/history`);
	},

	checkOverdue() {
		return post<{ marked: number }>('/invoices/check-overdue', {});
	}
};

// --- Payment Reminder API ---

export interface PaymentReminder {
	id: number;
	invoice_id: number;
	reminder_number: number;
	sent_at: string;
	sent_to: string;
	subject: string;
	body_preview: string;
	created_at: string;
}

export const remindersApi = {
	sendReminder(invoiceId: number) {
		return post<PaymentReminder>(`/invoices/${invoiceId}/remind`, {});
	},

	listReminders(invoiceId: number) {
		return get<PaymentReminder[]>(`/invoices/${invoiceId}/reminders`);
	}
};

// --- VAT Return Types ---

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

// --- VAT Control Statement Types ---

export interface ControlStatementLine {
	id: number;
	section: string;
	partner_dic: string;
	document_number: string;
	dppd: string;
	base: number;
	vat: number;
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

// --- VIES Summary Types ---

export interface VIESSummaryLine {
	id: number;
	partner_dic: string;
	country_code: string;
	total_amount: number;
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

// --- VAT Returns API ---

export const vatReturnApi = {
	list(year?: number): Promise<VATReturn[]> {
		const query = year ? `?year=${year}` : '';
		return get<VATReturn[]>(`/vat-returns${query}`);
	},

	getById(id: number): Promise<VATReturn> {
		return get<VATReturn>(`/vat-returns/${id}`);
	},

	create(data: {
		year: number;
		month?: number;
		quarter?: number;
		filing_type?: string;
	}): Promise<VATReturn> {
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
			try {
				body = await res.json();
			} catch {
				/* ignore */
			}
			throw new ApiError(res.status, res.statusText, body);
		}
		return res.blob();
	},

	markFiled(id: number): Promise<VATReturn> {
		return post<VATReturn>(`/vat-returns/${id}/mark-filed`, {});
	}
};

// --- VAT Control Statements API ---

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
			try {
				body = await response.json();
			} catch {
				/* ignore */
			}
			throw new ApiError(response.status, response.statusText, body);
		}
		return response.blob();
	},

	markFiled(id: number) {
		return post<ControlStatement>(`/vat-control-statements/${id}/mark-filed`);
	}
};

// --- VIES Summaries API ---

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
			try {
				body = await response.json();
			} catch {
				/* ignore */
			}
			throw new ApiError(response.status, response.statusText, body);
		}
		return response.blob();
	},

	markFiled(id: number) {
		return post<VIESSummary>(`/vies-summaries/${id}/mark-filed`);
	}
};

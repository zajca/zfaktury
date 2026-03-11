// Shared invoice status labels and colors for consistent UI across pages

export const statusLabels: Record<string, string> = {
	draft: 'Koncept',
	sent: 'Odeslan\u00e1',
	paid: 'Uhrazen\u00e1',
	overdue: 'Po splatnosti',
	cancelled: 'Stornovan\u00e1'
};

export const statusColors: Record<string, string> = {
	draft: 'bg-elevated text-secondary',
	sent: 'bg-info-bg text-info',
	paid: 'bg-success-bg text-success',
	overdue: 'bg-danger-bg text-danger',
	cancelled: 'bg-elevated text-muted'
};

export type StatusVariant = 'default' | 'info' | 'success' | 'danger' | 'muted';

export const paymentMethodLabels: Record<string, string> = {
	bank_transfer: 'Bankovní převod',
	cash: 'Hotovost',
	card: 'Karta'
};

export const frequencyLabels: Record<string, string> = {
	weekly: 'Týdně',
	monthly: 'Měsíčně',
	quarterly: 'Čtvrtletně',
	yearly: 'Ročně'
};

export const invoiceTypeLabels: Record<string, string> = {
	regular: 'Faktura',
	proforma: 'Zálohová faktura',
	credit_note: 'Dobropis'
};

export const statusVariant: Record<string, StatusVariant> = {
	draft: 'default',
	sent: 'info',
	paid: 'success',
	overdue: 'danger',
	cancelled: 'muted'
};

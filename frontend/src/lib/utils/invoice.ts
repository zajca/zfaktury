// Shared invoice status labels and colors for consistent UI across pages

export const statusLabels: Record<string, string> = {
	draft: 'Koncept',
	sent: 'Odeslan\u00e1',
	paid: 'Uhrazen\u00e1',
	overdue: 'Po splatnosti',
	cancelled: 'Stornovan\u00e1'
};

export const statusColors: Record<string, string> = {
	draft: 'bg-gray-100 text-gray-700',
	sent: 'bg-blue-100 text-blue-700',
	paid: 'bg-green-100 text-green-700',
	overdue: 'bg-red-100 text-red-700',
	cancelled: 'bg-gray-100 text-gray-500'
};

export const vatStatusLabels: Record<string, string> = {
	draft: 'Koncept',
	ready: 'Připraveno',
	filed: 'Podáno'
};

export const vatStatusColors: Record<string, string> = {
	draft: 'bg-gray-100 text-gray-700',
	ready: 'bg-blue-100 text-blue-700',
	filed: 'bg-green-100 text-green-700'
};

export const filingTypeLabels: Record<string, string> = {
	regular: 'Řádné',
	corrective: 'Následné',
	supplementary: 'Opravné'
};

export const monthLabels: Record<number, string> = {
	1: 'Leden', 2: 'Únor', 3: 'Březen', 4: 'Duben',
	5: 'Květen', 6: 'Červen', 7: 'Červenec', 8: 'Srpen',
	9: 'Září', 10: 'Říjen', 11: 'Listopad', 12: 'Prosinec'
};

export const quarterLabels: Record<number, string> = {
	1: 'Q1 (leden \u2013 březen)',
	2: 'Q2 (duben \u2013 červen)',
	3: 'Q3 (červenec \u2013 září)',
	4: 'Q4 (říjen \u2013 prosinec)'
};

export const quarters = [
	{ quarter: 1, months: [1, 2, 3], label: '1. čtvrtletí' },
	{ quarter: 2, months: [4, 5, 6], label: '2. čtvrtletí' },
	{ quarter: 3, months: [7, 8, 9], label: '3. čtvrtletí' },
	{ quarter: 4, months: [10, 11, 12], label: '4. čtvrtletí' }
];

/**
 * Czech text of a warning token returned with an insurance overview.
 * Unknown tokens are returned unchanged so a new backend token stays visible.
 */
export function insuranceWarningText(token: string, overviewYear: number): string {
	switch (token) {
		case 'advance_minimum_missing':
			return `Minimální záloha pro rok ${overviewYear + 1} není v katalogu, ověřte ji u ČSSZ / pojišťovny`;
		case 'filing_method_missing':
			return `Způsob podání daňového přiznání za rok ${overviewYear} není nastaven. Lhůta přehledu a první měsíc nové zálohy jsou spočítány pro podání přiznání do 1. dubna.`;
		case 'advance_differs_from_xml':
			return 'Přehled byl podán dříve, než aplikace počítala. Od prvního měsíce platí záloha podle tabulky níže, ne částka z XML přehledu.';
		default:
			return token;
	}
}

/** Month and year in Czech, e.g. "červen 2026". */
export function formatMonthYear(month: number, year: number): string {
	return new Intl.DateTimeFormat('cs-CZ', { month: 'long', year: 'numeric' }).format(
		new Date(year, month - 1, 1)
	);
}

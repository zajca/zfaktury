import { describe, it, expect, afterEach } from 'vitest';
import { render, screen, cleanup } from '@testing-library/svelte';
import AdvanceSchedule from './AdvanceSchedule.svelte';

afterEach(() => {
	cleanup();
});

const periods = [
	{ from_month: 6, monthly_base: 1958700, monthly_prepay: 572000 },
	{ from_month: 7, monthly_base: 1713900, monthly_prepay: 500500 }
];

function setup(overrides: Record<string, unknown> = {}) {
	return render(AdvanceSchedule, {
		props: {
			overviewYear: 2025,
			startMonth: 6,
			periods,
			warnings: [],
			reportedPrepay: 572000,
			...overrides
		}
	});
}

describe('AdvanceSchedule', () => {
	it('shows the start month of the new advance', () => {
		setup();
		expect(screen.getByText(/Nová záloha od:/)).toHaveTextContent('Nová záloha od: červen 2026');
	});

	it('lists every period when there is more than one', () => {
		setup();
		expect(screen.getByText(/5\s720,00/)).toBeInTheDocument();
		expect(screen.getByText(/5\s005,00/)).toBeInTheDocument();
		expect(screen.getByText('od červenec 2026:')).toBeInTheDocument();
	});

	it('does not list a single period', () => {
		setup({ periods: [periods[1]], startMonth: 7, reportedPrepay: 500500 });
		expect(screen.queryByText(/^od /)).toBeNull();
	});

	it('shows the reported advance when the first period differs', () => {
		setup({ reportedPrepay: 500500 });
		expect(screen.getByText(/Záloha uvedená v přehledu \(ř\. 36\)/)).toBeInTheDocument();
	});

	it('hides the reported advance when it equals the first period', () => {
		setup();
		expect(screen.queryByText(/Záloha uvedená v přehledu/)).toBeNull();
	});

	it('shows the owner text for a missing minimum in an alert', () => {
		setup({ overviewYear: 2026, warnings: ['advance_minimum_missing'] });
		expect(screen.getByRole('alert')).toHaveTextContent(
			'Minimální záloha pro rok 2027 není v katalogu, ověřte ji u ČSSZ / pojišťovny'
		);
	});

	it('links the missing filing method warning to the settings page', () => {
		setup({ warnings: ['filing_method_missing'] });
		expect(screen.getByRole('link', { name: 'Nastavit způsob podání' })).toHaveAttribute(
			'href',
			'/tax/prepayments'
		);
	});

	it('shows no start line when the start month is 0', () => {
		setup({ startMonth: 0, periods: [] });
		expect(screen.queryByText(/Nová záloha od:/)).toBeNull();
	});
});

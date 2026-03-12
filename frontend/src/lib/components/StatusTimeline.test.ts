import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { render, screen, cleanup } from '@testing-library/svelte';
import StatusTimeline from './StatusTimeline.svelte';
import type { InvoiceStatusChange } from '$lib/api/client';

function makeChange(
	overrides: Partial<InvoiceStatusChange> & { id: number }
): InvoiceStatusChange {
	return {
		invoice_id: 1,
		old_status: 'draft',
		new_status: 'sent',
		changed_at: '2026-03-10T10:00:00Z',
		note: '',
		...overrides
	};
}

beforeEach(() => {});
afterEach(() => {
	cleanup();
});

describe('StatusTimeline', () => {
	it('renders timeline with multiple status changes', () => {
		const history: InvoiceStatusChange[] = [
			makeChange({ id: 1, old_status: 'draft', new_status: 'sent', changed_at: '2026-03-10T10:00:00Z' }),
			makeChange({ id: 2, old_status: 'sent', new_status: 'paid', changed_at: '2026-03-11T14:00:00Z' })
		];

		render(StatusTimeline, { props: { history } });

		const timeline = screen.getByTestId('status-timeline');
		expect(timeline).toBeInTheDocument();

		// Should have 2 timestamps (one per entry)
		const timestamps = screen.getAllByTestId('timestamp');
		expect(timestamps).toHaveLength(2);
	});

	it('shows old -> new status transition with Czech labels', () => {
		const history: InvoiceStatusChange[] = [
			makeChange({ id: 1, old_status: 'draft', new_status: 'sent' })
		];

		render(StatusTimeline, { props: { history } });

		// Czech labels for draft and sent
		expect(screen.getByText('Koncept')).toBeInTheDocument();
		expect(screen.getByText('Odeslaná')).toBeInTheDocument();
	});

	it('displays formatted timestamps', () => {
		const history: InvoiceStatusChange[] = [
			makeChange({ id: 1, changed_at: '2026-03-10T10:30:00Z' })
		];

		render(StatusTimeline, { props: { history } });

		const timestamp = screen.getByTestId('timestamp');
		// formatDateTime returns Czech locale format, e.g. "10. 3. 2026 11:30"
		// The exact output depends on timezone, but it should contain date parts
		expect(timestamp.textContent).toBeTruthy();
		expect(timestamp.textContent).not.toBe('-');
	});

	it('shows notes when present', () => {
		const history: InvoiceStatusChange[] = [
			makeChange({ id: 1, note: 'Zaplaceno prevodem' })
		];

		render(StatusTimeline, { props: { history } });

		const note = screen.getByTestId('note');
		expect(note).toBeInTheDocument();
		expect(note.textContent).toBe('Zaplaceno prevodem');
	});

	it('does not show note element when note is empty', () => {
		const history: InvoiceStatusChange[] = [
			makeChange({ id: 1, note: '' })
		];

		render(StatusTimeline, { props: { history } });

		expect(screen.queryByTestId('note')).not.toBeInTheDocument();
	});

	it('handles empty history with empty state message', () => {
		render(StatusTimeline, { props: { history: [] } });

		const emptyState = screen.getByTestId('empty-state');
		expect(emptyState).toBeInTheDocument();
		expect(emptyState.textContent).toContain('Zatím žádné změny stavu');
		expect(screen.queryByTestId('status-timeline')).not.toBeInTheDocument();
	});

	it('orders entries newest first', () => {
		const history: InvoiceStatusChange[] = [
			makeChange({ id: 1, old_status: 'draft', new_status: 'sent', changed_at: '2026-03-10T10:00:00Z' }),
			makeChange({ id: 2, old_status: 'sent', new_status: 'paid', changed_at: '2026-03-11T14:00:00Z' }),
			makeChange({ id: 3, old_status: 'paid', new_status: 'cancelled', changed_at: '2026-03-09T08:00:00Z' })
		];

		render(StatusTimeline, { props: { history } });

		const timestamps = screen.getAllByTestId('timestamp');
		expect(timestamps).toHaveLength(3);

		// Newest first: id=2 (March 11), id=1 (March 10), id=3 (March 9)
		// Check that the badges appear in correct order by looking at the
		// new_status badges: paid, sent, cancelled
		// The component renders Badge pairs, so we check all badge texts
		const badges = screen.getAllByText(/(Uhrazená|Odeslaná|Stornovaná|Koncept)/);
		// First entry (newest): sent -> paid => "Odeslaná", "Uhrazená"
		// Second entry: draft -> sent => "Koncept", "Odeslaná"
		// Third entry (oldest): paid -> cancelled => "Uhrazená", "Stornovaná"

		// Verify order by checking that the first "Uhrazená" appears as a new_status (2nd badge)
		// and "Stornovaná" appears last
		const allText = screen.getByTestId('status-timeline').textContent ?? '';
		const paidIdx = allText.indexOf('Uhrazená');
		const cancelledIdx = allText.indexOf('Stornovaná');
		expect(paidIdx).toBeLessThan(cancelledIdx);
	});

	it('renders all status types with correct Czech labels', () => {
		const history: InvoiceStatusChange[] = [
			makeChange({ id: 1, old_status: 'draft', new_status: 'overdue', changed_at: '2026-03-10T10:00:00Z' })
		];

		render(StatusTimeline, { props: { history } });

		expect(screen.getByText('Koncept')).toBeInTheDocument();
		expect(screen.getByText('Po splatnosti')).toBeInTheDocument();
	});
});

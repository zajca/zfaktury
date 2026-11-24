import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, fireEvent, cleanup } from '@testing-library/svelte';
import InvoiceEditFormTestWrapper from './InvoiceEditFormTestWrapper.svelte';

beforeEach(() => {
	vi.useFakeTimers();
	vi.setSystemTime(new Date('2026-03-10T12:00:00Z'));
});

afterEach(() => {
	cleanup();
	vi.useRealTimers();
});

describe('InvoiceEditForm last day of previous month button', () => {
	it('renders a real button with the Czech label', () => {
		render(InvoiceEditFormTestWrapper);
		const button = screen.getByRole('button', { name: 'Poslední den minulého měsíce' });
		expect(button).toHaveAttribute('type', 'button');
	});

	it('sets issue, delivery and due dates using the default offset', async () => {
		render(InvoiceEditFormTestWrapper);
		await fireEvent.click(screen.getByRole('button', { name: 'Poslední den minulého měsíce' }));

		expect(document.getElementById('edit-issue')).toHaveValue('2026-02-28');
		expect(document.getElementById('edit-delivery')).toHaveValue('2026-02-28');
		expect(document.getElementById('edit-due')).toHaveValue('2026-03-14');
	});

	it('keeps the current due offset', async () => {
		render(InvoiceEditFormTestWrapper, { props: { dueDateOffset: 30 } });
		await fireEvent.click(screen.getByRole('button', { name: 'Poslední den minulého měsíce' }));

		expect(document.getElementById('edit-issue')).toHaveValue('2026-02-28');
		expect(document.getElementById('edit-delivery')).toHaveValue('2026-02-28');
		expect(document.getElementById('edit-due')).toHaveValue('2026-03-30');
	});
});

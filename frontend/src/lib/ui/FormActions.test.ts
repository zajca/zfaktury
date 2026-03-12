import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, fireEvent, cleanup } from '@testing-library/svelte';
import FormActions from './FormActions.svelte';

afterEach(() => {
	cleanup();
});

describe('FormActions', () => {
	it('renders save button with default label', () => {
		render(FormActions, { props: { saving: false, cancelHref: '/back' } });
		expect(screen.getByText('Uložit')).toBeInTheDocument();
	});

	it('renders saving label when saving', () => {
		render(FormActions, { props: { saving: true, cancelHref: '/back' } });
		expect(screen.getByText('Ukládám...')).toBeInTheDocument();
	});

	it('disables save button when saving', () => {
		render(FormActions, { props: { saving: true, cancelHref: '/back' } });
		const saveBtn = screen.getByText('Ukládám...');
		expect(saveBtn).toBeDisabled();
	});

	it('renders custom save labels', () => {
		render(FormActions, {
			props: {
				saving: false,
				saveLabel: 'Vytvořit',
				savingLabel: 'Vytvářím...',
				cancelHref: '/back'
			}
		});
		expect(screen.getByText('Vytvořit')).toBeInTheDocument();
	});

	it('renders cancel link when cancelHref is provided', () => {
		render(FormActions, { props: { saving: false, cancelHref: '/invoices' } });
		const cancelLink = screen.getByText('Zrušit');
		expect(cancelLink).toBeInTheDocument();
		expect(cancelLink.getAttribute('href')).toBe('/invoices');
	});

	it('renders cancel button when oncancel is provided', async () => {
		const oncancel = vi.fn();
		render(FormActions, { props: { saving: false, oncancel } });
		const cancelBtn = screen.getByText('Zrušit');
		await fireEvent.click(cancelBtn);
		expect(oncancel).toHaveBeenCalled();
	});

	it('save button has submit type', () => {
		render(FormActions, { props: { saving: false, cancelHref: '/back' } });
		const saveBtn = screen.getByText('Uložit');
		expect(saveBtn.getAttribute('type')).toBe('submit');
	});
});

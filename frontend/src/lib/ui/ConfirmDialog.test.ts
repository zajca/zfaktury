import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, fireEvent, cleanup } from '@testing-library/svelte';
import ConfirmDialog from './ConfirmDialog.svelte';

vi.mock('$app/environment', () => ({ browser: true }));

afterEach(() => {
	cleanup();
});

function renderDialog(props: Partial<Parameters<typeof ConfirmDialog>[1]> = {}) {
	return render(ConfirmDialog, {
		open: true,
		title: 'Smazat polozku',
		message: 'Opravdu chcete smazat tuto polozku?',
		onconfirm: vi.fn(),
		oncancel: vi.fn(),
		...props
	} as any);
}

describe('ConfirmDialog', () => {
	it('is not rendered when open is false', () => {
		render(ConfirmDialog, {
			open: false,
			title: 'Test',
			message: 'Test message',
			onconfirm: vi.fn(),
			oncancel: vi.fn()
		} as any);
		expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
	});

	it('renders with title and message when open', () => {
		renderDialog();
		expect(screen.getByRole('alertdialog')).toBeInTheDocument();
		expect(screen.getByText('Smazat polozku')).toBeInTheDocument();
		expect(screen.getByText('Opravdu chcete smazat tuto polozku?')).toBeInTheDocument();
	});

	it('has aria-modal="true"', () => {
		renderDialog();
		const dialog = screen.getByRole('alertdialog');
		expect(dialog.getAttribute('aria-modal')).toBe('true');
	});

	it('has backdrop with role="presentation"', () => {
		renderDialog();
		const backdrop = document.querySelector('[role="presentation"]');
		expect(backdrop).toBeInTheDocument();
	});

	it('calls onconfirm when confirm button clicked', async () => {
		const onconfirm = vi.fn();
		renderDialog({ onconfirm });
		const confirmBtn = screen.getByText('Potvrdit');
		await fireEvent.click(confirmBtn);
		expect(onconfirm).toHaveBeenCalledOnce();
	});

	it('calls oncancel when cancel button clicked', async () => {
		const oncancel = vi.fn();
		renderDialog({ oncancel });
		const cancelBtn = screen.getByText('Zrušit');
		await fireEvent.click(cancelBtn);
		expect(oncancel).toHaveBeenCalledOnce();
	});

	it('calls oncancel on Escape key', async () => {
		const oncancel = vi.fn();
		renderDialog({ oncancel });
		await fireEvent.keyDown(window, { key: 'Escape' });
		expect(oncancel).toHaveBeenCalledOnce();
	});

	it('calls oncancel when backdrop clicked', async () => {
		const oncancel = vi.fn();
		renderDialog({ oncancel });
		const backdrop = document.querySelector('[role="presentation"]') as HTMLElement;
		await fireEvent.click(backdrop);
		expect(oncancel).toHaveBeenCalledOnce();
	});

	it('shows loading state text', () => {
		renderDialog({ loading: true });
		expect(screen.getByText('Zpracovávám...')).toBeInTheDocument();
	});

	it('disables buttons when loading', () => {
		renderDialog({ loading: true });
		const buttons = screen.getAllByRole('button');
		buttons.forEach((btn) => {
			expect(btn).toBeDisabled();
		});
	});

	it('uses custom confirm and cancel labels', () => {
		renderDialog({ confirmLabel: 'Ano, smazat', cancelLabel: 'Ne' });
		expect(screen.getByText('Ano, smazat')).toBeInTheDocument();
		expect(screen.getByText('Ne')).toBeInTheDocument();
	});
});

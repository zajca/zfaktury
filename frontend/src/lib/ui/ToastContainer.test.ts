import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, cleanup } from '@testing-library/svelte';
import ToastContainer from './ToastContainer.svelte';
import {
	toasts,
	toastSuccess,
	toastError,
	toastWarning,
	toastInfo,
	removeToast,
	clearAllToasts
} from '$lib/data/toast-state.svelte';

vi.mock('$app/environment', () => ({ browser: true }));

// jsdom does not implement element.animate (used by Svelte transitions)
if (typeof Element.prototype.animate !== 'function') {
	Element.prototype.animate = vi.fn().mockReturnValue({
		finished: Promise.resolve(),
		cancel: vi.fn(),
		onfinish: null
	});
}

beforeEach(() => {
	vi.useFakeTimers();
	clearAllToasts();
});

afterEach(() => {
	cleanup();
	clearAllToasts();
	vi.useRealTimers();
});

describe('ToastContainer', () => {
	it('renders nothing when no toasts', () => {
		render(ToastContainer);
		expect(screen.queryByRole('status')).not.toBeInTheDocument();
		expect(screen.queryByRole('alert')).not.toBeInTheDocument();
	});

	it('renders success toast with role="status"', async () => {
		toastSuccess('Saved successfully');
		render(ToastContainer);
		await vi.advanceTimersByTimeAsync(10);
		expect(screen.getByRole('status')).toBeInTheDocument();
		expect(screen.getByText('Saved successfully')).toBeInTheDocument();
	});

	it('renders error toast with role="alert"', async () => {
		toastError('Something went wrong');
		render(ToastContainer);
		await vi.advanceTimersByTimeAsync(10);
		expect(screen.getByRole('alert')).toBeInTheDocument();
		expect(screen.getByText('Something went wrong')).toBeInTheDocument();
	});

	it('renders warning toast with role="alert"', async () => {
		toastWarning('Watch out');
		render(ToastContainer);
		await vi.advanceTimersByTimeAsync(10);
		expect(screen.getByRole('alert')).toBeInTheDocument();
		expect(screen.getByText('Watch out')).toBeInTheDocument();
	});

	it('renders info toast with role="status"', async () => {
		toastInfo('For your information');
		render(ToastContainer);
		await vi.advanceTimersByTimeAsync(10);
		expect(screen.getByRole('status')).toBeInTheDocument();
		expect(screen.getByText('For your information')).toBeInTheDocument();
	});

	it('dismiss button calls removeToast and updates state', async () => {
		const id = toastError('Dismissable error');
		render(ToastContainer);
		await vi.advanceTimersByTimeAsync(10);

		expect(screen.getByText('Dismissable error')).toBeInTheDocument();
		expect(toasts.length).toBe(1);

		const dismissBtn = screen.getByLabelText('Zavřít');
		await fireEvent.click(dismissBtn);

		// State should be updated immediately
		expect(toasts.length).toBe(0);
	});

	it('renders multiple toasts', async () => {
		toastSuccess('First');
		toastError('Second');
		render(ToastContainer);
		await vi.advanceTimersByTimeAsync(10);

		expect(screen.getByText('First')).toBeInTheDocument();
		expect(screen.getByText('Second')).toBeInTheDocument();
	});
});

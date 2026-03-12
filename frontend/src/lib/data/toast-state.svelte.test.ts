import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// We need to test the toast state module. Since it uses $state (runes),
// we import and test the exported functions directly.
import {
	toasts,
	toastSuccess,
	toastError,
	toastWarning,
	toastInfo,
	removeToast,
	clearAllToasts
} from './toast-state.svelte';

beforeEach(() => {
	vi.useFakeTimers();
	clearAllToasts();
});

afterEach(() => {
	clearAllToasts();
	vi.useRealTimers();
});

describe('toast-state', () => {
	describe('addToast', () => {
		it('adds a success toast', () => {
			toastSuccess('Operation completed');
			expect(toasts.length).toBe(1);
			expect(toasts[0].type).toBe('success');
			expect(toasts[0].message).toBe('Operation completed');
			expect(toasts[0].id).toBeTruthy();
		});

		it('adds an error toast', () => {
			toastError('Something failed');
			expect(toasts.length).toBe(1);
			expect(toasts[0].type).toBe('error');
			expect(toasts[0].message).toBe('Something failed');
		});

		it('adds a warning toast', () => {
			toastWarning('Be careful');
			expect(toasts.length).toBe(1);
			expect(toasts[0].type).toBe('warning');
		});

		it('adds an info toast', () => {
			toastInfo('FYI');
			expect(toasts.length).toBe(1);
			expect(toasts[0].type).toBe('info');
		});

		it('returns the toast id', () => {
			const id = toastSuccess('test');
			expect(id).toBeTruthy();
			expect(toasts[0].id).toBe(id);
		});
	});

	describe('removeToast', () => {
		it('removes a toast by id', () => {
			const id = toastSuccess('test');
			expect(toasts.length).toBe(1);
			removeToast(id);
			expect(toasts.length).toBe(0);
		});

		it('does nothing for unknown id', () => {
			toastSuccess('test');
			removeToast('nonexistent');
			expect(toasts.length).toBe(1);
		});
	});

	describe('FIFO eviction', () => {
		it('evicts oldest toast when exceeding max of 3', () => {
			const id1 = toastError('first');
			const id2 = toastError('second');
			const id3 = toastError('third');
			expect(toasts.length).toBe(3);

			// Adding a 4th should evict the first
			toastError('fourth');
			expect(toasts.length).toBe(3);
			expect(toasts.find((t) => t.id === id1)).toBeUndefined();
			expect(toasts[0].message).toBe('second');
			expect(toasts[2].message).toBe('fourth');
		});
	});

	describe('auto-dismiss', () => {
		it('auto-dismisses success toasts after 5 seconds', async () => {
			toastSuccess('will disappear');
			expect(toasts.length).toBe(1);

			await vi.advanceTimersByTimeAsync(4999);
			expect(toasts.length).toBe(1);

			await vi.advanceTimersByTimeAsync(1);
			expect(toasts.length).toBe(0);
		});

		it('auto-dismisses info toasts after 5 seconds', async () => {
			toastInfo('info message');
			expect(toasts.length).toBe(1);

			await vi.advanceTimersByTimeAsync(5000);
			expect(toasts.length).toBe(0);
		});

		it('auto-dismisses warning toasts after 8 seconds', async () => {
			toastWarning('warning message');
			expect(toasts.length).toBe(1);

			await vi.advanceTimersByTimeAsync(7999);
			expect(toasts.length).toBe(1);

			await vi.advanceTimersByTimeAsync(1);
			expect(toasts.length).toBe(0);
		});

		it('does not auto-dismiss error toasts', async () => {
			toastError('persistent error');
			expect(toasts.length).toBe(1);

			await vi.advanceTimersByTimeAsync(60000);
			expect(toasts.length).toBe(1);
		});

		it('clears timeout when manually removing toast', async () => {
			const id = toastSuccess('test');
			removeToast(id);
			expect(toasts.length).toBe(0);

			// Advancing time should not cause issues
			await vi.advanceTimersByTimeAsync(10000);
			expect(toasts.length).toBe(0);
		});
	});

	describe('clearAllToasts', () => {
		it('removes all toasts', () => {
			toastSuccess('a');
			toastError('b');
			toastWarning('c');
			expect(toasts.length).toBe(3);

			clearAllToasts();
			expect(toasts.length).toBe(0);
		});
	});
});

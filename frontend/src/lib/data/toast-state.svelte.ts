export type ToastType = 'success' | 'error' | 'warning' | 'info';

export interface Toast {
	id: string;
	type: ToastType;
	message: string;
}

const MAX_TOASTS = 3;

export const toasts: Toast[] = $state([]);

// Internal map to track timeout IDs for cleanup
const timeoutMap = new Map<string, ReturnType<typeof setTimeout>>();

export function removeToast(id: string) {
	const timeoutId = timeoutMap.get(id);
	if (timeoutId) {
		clearTimeout(timeoutId);
		timeoutMap.delete(id);
	}
	const idx = toasts.findIndex((t) => t.id === id);
	if (idx !== -1) toasts.splice(idx, 1);
}

function addToast(type: ToastType, message: string): string {
	const id = crypto.randomUUID();

	// FIFO eviction
	while (toasts.length >= MAX_TOASTS) {
		removeToast(toasts[0].id);
	}

	toasts.push({ id, type, message });

	// Auto-dismiss durations
	const durations: Record<ToastType, number | null> = {
		success: 5000,
		info: 5000,
		warning: 8000,
		error: null
	};
	const duration = durations[type];
	if (duration) {
		const timeoutId = setTimeout(() => removeToast(id), duration);
		timeoutMap.set(id, timeoutId);
	}

	return id;
}

export function toastSuccess(message: string): string {
	return addToast('success', message);
}
export function toastError(message: string): string {
	return addToast('error', message);
}
export function toastWarning(message: string): string {
	return addToast('warning', message);
}
export function toastInfo(message: string): string {
	return addToast('info', message);
}

/** Remove all toasts and clear all pending timeouts */
export function clearAllToasts() {
	for (const [id, timeoutId] of timeoutMap) {
		clearTimeout(timeoutId);
		timeoutMap.delete(id);
	}
	toasts.splice(0, toasts.length);
}

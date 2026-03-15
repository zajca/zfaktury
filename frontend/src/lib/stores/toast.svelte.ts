export type ToastType = 'success' | 'error' | 'warning' | 'info';

export interface Toast {
	id: string;
	type: ToastType;
	message: string;
	duration: number;
}

let toasts = $state<Toast[]>([]);

export function addToast(message: string, type: ToastType = 'success', duration = 4000) {
	const id = crypto.randomUUID();
	toasts.push({ id, type, message, duration });
	setTimeout(() => removeToast(id), duration);
}

export function removeToast(id: string) {
	toasts = toasts.filter((t) => t.id !== id);
}

export function getToasts() {
	return toasts;
}

export const toast = {
	success: (msg: string) => addToast(msg, 'success'),
	error: (msg: string) => addToast(msg, 'error', 6000),
	warning: (msg: string) => addToast(msg, 'warning'),
	info: (msg: string) => addToast(msg, 'info')
};

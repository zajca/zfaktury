<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		variant?: 'default' | 'success' | 'danger' | 'warning' | 'info' | 'muted';
		children: Snippet;
		class?: string;
	}

	let { variant = 'default', children, class: className = '' }: Props = $props();

	const variantClasses: Record<string, string> = {
		default: 'bg-elevated text-secondary',
		success: 'bg-success-bg text-success',
		danger: 'bg-danger-bg text-danger',
		warning: 'bg-warning-bg text-warning',
		info: 'bg-info-bg text-info',
		muted: 'bg-elevated text-muted'
	};
</script>

<span class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium {variantClasses[variant]} {className}">
	{@render children()}
</span>

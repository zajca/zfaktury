<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLButtonAttributes, HTMLAnchorAttributes } from 'svelte/elements';

	interface Props {
		variant?: 'primary' | 'secondary' | 'ghost' | 'danger' | 'success';
		size?: 'sm' | 'md';
		disabled?: boolean;
		href?: string;
		type?: 'button' | 'submit';
		children: Snippet;
		onclick?: (e: MouseEvent) => void;
		class?: string;
		title?: string;
		'aria-label'?: string;
	}

	let {
		variant = 'secondary',
		size = 'md',
		disabled = false,
		href,
		type = 'button',
		children,
		onclick,
		class: className = '',
		title,
		'aria-label': ariaLabel
	}: Props = $props();

	const variantClasses: Record<string, string> = {
		primary: 'bg-accent text-white hover:bg-accent-hover',
		secondary: 'border border-border-strong text-secondary hover:bg-hover hover:text-primary',
		ghost: 'text-secondary hover:bg-hover hover:text-primary',
		danger: 'border border-danger/30 text-danger hover:bg-danger-bg',
		success: 'bg-success/90 text-base hover:bg-success'
	};

	const sizeClasses: Record<string, string> = {
		sm: 'px-2.5 py-1.5 text-xs gap-1.5 rounded-md',
		md: 'px-4 py-2 text-sm gap-2 rounded-lg'
	};

	let classes = $derived(
		`inline-flex items-center font-medium transition-colors disabled:opacity-40 disabled:cursor-not-allowed ${variantClasses[variant]} ${sizeClasses[size]} ${className}`
	);
</script>

{#if href && !disabled}
	<a {href} class={classes} {title} aria-label={ariaLabel}>
		{@render children()}
	</a>
{:else}
	<button {type} {disabled} {onclick} class={classes} {title} aria-label={ariaLabel}>
		{@render children()}
	</button>
{/if}

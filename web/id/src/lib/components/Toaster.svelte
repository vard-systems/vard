<script lang="ts">
	import { fly } from 'svelte/transition';
	import { useTranslator } from '$lib/i18n';
	import { dismiss, pause, resume, toasts, type ToastTone } from '$lib/toast.svelte';
	import Icon, { type IconName } from './Icon.svelte';

	const t = useTranslator();

	const icons: Record<ToastTone, IconName> = { success: 'success', danger: 'alert' };

	const motion =
		typeof window !== 'undefined' && matchMedia('(prefers-reduced-motion: reduce)').matches
			? { y: 0, duration: 120 }
			: { y: 16, duration: 220 };
</script>

<section class="toaster" aria-live="polite" aria-label={t('toast.region')}>
	{#each toasts as toast (toast.id)}
		<div
			class="toast"
			role={toast.tone === 'danger' ? 'alert' : 'status'}
			in:fly={motion}
			out:fly={{ ...motion, y: 8 }}
			onmouseenter={() => pause(toast.id)}
			onmouseleave={() => resume(toast.id)}
			onfocusin={() => pause(toast.id)}
			onfocusout={() => resume(toast.id)}
		>
			<span class="glyph"><Icon name={icons[toast.tone]} /></span>
			<p class="message">
				<span class="title">{toast.title}</span>
				{#if toast.description}
					<span class="description">{toast.description}</span>
				{/if}
			</p>
			<button
				type="button"
				class="close"
				aria-label={t('action.dismiss')}
				onclick={() => dismiss(toast.id)}
			>
				<Icon name="close" size="1rem" />
			</button>
		</div>
	{/each}
</section>

<style>
	.toaster {
		position: fixed;
		bottom: var(--space-4);
		left: 50%;
		z-index: 100;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: var(--space-2);
		width: min(30rem, calc(100vw - 2 * var(--space-4)));
		transform: translateX(-50%);
		pointer-events: none;
	}

	.toast {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		max-width: 100%;
		min-height: 44px;
		padding: 6px 6px 6px 12px;
		border-radius: 999px;
		background: var(--toast-surface);
		color: var(--toast-text);
		box-shadow:
			0 12px 32px -8px rgb(0 0 0 / 45%),
			0 2px 6px rgb(0 0 0 / 20%);
		pointer-events: auto;
	}

	.glyph {
		display: inline-flex;
		flex: none;
		color: var(--toast-icon);
	}

	.message {
		flex: 1;
		min-width: 0;
		margin: 0;
		font-size: var(--text-base);
		line-height: 1.4;
		overflow-wrap: anywhere;
	}

	.title {
		font-weight: 600;
	}

	.description {
		margin-left: 6px;
		color: var(--toast-hint);
		font-size: var(--text-sm);
	}

	.close {
		display: inline-flex;
		flex: none;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		border: none;
		border-radius: 999px;
		background: color-mix(in srgb, var(--toast-text), transparent 90%);
		color: var(--toast-text);
		cursor: pointer;
	}

	.close:hover {
		background: color-mix(in srgb, var(--toast-text), transparent 80%);
	}

	.close:focus-visible {
		outline: 2px solid var(--toast-icon);
		outline-offset: 1px;
	}
</style>

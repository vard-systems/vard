<script lang="ts">
	import { PasswordInput } from '@ark-ui/svelte/password-input';
	import { RiEyeLine, RiEyeOffLine } from '#lib/icons.js';
	import FieldText from './FieldText.svelte';
	import Icon from './Icon.svelte';

	type Props = {
		label: string;
		value: string;
		disabled?: boolean;
		required?: boolean;
		name?: string;
		autocomplete?: 'current-password' | 'new-password';
	};

	let {
		label,
		value = $bindable(''),
		disabled,
		required,
		name,
		autocomplete = 'current-password'
	}: Props = $props();
</script>

<PasswordInput.Root class="field-root" {disabled} {required}>
	<div class="field-box" data-float={value !== '' || undefined}>
		<PasswordInput.Label><FieldText {label} {required} /></PasswordInput.Label>
		<PasswordInput.Control>
			<PasswordInput.Input {name} bind:value {autocomplete} />
			<PasswordInput.VisibilityTrigger aria-label="Show password">
				<PasswordInput.Indicator>
					{#snippet fallback()}
						<Icon icon={RiEyeLine} size="1.0625rem" />
					{/snippet}
					<Icon icon={RiEyeOffLine} size="1.0625rem" />
				</PasswordInput.Indicator>
			</PasswordInput.VisibilityTrigger>
		</PasswordInput.Control>
	</div>
</PasswordInput.Root>

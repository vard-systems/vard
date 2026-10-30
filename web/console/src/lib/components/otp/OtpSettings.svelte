<script lang="ts">
	import { enhance } from '$app/forms';
	import { useQueryClient } from '@tanstack/svelte-query';
	import { resolve } from '$app/paths';
	import { otpApi, type OTPResponse, type OTPSettingsInput } from '#lib/api/index.js';
	import {
		Alert,
		FieldGrid,
		Input,
		List,
		ListItem,
		Panel,
		SaveBar,
		Tag,
		Thumb,
		notify
	} from '#lib/components/ui/index.js';
	import { keys } from '#lib/query/index.js';
	import { submit } from '#lib/forms.js';

	type Props = {
		settings: OTPResponse;
	};

	let { settings }: Props = $props();

	const queryClient = useQueryClient();

	const stored = $derived(settings.otp);
	const limits = $derived(settings.limits);

	function formOf(otp: OTPResponse['otp']) {
		return {
			code_length: String(otp.code_length),
			lifetime_minutes: String(otp.lifetime_minutes),
			max_attempts: String(otp.max_attempts),
			resend_seconds: String(otp.resend_seconds)
		};
	}

	// Seeded once, when the panel is first drawn: a refetch in the background
	// leaves what is being typed alone, and a save fills it in again.
	// svelte-ignore state_referenced_locally
	let form = $state(formOf(settings.otp));

	let saving = $state(false);

	function discard() {
		form = formOf(stored);
	}

	const input = $derived<Required<OTPSettingsInput>>({
		code_length: Number(form.code_length),
		lifetime_minutes: Number(form.lifetime_minutes),
		max_attempts: Number(form.max_attempts),
		resend_seconds: Number(form.resend_seconds)
	});

	const dirty = $derived(
		(Object.keys(input) as (keyof typeof input)[]).some((key) => input[key] !== stored[key])
	);

	function within(value: number, min: number, max: number): boolean {
		return Number.isInteger(value) && value >= min && value <= max;
	}

	const complete = $derived(
		within(input.code_length, limits.min_length, limits.max_length) &&
			within(input.lifetime_minutes, 1, limits.max_lifetime_minutes) &&
			within(input.max_attempts, 1, limits.max_attempts) &&
			within(input.resend_seconds, 0, limits.max_resend_seconds)
	);

	const odds = $derived.by(() => {
		if (!complete) return '';

		const codes = 10 ** input.code_length;
		const chance = Math.round(codes / input.max_attempts);

		return `A code is 1 of ${codes.toLocaleString()}, and one sign-in gets ${input.max_attempts} ${
			input.max_attempts === 1 ? 'try' : 'tries'
		} — about 1 in ${chance.toLocaleString()} of being guessed.`;
	});

	const save = submit<Awaited<ReturnType<typeof otpApi.update>>>({
		input: () => [input],
		when: () => !(!dirty || !complete),
		pending: (busy) => (saving = busy),
		done: async (result) => {
			queryClient.setQueryData(keys.otp.settings, result);
			form = formOf(result.otp);
			notify.success('Settings saved', 'The next code sent follows them.');
			await queryClient.invalidateQueries({ queryKey: keys.admin.overview });
		},
		failed: (err) => notify.error(err, 'Could not save these settings')
	});
</script>

<form method="POST" action="/actions/otp?/update" use:enhance={save}>
	{#if settings.flows.length === 0}
		<Alert tone="warning">
			No login flow asks for an emailed code yet, so nothing reads these. Add the
			<strong>Emailed code</strong> step to a flow on the
			<a href={resolve('/(panel)/flows')}>Login flows</a> page.
		</Alert>
	{/if}

	<Panel title="The code" icon="ri-hashtag">
		<FieldGrid spacing="comfortable">
			<Input
				label="Digits"
				bind:value={form.code_length}
				type="number"
				min={limits.min_length}
				max={limits.max_length}
				hint="Between {limits.min_length} and {limits.max_length}. Six is what most people expect."
			/>

			<Input
				label="Guesses"
				bind:value={form.max_attempts}
				type="number"
				min={1}
				max={limits.max_attempts}
				hint="Wrong codes before the sign-in is over and has to be started again."
			/>
		</FieldGrid>

		{#if odds}
			<p class="note">{odds}</p>
		{/if}
	</Panel>

	<Panel title="Timing" icon="ri-timer-line">
		<FieldGrid spacing="comfortable">
			<Input
				label="Valid for"
				bind:value={form.lifetime_minutes}
				type="number"
				min={1}
				max={limits.max_lifetime_minutes}
				suffix="minutes"
				hint="A code that outlives the message it came in is a password sitting in an inbox."
			/>

			<Input
				label="Wait before another"
				bind:value={form.resend_seconds}
				type="number"
				min={0}
				max={limits.max_resend_seconds}
				suffix="seconds"
				hint="What stops “send it again” being a way to post mail to a stranger. Zero turns the wait off."
			/>
		</FieldGrid>
	</Panel>

	{#if settings.flows.length > 0}
		<Panel title="Flows that ask for a code" icon="ri-git-branch-line" flush>
			{#snippet meta()}
				<Tag small>{settings.flows.length}</Tag>
			{/snippet}

			<List>
				{#each settings.flows as flow (flow.id)}
					<ListItem title={flow.name} description={flow.slug}>
						{#snippet lead()}
							<Thumb icon="ri-shield-keyhole-line" />
						{/snippet}

						{#snippet end()}
							{#if flow.is_default}
								<Tag tone="info" small>Default</Tag>
							{/if}
							{#if flow.is_enabled}
								<Tag tone="success" dot small>On</Tag>
							{:else}
								<Tag small>Off</Tag>
							{/if}
						{/snippet}
					</ListItem>
				{/each}
			</List>
		</Panel>
	{/if}

	{#if dirty}
		<SaveBar {saving} ready={complete} ondiscard={discard} />
	{/if}
</form>

<style>
	form {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}

	.note {
		margin: var(--space-3) 0 0;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		line-height: 1.5;
	}
</style>

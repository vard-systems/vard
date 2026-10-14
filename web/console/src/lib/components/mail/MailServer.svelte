<script lang="ts">
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import {
		RiAtLine,
		RiLockLine,
		RiMailCheckLine,
		RiMailLine,
		RiMailSendLine,
		RiServerLine,
		RiUserLine
	} from 'svelte-remixicon';
	import { mailApi, type MailEncryption, type MailResponse } from '#lib/api/index.js';
	import {
		Button,
		FieldGrid,
		FormSection,
		Input,
		SaveBar,
		Select,
		SwitchField,
		Tag,
		notify
	} from '#lib/components/ui/index.js';
	import { keys } from '#lib/query/index.js';

	type Props = {
		settings: MailResponse;
	};

	let { settings }: Props = $props();

	const queryClient = useQueryClient();

	const stored = $derived(settings.mail);

	function formOf({ has_password, port, ...rest }: MailResponse['mail']) {
		void has_password;
		return { ...rest, port: String(port) };
	}

	// Seeded once, when the panel is first drawn: a refetch in the background
	// leaves what is being typed alone, and a save fills it in again.
	// svelte-ignore state_referenced_locally
	let form = $state(formOf(settings.mail));

	/** The password is never read back, so the field starts empty whatever is
	    stored: typing one replaces it, and leaving it alone keeps it. */
	let password = $state('');
	let removePassword = $state(false);

	let saving = $state(false);

	function discard() {
		password = '';
		removePassword = false;
		form = formOf(stored);
	}

	const input = $derived({
		is_enabled: form.is_enabled,
		host: form.host.trim().toLowerCase(),
		port: Number(form.port),
		encryption: form.encryption,
		username: form.username.trim(),
		from_address: form.from_address.trim().toLowerCase(),
		from_name: form.from_name.trim()
	});

	const passwordInput = $derived(
		password !== '' ? password : removePassword && stored.has_password ? '' : undefined
	);

	const dirty = $derived(
		passwordInput !== undefined ||
			(Object.keys(input) as (keyof typeof input)[]).some((key) => input[key] !== stored[key])
	);

	const complete = $derived(
		Number.isInteger(input.port) &&
			input.port >= 1 &&
			input.port <= 65535 &&
			(!input.is_enabled || (input.host !== '' && input.from_address !== ''))
	);

	const encryptions = $derived(
		settings.encryptions.map((value) => ({
			value,
			label: labelOf(value),
			description: describe(value)
		}))
	);

	function labelOf(encryption: MailEncryption): string {
		return { starttls: 'STARTTLS', tls: 'TLS', none: 'None' }[encryption];
	}

	function describe(encryption: MailEncryption): string {
		return {
			starttls: 'Connect in the clear and upgrade. What port 587 expects.',
			tls: 'Encrypted from the first byte. What port 465 expects.',
			none: 'Nothing encrypted. Only for a relay on this machine.'
		}[encryption];
	}

	const save = createMutation(() => ({
		mutationFn: () => mailApi.update({ ...input, password: passwordInput }),
		onSuccess: async (result) => {
			queryClient.setQueryData(keys.mail.settings, result);
			form = formOf(result.mail);
			password = '';
			removePassword = false;
			notify.success(
				'Settings saved',
				'The next password reset, confirmation and one-time code goes through this server.'
			);
			await queryClient.invalidateQueries({ queryKey: keys.admin.overview });
		},
		onError: (err: unknown) => {
			notify.error(err, 'Could not save these settings');
		},
		onSettled: () => {
			saving = false;
		}
	}));

	function submit(event: SubmitEvent) {
		event.preventDefault();

		if (saving || !dirty || !complete) return;

		saving = true;
		save.mutate();
	}

	let testTo = $state('');

	const test = createMutation(() => ({
		mutationFn: () => mailApi.test({ ...input, password: passwordInput, to: testTo.trim() }),
		onSuccess: async (result) => {
			notify.success(
				`Test message sent to ${result.to}`,
				'If it does not arrive, look in the spam folder before changing anything.'
			);
			await queryClient.invalidateQueries({ queryKey: keys.admin.overview });
		},
		onError: (err: unknown) => {
			notify.error(err, 'Could not send a test message');
		}
	}));

	function sendTest() {
		if (test.isPending || testTo.trim() === '') return;

		test.mutate();
	}
</script>

<form onsubmit={submit}>
	<FormSection
		title="Sending"
		description="Whether messages leave this server at all."
		icon={RiMailSendLine}
	>
		{#snippet meta()}
			{#if form.is_enabled}
				<Tag tone="success" dot small>Sending</Tag>
			{:else}
				<Tag tone="warning" dot small>Logged, not sent</Tag>
			{/if}
		{/snippet}

		<SwitchField
			label="Send email through an SMTP server"
			description="Off, every message is written to the server's log instead — enough to follow a reset link while developing, and nothing anybody receives."
			bind:checked={form.is_enabled}
		/>
	</FormSection>

	<FormSection
		title="Mail server"
		description="Where messages are handed over, and how the connection to it is protected."
		icon={RiServerLine}
	>
		<Input
			label="Host"
			icon={RiServerLine}
			bind:value={form.host}
			maxlength={255}
			placeholder="smtp.example.com"
			hint="The host name on its own, with no scheme and no port."
		/>

		<FieldGrid spacing="comfortable">
			<Input
				label="Port"
				type="number"
				bind:value={form.port}
				min={1}
				max={65535}
				hint="587 for STARTTLS, 465 for TLS, 25 for a local relay."
			/>

			<Select
				label="Encryption"
				bind:value={form.encryption}
				options={encryptions}
				hint="How the connection is protected."
			/>

			<Input
				label="Username"
				icon={RiUserLine}
				bind:value={form.username}
				maxlength={255}
				hint="Leave empty for a server that takes no credentials."
			/>

			<Input
				label="Password"
				icon={RiLockLine}
				bind:value={password}
				type="password"
				autocomplete="off"
				maxlength={255}
				disabled={removePassword}
				placeholder={stored.has_password ? 'Stored — type to replace it' : ''}
				hint="Sealed with the server's secret key, and never read back."
			/>
		</FieldGrid>

		{#if stored.has_password}
			<SwitchField
				label="Forget the stored password"
				description="For a server that takes no credentials. The password is removed when you save."
				bind:checked={removePassword}
				onChange={() => (password = '')}
			/>
		{/if}
	</FormSection>

	<FormSection title="From" description="Who every message says it comes from." icon={RiMailLine}>
		<FieldGrid spacing="comfortable">
			<Input
				label="From address"
				icon={RiAtLine}
				bind:value={form.from_address}
				type="email"
				maxlength={255}
				placeholder="no-reply@example.com"
				hint="Most providers refuse an address they do not host."
			/>

			<Input
				label="From name"
				bind:value={form.from_name}
				maxlength={100}
				placeholder="Acme"
				hint="What a mail client shows instead of the address."
			/>
		</FieldGrid>
	</FormSection>

	<FormSection
		title="Send a test"
		description="One message with the settings on this form, saved or not — so a server can be tried before anybody's sign-in depends on it. A password left blank is the stored one."
		icon={RiMailCheckLine}
	>
		<div class="try">
			<Input
				label="Send a test message to"
				icon={RiAtLine}
				bind:value={testTo}
				type="email"
				maxlength={255}
				placeholder="you@example.com"
			/>

			<Button
				size="lg"
				variant="outline"
				onclick={sendTest}
				loading={test.isPending}
				disabled={test.isPending || testTo.trim() === '' || !complete}
			>
				{test.isPending ? 'Sending…' : 'Send'}
			</Button>
		</div>
	</FormSection>

	{#if dirty}
		<SaveBar {saving} ready={complete} ondiscard={discard} />
	{/if}
</form>

<style>
	form > :global(.save-bar) {
		margin-top: var(--space-5);
	}

	.try {
		display: flex;
		align-items: center;
		gap: var(--space-2);
	}

	.try > :global(:first-child) {
		flex: 1;
		min-width: 0;
	}
</style>

<script lang="ts">
	import { FormSection, Input, Note, SwitchField, Textarea } from '#lib/components/ui/index.js';
	import type { ApplicationForm } from '../form.svelte';

	type Props = { form: ApplicationForm };

	let { form }: Props = $props();
</script>

<FormSection
	title="Redirects"
	description="Where users may be sent back to. Each URI is matched exactly, with no wildcards."
>
	<Textarea
		label="Redirect URIs"
		bind:value={form.redirects}
		rows={3}
		placeholder="https://shop.example.com/auth/callback"
		hint={form.input.type === 'native'
			? 'One per line. https, a private-use scheme such as com.example.app:/callback, or a loopback address such as http://127.0.0.1/callback, which matches on any port.'
			: 'One per line. https, or http on localhost.'}
	/>

	{#if form.input.type === 'native'}
		<SwitchField
			label="Allow development redirects"
			description="Lets this app register Expo Go's exp:// addresses while it is developed."
			bind:checked={form.input.allow_development_redirects}
		/>
		{#if form.input.allow_development_redirects}
			<Note boxed>
				exp:// addresses name a developer's machine on their network. Remove them and turn this off
				before the app is released.
			</Note>
		{/if}
	{/if}

	<Textarea
		label="Post-logout redirect URIs"
		bind:value={form.logoutRedirects}
		rows={2}
		placeholder="https://shop.example.com/"
		hint="One per line. Where the app may send users after they sign out."
	/>

	{#if form.input.type === 'web' || form.input.type === 'spa'}
		<Input
			label="Front-channel logout URI"
			type="url"
			bind:value={form.input.frontchannel_logout_uri}
			maxlength={512}
			placeholder="https://shop.example.com/auth/frontchannel-logout"
			hint="Optional. Loaded in a hidden frame of the browser that signs out, with iss and sid, so the app can end its session there. The page has to allow being framed by the sign-in pages."
		/>
	{/if}

	{#if form.input.type === 'web'}
		<Input
			label="Back-channel logout URI"
			type="url"
			bind:value={form.input.backchannel_logout_uri}
			maxlength={512}
			placeholder="https://shop.example.com/auth/backchannel-logout"
			hint="Optional. The server posts a logout token here when someone signed in to this app signs out, so the app can end its own session too."
		/>
	{/if}
</FormSection>

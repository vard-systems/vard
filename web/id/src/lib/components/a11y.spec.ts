// @vitest-environment jsdom
import { createRawSnippet } from 'svelte';
import { describe, it } from 'vitest';
import { expectAccessible } from '#lib/testing/a11y.js';
import Decision from './auth/Decision.svelte';
import PresenceCheck from './account/PresenceCheck.svelte';
import RecoveryCodeSheet from './account/RecoveryCodeSheet.svelte';
import SessionList from './account/SessionList.svelte';
import Alert from './ui/Alert.svelte';
import Button from './ui/Button.svelte';
import PasswordField from './ui/PasswordField.svelte';
import TextField from './ui/TextField.svelte';

const text = (words: string) => createRawSnippet(() => ({ render: () => `<span>${words}</span>` }));
const nothing = () => {};

describe('the sign-in pages are usable without seeing them', () => {
	it('a text field', () => expectAccessible(TextField, { label: 'Email', value: '' }));

	it('a text field with a hint and an error', () =>
		expectAccessible(TextField, {
			label: 'Code',
			value: '12',
			hint: 'Six digits',
			error: 'That code is not right'
		}));

	it('a password field', () =>
		expectAccessible(PasswordField, { label: 'Password', value: 'secret' }));

	it('a button', () => expectAccessible(Button, { children: text('Continue') }));

	it('an alert', () => expectAccessible(Alert, { tone: 'warning', children: text('Careful') }));

	it('a yes or no answer', () =>
		expectAccessible(Decision, { yes: 'Allow', no: 'Deny', onanswer: nothing }));

	it('recovery codes shown once', () =>
		expectAccessible(RecoveryCodeSheet, {
			codes: ['abcde-fghjk', 'mnpqr-stuvw'],
			ondone: nothing
		}));

	it('a password asked before a change', () =>
		expectAccessible(PresenceCheck, {
			asksPassword: true,
			confirm: 'Remove',
			onconfirm: nothing,
			oncancel: nothing
		}));

	it('the list of devices signed in', () =>
		expectAccessible(SessionList, {
			timezone: 'UTC',
			sessions: [
				{
					id: '1',
					ip: '203.0.113.7',
					user_agent: 'Mozilla/5.0 (Macintosh) Chrome/140.0',
					signed_in_at: '2026-10-01T10:00:00Z',
					expires_at: '2026-11-01T10:00:00Z',
					current: true
				},
				{
					id: '2',
					ip: '',
					user_agent: '',
					signed_in_at: '2026-10-02T10:00:00Z',
					expires_at: '2026-11-02T10:00:00Z',
					current: false
				}
			]
		}));
});

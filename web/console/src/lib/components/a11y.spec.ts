// @vitest-environment jsdom
import { createRawSnippet } from 'svelte';
import { describe, it } from 'vitest';
import type { Admin } from '#lib/api/index.js';
import { expectAccessible } from '#lib/testing/a11y.js';
import IntrospectorList from './apis/IntrospectorList.svelte';
import Inspector from './flows/editor/Inspector.svelte';
import { TEMPLATES } from './flows/steps';
import LogoField from './organization/LogoField.svelte';
import RoleCodes from './profile/RoleCodes.svelte';
import DomainProofList from './sso/DomainProofList.svelte';
import Button from './ui/Button.svelte';
import EmptyState from './ui/EmptyState.svelte';
import Input from './ui/Input.svelte';
import PasswordInput from './ui/PasswordInput.svelte';
import SwitchField from './ui/SwitchField.svelte';
import TabList from './ui/TabList.svelte';

const text = (words: string) => createRawSnippet(() => ({ render: () => `<span>${words}</span>` }));

const admin = {
	id: '1',
	username: 'ada',
	email: 'ada@example.com',
	full_name: 'Ada Lovelace',
	first_name: 'Ada',
	last_name: 'Lovelace',
	avatar_url: '',
	status: 'active',
	roles: ['auditor', 'super_admin'],
	permissions: [],
	scoped_permissions: {},
	is_super_admin: true,
	mfa_enabled: true
} satisfies Admin;

describe('the panel is usable without seeing it', () => {
	it("an API's introspectors: one listed, one not, and a public client left out", () =>
		expectAccessible(IntrospectorList, {
			introspectors: [
				{
					id: '1',
					name: 'Orders backend',
					type: 'm2m',
					client_id: 'a1',
					confidential: true,
					listed: true
				},
				{ id: '2', name: 'Shop', type: 'web', client_id: 'b2', confidential: true, listed: false },
				{
					id: '3',
					name: 'Phone app',
					type: 'native',
					client_id: 'c3',
					confidential: false,
					listed: false
				}
			],
			editable: true,
			saving: null,
			onChange: () => {}
		}));

	it('the logo picker with a logo uploaded', () =>
		expectAccessible(LogoField, {
			version: 'a1b2c3d4e5f60718',
			editable: true,
			onsaved: () => {}
		}));

	it('the logo picker showing the default, read only', () =>
		expectAccessible(LogoField, { version: '', editable: false, onsaved: () => {} }));

	it("a flow's two-step step, with a second factor required", () =>
		expectAccessible(Inspector, {
			draft: {
				...TEMPLATES[0].draft,
				name: 'Staff',
				slug: 'staff',
				steps: ['identifier', 'password', 'totp'],
				require_second_factor: true
			},
			kinds: [
				{
					step: 'totp',
					label: 'Two-step verification',
					description: 'An authenticator app or passkeys.',
					fixed: false,
					implemented: true
				}
			],
			selected: 'totp',
			editable: true,
			creating: false,
			savedDefault: false,
			applications: 1,
			onRemove: () => {}
		}));

	it('a text input', () => expectAccessible(Input, { label: 'Name', value: '' }));

	it("a connection's domain proofs: proved and waiting for its record", () =>
		expectAccessible(DomainProofList, {
			proofs: [
				{
					domain: 'acme.com',
					proved: true,
					method: 'dns',
					record_name: '_x-challenge.acme.com',
					record_value: 'x-domain-verification=abc',
					verified_at: '2026-10-01T00:00:00Z',
					checked_at: '2026-10-01T00:00:00Z'
				},
				{
					domain: 'acme.org',
					proved: false,
					method: '',
					record_name: '_x-challenge.acme.org',
					record_value: 'x-domain-verification=def',
					verified_at: null,
					checked_at: null
				}
			],
			canWrite: true,
			oncheck: () => {}
		}));

	it('a text input with a hint and an error', () =>
		expectAccessible(Input, {
			label: 'Slug',
			value: 'a b',
			hint: 'Lower case',
			error: 'Not a slug'
		}));

	it('a password input', () =>
		expectAccessible(PasswordInput, { label: 'Password', value: '', hint: 'At least 10' }));

	it('a switch', () =>
		expectAccessible(SwitchField, {
			label: 'Default role',
			description: 'Given to every new user.',
			checked: true
		}));

	it('a button', () => expectAccessible(Button, { children: text('Save') }));

	it('tabs', () =>
		expectAccessible(TabList, {
			label: 'Kinds of role',
			value: 'global',
			tabs: [
				{ value: 'global', label: 'Global roles', icon: 'ri-global-line', count: 3 },
				{ value: 'application', label: 'Application roles', icon: 'ri-apps-line', count: 0 }
			]
		}));

	it('an empty state', () =>
		expectAccessible(EmptyState, { icon: 'ri-links-line', title: 'No connections yet' }));

	it("an administrator's roles", () => expectAccessible(RoleCodes, { admin }));
});

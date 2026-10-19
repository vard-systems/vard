import {
	RiAtLine,
	RiCheckboxCircleLine,
	RiFileAddLine,
	RiFileTextLine,
	RiKey2Line,
	RiMailLine,
	RiShareLine,
	RiShieldCheckLine,
	RiTimerFlashLine,
	type IconData
} from '#lib/icons.js';
import type { LoginStep, LoginStepSpec } from '#lib/api/index.js';
import { BRAND } from '#lib/brand.js';

const marks: Record<LoginStep, IconData> = {
	identifier: RiAtLine,
	password: RiKey2Line,
	social: RiShareLine,
	email_code: RiMailLine,
	totp: RiTimerFlashLine,
	terms: RiFileTextLine,
	consent: RiCheckboxCircleLine
};

export function markFor(step: LoginStep): IconData {
	return marks[step] ?? RiCheckboxCircleLine;
}

export function labelFor(step: LoginStep, kinds: LoginStepSpec[]): string {
	return kinds.find((kind) => kind.step === step)?.label ?? step;
}

export function describe(step: LoginStep, kinds: LoginStepSpec[]): string {
	return kinds.find((kind) => kind.step === step)?.description ?? '';
}

export function specFor(step: LoginStep, kinds: LoginStepSpec[]): LoginStepSpec | undefined {
	return kinds.find((kind) => kind.step === step);
}

export type FlowDraft = {
	name: string;
	slug: string;
	description: string;
	is_default: boolean;
	is_enabled: boolean;
	steps: LoginStep[];
	allow_sign_in: boolean;
	allow_registration: boolean;
	allow_password_reset: boolean;
	allow_remember_me: boolean;
	verify_email_on_register: boolean;
	require_verified_email: boolean;
	allow_email_change: boolean;
	session_lifetime_hours: number;
};

export function draftOf(flow: FlowDraft): FlowDraft {
	return {
		name: flow.name,
		slug: flow.slug,
		description: flow.description,
		is_default: flow.is_default,
		is_enabled: flow.is_enabled,
		steps: [...flow.steps],
		allow_sign_in: flow.allow_sign_in,
		allow_registration: flow.allow_registration,
		allow_password_reset: flow.allow_password_reset,
		allow_remember_me: flow.allow_remember_me,
		verify_email_on_register: flow.verify_email_on_register,
		require_verified_email: flow.require_verified_email,
		allow_email_change: flow.allow_email_change,
		session_lifetime_hours: flow.session_lifetime_hours
	};
}

const PROOF_STEPS: LoginStep[] = ['password', 'social'];

export const MAX_SESSION_HOURS = 24 * 90;

const SLUG = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;

export function problemsOf(draft: FlowDraft): string[] {
	const problems: string[] = [];

	if (draft.name.trim() === '') problems.push('Name the flow.');
	if (!SLUG.test(draft.slug))
		problems.push('The identifier has to be lower case letters, numbers and dashes.');
	if (!draft.steps.some((step) => PROOF_STEPS.includes(step)))
		problems.push('Add Password or Other accounts: without one, nobody can sign in.');
	if (draft.is_default && !draft.is_enabled)
		problems.push('The default flow cannot be turned off.');
	if (
		!Number.isInteger(draft.session_lifetime_hours) ||
		draft.session_lifetime_hours < 1 ||
		draft.session_lifetime_hours > MAX_SESSION_HOURS
	)
		problems.push('A session has to last between one hour and ninety days.');

	return problems;
}

export function slugFrom(name: string): string {
	return name
		.toLowerCase()
		.normalize('NFKD')
		.replace(/[^a-z0-9]+/g, '-')
		.replace(/^-+|-+$/g, '')
		.slice(0, 64);
}

export function freeSlug(wanted: string, taken: string[]): string {
	const base = wanted || 'flow';
	if (!taken.includes(base)) return base;

	for (let n = 2; ; n++) {
		const candidate = `${base.slice(0, 60)}-${n}`;
		if (!taken.includes(candidate)) return candidate;
	}
}

export type FlowTemplate = {
	id: string;
	name: string;
	hint: string;
	icon: IconData;
	draft: FlowDraft;
};

const base: Omit<FlowDraft, 'name' | 'slug' | 'steps'> = {
	description: '',
	is_default: false,
	is_enabled: true,
	allow_sign_in: true,
	allow_registration: true,
	allow_password_reset: true,
	allow_remember_me: true,
	verify_email_on_register: false,
	require_verified_email: false,
	allow_email_change: false,
	session_lifetime_hours: 24 * 14
};

export const TEMPLATES: FlowTemplate[] = [
	{
		id: 'password',
		name: 'Password',
		hint: 'An email and a password, with the accounts from the Social page beside them.',
		icon: RiKey2Line,
		draft: { ...base, name: '', slug: '', steps: ['identifier', 'password', 'social'] }
	},
	{
		id: 'passwordless',
		name: 'Other accounts only',
		hint: 'No passwords here: people sign in with Google, GitHub and the other providers.',
		icon: RiShareLine,
		draft: {
			...base,
			name: '',
			slug: '',
			steps: ['identifier', 'social'],
			allow_password_reset: false
		}
	},
	{
		id: 'staff',
		name: 'Staff',
		hint: 'A password, a verified address, no sign-ups, and eight-hour sessions.',
		icon: RiShieldCheckLine,
		draft: {
			...base,
			name: '',
			slug: '',
			steps: ['identifier', 'password'],
			allow_registration: false,
			require_verified_email: true,
			session_lifetime_hours: 8
		}
	},
	{
		id: 'blank',
		name: 'Blank',
		hint: 'An email and a password, and nothing else to start from.',
		icon: RiFileAddLine,
		draft: { ...base, name: '', slug: '', steps: ['identifier', 'password'] }
	}
];

const FLOW_SCHEMA = `${BRAND.slug}.login-flow/1`;

export type FlowFile = Omit<FlowDraft, 'is_default'> & { $schema: typeof FLOW_SCHEMA };

export function toFile(draft: FlowDraft): FlowFile {
	const copy = draftOf(draft);
	return {
		$schema: FLOW_SCHEMA,
		name: copy.name,
		slug: copy.slug,
		description: copy.description,
		is_enabled: copy.is_enabled,
		steps: copy.steps,
		allow_sign_in: copy.allow_sign_in,
		allow_registration: copy.allow_registration,
		allow_password_reset: copy.allow_password_reset,
		allow_remember_me: copy.allow_remember_me,
		verify_email_on_register: copy.verify_email_on_register,
		require_verified_email: copy.require_verified_email,
		allow_email_change: copy.allow_email_change,
		session_lifetime_hours: copy.session_lifetime_hours
	};
}

/** Reads an exported flow, or says why it is not one. Only the shape is
    checked here: the server holds it to the rules when it is saved. */
export function fromFile(text: string, kinds: LoginStepSpec[]): FlowDraft {
	let raw: unknown;
	try {
		raw = JSON.parse(text);
	} catch {
		throw new Error('That file is not JSON.');
	}

	const file = raw as Partial<FlowFile>;
	const known = new Set(kinds.map((kind) => kind.step));
	if (
		typeof file !== 'object' ||
		file === null ||
		typeof file.name !== 'string' ||
		!Array.isArray(file.steps) ||
		!file.steps.every((step) => known.has(step))
	) {
		throw new Error('That file is not a login flow exported from here.');
	}

	return {
		...base,
		name: file.name,
		slug: typeof file.slug === 'string' ? file.slug : slugFrom(file.name),
		description: typeof file.description === 'string' ? file.description : '',
		is_enabled: file.is_enabled !== false,
		steps: file.steps,
		allow_sign_in: file.allow_sign_in !== false,
		allow_registration: file.allow_registration !== false,
		allow_password_reset: file.allow_password_reset !== false,
		allow_remember_me: file.allow_remember_me !== false,
		verify_email_on_register: file.verify_email_on_register === true,
		require_verified_email: file.require_verified_email === true,
		allow_email_change: file.allow_email_change === true,
		session_lifetime_hours:
			typeof file.session_lifetime_hours === 'number' ? file.session_lifetime_hours : 24 * 14
	};
}

export function download(name: string, content: string) {
	const url = URL.createObjectURL(new Blob([content], { type: 'application/json' }));
	const link = Object.assign(document.createElement('a'), { href: url, download: name });
	link.click();
	URL.revokeObjectURL(url);
}

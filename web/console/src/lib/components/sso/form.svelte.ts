import type {
	SSOConnection,
	SSOConnectionInput,
	SSOProtocol,
	SSORoleMapping,
	SSOTestInput
} from '#lib/api/index.js';

export type SSOTab = 'connection' | 'provider' | 'signin' | 'users';

/** What stops the connection being saved, and the tab where it is fixed. */
export type Missing = { text: string; tab: SSOTab };

/** The SSO dialog's fields: filled from a connection, edited across its tabs, sent back whole. */
export class SSOForm {
	protocol = $state<SSOProtocol>('oidc');
	name = $state('');
	slug = $state('');
	enabled = $state(false);

	issuer = $state('');
	clientId = $state('');
	/** Never sent back by the server: empty keeps the stored one. */
	clientSecret = $state('');
	scopes = $state('');

	metadataUrl = $state('');
	metadata = $state('');
	nameIdFormat = $state<SSOConnection['name_id_format']>('email');
	signRequests = $state(false);

	domains = $state<string[]>([]);
	enforceDomains = $state(false);
	showOnLogin = $state(true);

	matching = $state<SSOConnection['matching']>('link');
	createUsers = $state(true);
	syncProfile = $state(true);
	emailAttribute = $state('');
	firstNameAttribute = $state('');
	lastNameAttribute = $state('');
	groupsAttribute = $state('');
	mappings = $state<SSORoleMapping[]>([]);
	syncRoles = $state(false);

	/** SSO is required only of addresses at some domain: none, nothing to require. */
	enforced = $derived(this.domains.length > 0 && this.enforceDomains);

	missing = $derived.by((): Missing | null => {
		if (this.name.trim() === '')
			return { text: 'Name the connection to create it.', tab: 'connection' };
		if (this.protocol === 'oidc' && (this.issuer.trim() === '' || this.clientId.trim() === ''))
			return { text: 'Give the issuer and the client ID to create it.', tab: 'connection' };
		if (this.protocol === 'saml' && this.metadataUrl.trim() === '' && this.metadata.trim() === '')
			return {
				text: 'Give the metadata URL or paste the metadata to create it.',
				tab: 'connection'
			};
		if (this.domains.length === 0 && !this.showOnLogin)
			return {
				text: 'Add a domain or turn on the sign-in button, on Domains & sign-in, to create it.',
				tab: 'signin'
			};
		return null;
	});

	constructor(from: SSOConnection | null = null) {
		this.fill(from);
	}

	fill(from: SSOConnection | null): void {
		this.protocol = from?.protocol ?? 'oidc';
		this.name = from?.name ?? '';
		this.slug = from?.slug ?? '';
		this.enabled = from?.is_enabled ?? false;
		this.issuer = from?.issuer ?? '';
		this.clientId = from?.client_id ?? '';
		this.clientSecret = '';
		this.scopes = (from?.scopes ?? []).join(' ');
		this.metadataUrl = from?.metadata_url ?? '';
		// Metadata read from an address is the server's copy, not something typed.
		this.metadata = from?.metadata_url ? '' : (from?.metadata ?? '');
		this.nameIdFormat = from?.name_id_format ?? 'email';
		this.signRequests = from?.sign_requests ?? false;
		this.domains = [...(from?.domains ?? [])];
		this.enforceDomains = from?.enforce_domains ?? false;
		this.showOnLogin = from?.show_on_login ?? true;
		this.matching = from?.matching ?? 'link';
		this.createUsers = from?.create_users ?? true;
		this.syncProfile = from?.sync_profile ?? true;
		this.emailAttribute = from?.email_attribute ?? '';
		this.firstNameAttribute = from?.first_name_attribute ?? '';
		this.lastNameAttribute = from?.last_name_attribute ?? '';
		this.groupsAttribute = from?.groups_attribute ?? '';
		this.mappings = (from?.role_mappings ?? []).map((mapping) => ({ ...mapping }));
		this.syncRoles = from?.sync_roles ?? false;
	}

	/** The body to save. The protocol and identifier are only given once, when it is created. */
	toInput(creating: boolean): SSOConnectionInput {
		const body: SSOConnectionInput = {
			name: this.name.trim(),
			is_enabled: this.enabled,
			domains: this.domains,
			enforce_domains: this.enforced,
			show_on_login: this.showOnLogin,
			matching: this.matching,
			create_users: this.createUsers,
			sync_profile: this.syncProfile,
			email_attribute: this.emailAttribute.trim(),
			first_name_attribute: this.firstNameAttribute.trim(),
			last_name_attribute: this.lastNameAttribute.trim(),
			groups_attribute: this.groupsAttribute.trim(),
			role_mappings: this.mappings.filter((mapping) => mapping.group.trim() !== ''),
			sync_roles: this.syncRoles
		};

		if (creating) {
			body.protocol = this.protocol;
			if (this.slug.trim()) body.slug = this.slug.trim();
		}

		if (this.protocol === 'oidc') {
			body.issuer = this.issuer.trim();
			body.client_id = this.clientId.trim();
			body.scopes = this.scopeList();
			if (this.clientSecret.trim()) body.client_secret = this.clientSecret.trim();
		} else {
			body.metadata_url = this.metadataUrl.trim();
			if (!this.metadataUrl.trim()) body.metadata = this.metadata.trim();
			body.name_id_format = this.nameIdFormat;
			body.sign_requests = this.signRequests;
		}

		return body;
	}

	/** What "Test connection" asks the server to try, before anything is saved. */
	toTest(): SSOTestInput {
		return {
			protocol: this.protocol,
			issuer: this.issuer.trim(),
			scopes: this.scopeList(),
			metadata_url: this.metadataUrl.trim(),
			metadata: this.metadataUrl.trim() ? undefined : this.metadata.trim()
		};
	}

	private scopeList(): string[] {
		return this.scopes.split(/\s+/).filter(Boolean);
	}
}

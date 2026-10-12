/** The shapes the API returns. They mirror the response.go files under
    internal/api/; change them together. */

export type Admin = {
	id: string;
	username: string;
	email: string;
	full_name: string;
	first_name: string;
	last_name: string;
	avatar_url: string;
	status: string;
	roles: string[];
	last_login_at?: string;
	/** What the whole-panel roles allow. The panel shows and hides its
	    controls by these; the server checks them regardless. */
	permissions: AdminPermissionName[];
	scoped_permissions: Record<string, AdminPermissionName[]>;
	is_super_admin: boolean;
	mfa_enabled: boolean;
};

export type SessionState = 'none' | 'signed_in' | 'mfa' | 'enroll';

export type LoginResult =
	{ admin: Admin; next?: undefined } | { next: 'mfa' | 'enroll'; admin?: undefined };

export type MfaStatus = {
	enabled: boolean;
	required: boolean;
	confirmed_at: string | null;
	last_used_at: string | null;
	recovery_codes_left: number;
};

export type MfaEnrolment = { secret: string; uri: string };

/** The names in the admin permission catalog. They mirror the constants in
    internal/model/admin_permission.go; change them together. */
export type AdminPermissionName =
	| 'activity.read'
	| 'users.read'
	| 'users.write'
	| 'user_fields.write'
	| 'apis.read'
	| 'apis.write'
	| 'organization.read'
	| 'organization.write'
	| 'social.read'
	| 'social.write'
	| 'login_flows.read'
	| 'login_flows.write'
	| 'sso.read'
	| 'sso.write'
	| 'languages.read'
	| 'languages.write'
	| 'applications.read'
	| 'applications.write'
	| 'user_roles.write'
	| 'role_assignments.write';

export type AdminPermission = {
	name: AdminPermissionName;
	group: string;
	description: string;
	scopable: boolean;
};

export type AdminStatus = 'active' | 'suspended' | 'disabled' | 'invited';

export type AdminAssignment = {
	id: string;
	role: Reference;
	application: Reference | null;
};

export type AdminRecord = {
	id: string;
	username: string;
	email: string;
	first_name: string;
	last_name: string;
	full_name: string;
	status: AdminStatus;
	assignments: AdminAssignment[];
	permissions: AdminPermissionName[];
	scoped_permissions: Record<string, AdminPermissionName[]>;
	is_super_admin: boolean;
	mfa_enabled: boolean;
	last_login_at: string | null;
	last_login_ip: string;
	created_at: string;
	updated_at: string;
};

export type AdminAssignmentInput = {
	role_id: string;
	application_id: string | null;
};

export type AdminInput = {
	email: string;
	first_name: string;
	last_name: string;
	status: Exclude<AdminStatus, 'invited'>;
	password?: string;
	confirm_password?: string;
	assignments: AdminAssignmentInput[];
};

export type AdminPage = {
	admins: AdminRecord[];
	total: number;
	limit: number;
	offset: number;
};

export type AdminRole = {
	id: string;
	name: string;
	description: string;
	permissions: AdminPermissionName[];
	is_builtin: boolean;
	admin_count: number;
	created_at: string;
	updated_at: string;
};

export type AdminRoleInput = {
	name: string;
	description: string;
	permissions: AdminPermissionName[];
};

export type ApplicationType = 'web' | 'spa' | 'native' | 'm2m';

export type AuthMethod = 'client_secret_basic' | 'client_secret_post' | 'none';

export type GrantType = 'authorization_code' | 'refresh_token' | 'client_credentials';

export type Scope = 'openid' | 'profile' | 'email' | 'offline_access' | 'roles';

export type Application = {
	id: string;
	name: string;
	description: string;
	type: ApplicationType;
	logo_url: string;
	website_url: string;
	privacy_url: string;
	terms_url: string;
	client_id: string;
	client_id_issued_at: number;
	has_secret: boolean;
	secret_hint: string;
	secret_created_at: string | null;
	token_auth_method: AuthMethod;
	grant_types: GrantType[];
	response_types: string[];
	redirect_uris: string[];
	post_logout_redirect_uris: string[];
	scopes: Scope[];
	require_pkce: boolean;
	access_token_lifetime: number;
	id_token_lifetime: number;
	refresh_token_lifetime: number;
	assert_roles: boolean;
	require_role_assignment: boolean;
	is_enabled: boolean;
	allow_registration: boolean;
	login_flow_id: string | null;
	role_count: number;
	created_at: string;
	updated_at: string;
};

export type ApplicationInput = Pick<
	Application,
	| 'name'
	| 'description'
	| 'type'
	| 'logo_url'
	| 'website_url'
	| 'privacy_url'
	| 'terms_url'
	| 'token_auth_method'
	| 'grant_types'
	| 'redirect_uris'
	| 'post_logout_redirect_uris'
	| 'scopes'
	| 'require_pkce'
	| 'access_token_lifetime'
	| 'id_token_lifetime'
	| 'refresh_token_lifetime'
	| 'assert_roles'
	| 'require_role_assignment'
	| 'is_enabled'
	| 'allow_registration'
> & {
	login_flow_id?: string;
};

export type ApplicationWithSecret = {
	application: Application;
	client_secret?: string;
};

export type ApplicationPage = {
	applications: Application[];
	total: number;
	limit: number;
	offset: number;
};

export type SetupInput = {
	email: string;
	password: string;
	first_name: string;
	last_name: string;
};

export type ActivityTarget = {
	type: string;
	id: string;
	name?: string;
};

export type ActivityEvent = {
	id: string;
	action: string;
	actor: string;
	ip: string;
	target: ActivityTarget | null;
	detail?: string;
	created_at: string;
};

export type LogEntry = ActivityEvent & {
	user_agent: string;
};

export type LogPage = {
	logs: LogEntry[];
	next: string;
};

export type LogFilter = {
	q?: string;
	actions?: string[];
	actor?: string;
	from?: string;
	to?: string;
};

export type OverviewDays = 7 | 14 | 30 | 90;

export type Overview = {
	days: OverviewDays;
	counts: {
		users: number;
		active_users: number;
		new_users: number;
		applications: number;
		enabled_applications: number;
		apis: number;
		api_scopes: number;
		user_roles: number;
		admins: number;
		locked_admins: number;
		active_sessions: number;
	};
	sign_ins: {
		since: string;
		succeeded: number;
		failed: number;
		blocked: number;
	};
	daily: { day: string; events: number; failures: number }[];
	top_actors: { actor: string; events: number }[];
	activity: ActivityEvent[];
};

export type AdminSession = {
	id: string;
	ip: string;
	user_agent: string;
	created_at: string;
	expires_at: string;
	active: boolean;
	current: boolean;
};

export type FieldType = 'text' | 'number' | 'bool' | 'email' | 'date';

export type FieldRules = {
	is_required: boolean;
	is_unique: boolean;
	min: number | null;
	max: number | null;
	starts_with: string;
};

export type UserField = FieldRules & {
	id?: string;
	name: string;
	label: string;
	type: FieldType;
	position: number;
	is_builtin: boolean;
};

export type FieldInput = FieldRules & {
	name: string;
	label: string;
	type: FieldType;
};

export type UserBuiltins = {
	email: string;
	is_email_verified: boolean;
	first_name: string;
	last_name: string;
	is_active: boolean;
	is_password_temporary: boolean;
};

export type SocialAccount = {
	id: string;
	provider: string;
	slug: string;
	kind: SocialKind;
	email: string;
	connected_at: string;
	last_login_at: string | null;
};

export type UserRecord = UserBuiltins & {
	id: string;
	has_password: boolean;
	roles: UserRoleRef[];
	social_accounts: SocialAccount[];
	data: Record<string, unknown> | null;
	created_at: string;
	updated_at: string;
};

export type UserInput = UserBuiltins & {
	password?: string;
	confirm_password?: string;
	data: Record<string, unknown>;
};

export type UserPage = {
	users: UserRecord[];
	total: number;
	limit: number;
	offset: number;
};

export type Reference = {
	id: string;
	name: string;
};

export type UserRoleRef = Reference & {
	application_id: string | null;
};

export type RoleMapping = {
	id: string;
	name: string;
	description: string;
	application: { id: string; name: string; client_id: string } | null;
	composite: boolean;
	assigned: boolean;
	via: Reference[];
};

export type Role = {
	id: string;
	application_id: string | null;
	name: string;
	description: string;
	is_default: boolean;
	inherits: UserRoleRef[];
	inherited_roles: UserRoleRef[];
	api_scopes: APIScopeRef[];
	user_count: number;
	created_at: string;
	updated_at: string;
};

export type RoleInput = {
	application_id: string | null;
	name: string;
	description: string;
	is_default: boolean;
	inherits: string[];
	api_scopes?: string[];
};

export type RolePage = {
	roles: Role[];
	total: number;
	limit: number;
	offset: number;
};

export type API = {
	id: string;
	name: string;
	identifier: string;
	description: string;
	enforce_roles: boolean;
	scopes: APIScope[];
	signing_algorithm: SigningAlgorithm;
	token_lifetime: number;
	allow_offline_access: boolean;
	application_count: number;
	role_count: number;
	issuer: string;
	jwks_uri: string;
	system: '' | 'admin' | 'account';
	created_at: string;
	updated_at: string;
};

export type SigningAlgorithm = 'RS256' | 'PS256' | 'ES256';

export type APIScope = {
	id: string;
	name: string;
	description: string;
	is_default: boolean;
};

export type APIApplication = {
	id: string;
	name: string;
	type: ApplicationType;
	client_id: string;
	is_enabled: boolean;
	authorized: boolean;
	allowed: string[];
};

export type APILogEntry = {
	id: string;
	action: string;
	actor: string;
	ip: string;
	metadata: Record<string, unknown> | null;
	created_at: string;
};

export type APIScopeRef = {
	id: string;
	name: string;
	api_id: string;
};

export type APIInput = {
	name: string;
	identifier: string;
	description: string;
	enforce_roles: boolean;
	signing_algorithm: SigningAlgorithm;
	token_lifetime: number;
	allow_offline_access: boolean;
	scopes: { id?: string; name: string; description: string; is_default: boolean }[];
};

export type APIAccess = {
	id: string;
	name: string;
	identifier: string;
	enforce_roles: boolean;
	authorized: boolean;
	scopes: (APIScope & { allowed: boolean })[];
};

export type TokenPreviewInput = {
	user_id: string | null;
	audience: string;
	scope: string;
};

export type ScopeDecision = {
	scope: string;
	kind: 'openid' | 'api';
	granted: boolean;
	reason: string;
	is_default?: boolean;
};

export type TokenPreview = {
	issued: boolean;
	reason?: string;
	decisions: ScopeDecision[];
	access_token_header?: Record<string, unknown>;
	access_token?: Record<string, unknown>;
	id_token?: Record<string, unknown>;
};

export type Organization = {
	name: string;
	slug: string;
	logo_url: string;
	support_email: string;
	support_phone: string;
	terms_url: string;
	privacy_url: string;
	timezone: string;
	created_at: string;
};

export type OrganizationBrand = Pick<Organization, 'name' | 'logo_url'>;

export type OrganizationSettings = Omit<Organization, 'created_at'>;

export type OrganizationInput = Partial<OrganizationSettings>;

export type SocialKind = 'google' | 'apple' | 'facebook' | 'yandex' | 'vk' | 'oidc' | 'oauth2';

export type SocialTokenAuth = 'basic' | 'post';

export type SocialSpec = {
	kind: SocialKind;
	label: string;
	authorize_url: string;
	token_url: string;
	userinfo_url: string;
	scopes: string[];
	custom: boolean;
	signed_secret: boolean;
	token_auth: SocialTokenAuth;
	docs: string;
};

export type SocialProvider = {
	id: string;
	kind: SocialKind;
	slug: string;
	name: string;
	client_id: string;
	has_client_secret: boolean;
	team_id: string;
	key_id: string;
	has_private_key: boolean;
	scopes: string[];
	token_auth: SocialTokenAuth | '';
	token_auth_used: SocialTokenAuth;
	authorize_url: string;
	token_url: string;
	userinfo_url: string;
	is_enabled: boolean;
	link_verified_emails: boolean;
	allow_registration: boolean;
	position: number;
	callback_url: string;
	identities: number;
	created_at: string;
};

export type SocialProviderInput = {
	kind?: SocialKind;
	slug?: string;
	name?: string;
	client_id?: string;
	client_secret?: string;
	private_key?: string;
	team_id?: string;
	key_id?: string;
	scopes?: string[];
	token_auth?: SocialTokenAuth | '';
	authorize_url?: string;
	token_url?: string;
	userinfo_url?: string;
	is_enabled?: boolean;
	link_verified_emails?: boolean;
	allow_registration?: boolean;
};

export type AdminSecurity = {
	require_mfa: boolean;
	administrators: number;
	with_mfa: number;
};

/** One step a login flow can be made of. The names mirror the constants in
    internal/model/login_flow.go; change them together. */
export type LoginStep =
	'identifier' | 'password' | 'social' | 'email_code' | 'totp' | 'terms' | 'consent';

export type LoginStepSpec = {
	step: LoginStep;
	label: string;
	description: string;
	fixed: boolean;
	implemented: boolean;
};

export type LoginFlow = {
	id: string;
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
	applications: number;
	planned: LoginStep[];
	created_at: string;
	updated_at: string;
};

export type LoginFlowInput = {
	name?: string;
	slug?: string;
	description?: string;
	is_default?: boolean;
	is_enabled?: boolean;
	steps?: LoginStep[];
	allow_sign_in?: boolean;
	allow_registration?: boolean;
	allow_password_reset?: boolean;
	allow_remember_me?: boolean;
	verify_email_on_register?: boolean;
	require_verified_email?: boolean;
	allow_email_change?: boolean;
	session_lifetime_hours?: number;
};

export type UserSessionRecord = {
	id: string;
	user: { id: string; email: string; name: string };
	ip: string;
	user_agent: string;
	signed_in_at: string;
	expires_at: string;
};

/** A page of sessions, and the session to continue after when there are
    more. There is no total: counting every session on every visit is what
    the page is built to avoid. */
export type UserSessionPage = {
	sessions: UserSessionRecord[];
	next?: string;
};

export type LocaleApp = 'id';

export type Language = {
	code: string;
	name: string;
	native_name: string;
	apps: LocaleApp[];
	coverage: Partial<Record<LocaleApp, number>>;
	missing: Partial<Record<LocaleApp, number>>;
	is_enabled: boolean;
	is_default: boolean;
	position: number;
	base: boolean;
	shipped: boolean;
	updated_at: string;
};

export type ShippedLanguage = {
	code: string;
	name: string;
	native_name: string;
};

export type LanguageList = {
	languages: Language[];
	apps: LocaleApp[];
	shipped: ShippedLanguage[];
};

export type LanguageInput = {
	name?: string;
	native_name?: string;
	is_enabled?: boolean;
	is_default?: boolean;
	position?: number;
};

export type NewLanguageInput = {
	code: string;
	name: string;
	native_name: string;
	is_enabled?: boolean;
	is_default?: boolean;
	copy_from?: string;
};

export type Translation = {
	app: LocaleApp;
	keys: string[];
	base: Record<string, string>;
	messages: Record<string, string>;
};

export type SSOProtocol = 'oidc' | 'saml';

export type SSORoleMapping = { group: string; role_id: string };

export type SSOConnection = {
	id: string;
	slug: string;
	name: string;
	protocol: SSOProtocol;
	is_enabled: boolean;
	domains: string[];
	enforce_domains: boolean;
	show_on_login: boolean;

	issuer: string;
	client_id: string;
	has_client_secret: boolean;
	scopes: string[];

	metadata_url: string;
	metadata: string;
	name_id_format: 'email' | 'persistent' | 'unspecified';
	sign_requests: boolean;

	matching: 'link' | 'deny';
	create_users: boolean;
	sync_profile: boolean;
	email_attribute: string;
	first_name_attribute: string;
	last_name_attribute: string;
	groups_attribute: string;
	role_mappings: SSORoleMapping[];
	sync_roles: boolean;

	users: number;
	service_provider: {
		callback_url?: string;
		acs_url?: string;
		entity_id?: string;
		metadata_url?: string;
		certificate?: string;
	};
	identity_provider?: SSOIdentityProvider;
	created_at: string;
	updated_at: string;
};

export type SSOIdentityProvider = {
	entity_id: string;
	sso_url: string;
	certificates: number;
	certificate_expires?: string;
};

export type SSOConnectionInput = Partial<
	Omit<
		SSOConnection,
		| 'id'
		| 'has_client_secret'
		| 'users'
		| 'service_provider'
		| 'identity_provider'
		| 'created_at'
		| 'updated_at'
	>
> & { client_secret?: string };

export type SSOTestResult = {
	issuer?: string;
	authorization_endpoint?: string;
	token_endpoint?: string;
	jwks_uri?: string;
	unsupported_scopes?: string[];
	identity_provider?: SSOIdentityProvider;
	metadata?: string;
};

/** How the connection to the mail server is protected. The names mirror the
    constants in internal/model/mail.go; change them together. */
export type MailEncryption = 'starttls' | 'tls' | 'none';

export type MailSettings = {
	is_enabled: boolean;
	host: string;
	port: number;
	encryption: MailEncryption;
	username: string;
	from_address: string;
	from_name: string;
	has_password: boolean;
};

export type MailResponse = {
	mail: MailSettings;
	encryptions: MailEncryption[];
};

export type MailSettingsInput = Partial<Omit<MailSettings, 'has_password'>> & {
	password?: string;
};

export type MailTestInput = MailSettingsInput & { to: string };

export type MailParam = {
	name: string;
	description: string;
};

export type MailMessageSpec = {
	kind: string;
	label: string;
	description: string;
	subject_key: string;
	body_key: string;
	params: MailParam[];
};

export type MailLanguageContent = {
	code: string;
	name: string;
	native_name: string;
	offered: boolean;
	messages: Record<string, string>;
	base: Record<string, string>;
};

export type MailContent = {
	messages: MailMessageSpec[];
	languages: MailLanguageContent[];
};

export type OTPSettings = {
	code_length: number;
	lifetime_minutes: number;
	max_attempts: number;
	resend_seconds: number;
};

export type OTPLimits = {
	min_length: number;
	max_length: number;
	max_lifetime_minutes: number;
	max_attempts: number;
	max_resend_seconds: number;
};

export type OTPFlow = {
	id: string;
	name: string;
	slug: string;
	is_enabled: boolean;
	is_default: boolean;
};

export type OTPResponse = {
	otp: OTPSettings;
	limits: OTPLimits;
	flows: OTPFlow[];
};

export type OTPSettingsInput = Partial<OTPSettings>;

export type CacheDatabaseName = 'cache' | 'sessions';

export type CacheKind = 'entry' | 'stale' | 'generation' | 'session' | 'ratelimit' | 'other';

export type CacheGroup = {
	name: string;
	generation: number;
	entries: number;
	stale: number;
	known: boolean;
};

export type CacheDatabase = {
	name: CacheDatabaseName;
	number: number;
	available: boolean;
	keys: number;
	kinds: Partial<Record<CacheKind, number>>;
	groups: CacheGroup[];
	sessions: Record<string, number>;
	editable: boolean;
	memory_bytes: number;
	version: string;
};

export type CacheOverview = {
	configured: boolean;
	databases: CacheDatabase[];
};

export type CacheKey = {
	name: string;
	kind: CacheKind;
	group: string;
	entry: string;
	ttl_seconds: number;
	size_bytes: number;
	editable: boolean;
};

export type CacheKeyValue = CacheKey & {
	type: string;
	value: unknown;
};

export type CacheKeyPage = {
	keys: CacheKey[];
	cursor: string;
};

export type CacheKeyQuery = {
	kind?: CacheKind | '';
	group?: string;
	search?: string;
	cursor?: string;
	limit?: number;
};

export type CacheKeyInput = {
	value: unknown;
	ttl_seconds?: number;
};

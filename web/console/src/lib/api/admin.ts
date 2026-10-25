import { api, apiPath, type Fetch } from './client';
import type {
	API,
	LogFilter,
	LogPage,
	Overview,
	OverviewDays,
	APIApplication,
	APILogEntry,
	APIAccess,
	APIInput,
	TokenPreview,
	TokenPreviewInput,
	Admin,
	AdminSession,
	ApplicationInput,
	ApplicationPage,
	ApplicationWithSecret,
	AdminInput,
	AdminPage,
	AdminPermission,
	AdminRecord,
	AdminRole,
	AdminRoleInput,
	UserSessionPage,
	Language,
	LanguageInput,
	LanguageList,
	LocaleApp,
	NewLanguageInput,
	Translation,
	LoginFlow,
	LoginFlowInput,
	LoginResult,
	LoginStepSpec,
	MailContent,
	MailLanguageContent,
	MailResponse,
	MailSettingsInput,
	MailTestInput,
	MfaEnrolment,
	MfaStatus,
	SessionState,
	FieldInput,
	FieldRules,
	AdminSecurity,
	Organization,
	OrganizationBrand,
	OrganizationInput,
	OTPResponse,
	OTPSettingsInput,
	CacheDatabaseName,
	CacheKeyInput,
	CacheKeyPage,
	CacheKeyQuery,
	CacheKeyValue,
	CacheOverview,
	SocialProvider,
	SocialProviderInput,
	SocialSpec,
	SSOConnection,
	SSOConnectionInput,
	SSOTestInput,
	SSOTestResult,
	Role,
	RoleInput,
	RoleMapping,
	RolePage,
	SetupInput,
	UserField,
	UserInput,
	UserPage,
	UserRecord
} from './types';

function logQuery(filter: LogFilter, extra: Record<string, string> = {}): string {
	const params = new URLSearchParams();

	if (filter.q) params.set('q', filter.q);
	for (const action of filter.actions ?? []) params.append('action', action);
	if (filter.actor) params.set('actor', filter.actor);
	if (filter.from) params.set('from', filter.from);
	if (filter.to) params.set('to', filter.to);
	for (const [key, value] of Object.entries(extra)) params.set(key, value);

	const text = params.toString();
	return text ? `?${text}` : '';
}

export const activityApi = {
	overview: (days: OverviewDays, fetcher?: Fetch) =>
		api.get<Overview>(`/admin/overview?days=${days}`, fetcher),

	logs: (filter: LogFilter, before = '', limit = 50, fetcher?: Fetch) =>
		api.get<LogPage>(
			`/admin/logs${logQuery(filter, { limit: String(limit), ...(before ? { before } : {}) })}`,
			fetcher
		),

	/** Where the browser downloads the entries the filter matches, as CSV.
	    A link rather than a call: the browser saves the file itself. */
	exportUrl: (filter: LogFilter) => `/api/v1/admin/logs/export${logQuery(filter)}`
};

export const setupApi = {
	status: (fetcher?: Fetch) => api.get<{ required: boolean }>('/admin/setup', fetcher),

	create: (input: SetupInput) => api.post<{ admin: { id: string } }>('/admin/setup', input)
};

export const adminApi = {
	login: (username: string, password: string) =>
		api.post<LoginResult>('/admin/auth/login', { username, password }),

	session: (fetcher?: Fetch) =>
		api.get<{ state: SessionState; mfa_required: boolean }>('/admin/auth/session', fetcher),

	verifyMfa: (code: string) => api.post<{ admin: Admin }>('/admin/auth/mfa', { code }),

	logout: () => api.post<{ status: string }>('/admin/auth/logout'),

	me: (fetcher?: Fetch) =>
		api.get<{ admin: Admin; organization: OrganizationBrand }>('/admin/me', fetcher),

	profileSessions: (fetcher?: Fetch) =>
		api.get<{ sessions: AdminSession[] | null }>('/admin/sessions', fetcher),

	endSession: (id: string) => api.delete<void>(`/admin/sessions/${encodeURIComponent(id)}`),

	endOtherSessions: () => api.delete<{ ended: number }>('/admin/sessions'),

	updateProfile: (body: {
		first_name: string;
		last_name: string;
		email: string;
		avatar_url: string;
		current_password?: string;
	}) => api.patch<{ admin: Admin }>('/admin/me', body),

	changePassword: (current_password: string, new_password: string) =>
		api.post<{ status: string }>('/admin/me/password', { current_password, new_password })
};

export const usersApi = {
	list: (
		params: { search?: string; verified?: string; role?: string; offset?: number; limit?: number },
		fetcher?: Fetch
	) => {
		const query = new URLSearchParams();
		if (params.search) query.set('search', params.search);
		if (params.verified === 'true' || params.verified === 'false') {
			query.set('verified', params.verified);
		}
		if (params.role) query.set('role', params.role);
		if (params.offset) query.set('offset', String(params.offset));
		if (params.limit) query.set('limit', String(params.limit));

		return api.get<UserPage>(`/admin/users?${query}`, fetcher);
	},

	fields: (fetcher?: Fetch) => api.get<{ fields: UserField[] }>('/admin/user-fields', fetcher),

	create: (input: UserInput) => api.post<{ user: UserRecord }>('/admin/users', input),

	roleMappings: (id: string, fetcher?: Fetch) =>
		api.get<{ roles: RoleMapping[] }>(`/admin/users/${id}/role-mappings`, fetcher),

	assignRoles: (id: string, roles: string[]) =>
		api.post<{ roles: RoleMapping[] }>(`/admin/users/${id}/role-mappings`, { roles }),

	unassignRole: (id: string, role: string) =>
		api.delete<{ roles: RoleMapping[] }>(`/admin/users/${id}/role-mappings/${role}`),

	update: (id: string, input: UserInput) =>
		api.patch<{ user: UserRecord }>(`/admin/users/${id}`, input),

	remove: (id: string) => api.delete<void>(`/admin/users/${id}`),

	addField: (field: FieldInput) => api.post<{ field: UserField }>('/admin/user-fields', field),

	updateField: (id: string, rules: FieldRules & { label: string }) =>
		api.patch<{ field: UserField }>(`/admin/user-fields/${id}`, rules),

	removeField: (id: string) => api.delete<void>(`/admin/user-fields/${id}`),

	disconnect: (id: string, identity: string) =>
		api.delete<void>(`/admin/users/${id}/social-accounts/${identity}`),

	resetTwoStep: (id: string) => api.delete<void>(`/admin/users/${id}/two-step`)
};

export const organizationApi = {
	get: (fetcher?: Fetch) => api.get<{ organization: Organization }>('/admin/organization', fetcher),

	update: (input: OrganizationInput) =>
		api.patch<{ organization: Organization }>('/admin/organization', input),

	/** Where the panel loads the app logo, by its version so the browser keeps it; "" while there
	 *  is none. */
	logoUrl: (version: string) =>
		version ? apiPath(`/admin/organization/logo?v=${encodeURIComponent(version)}`) : '',

	/** The file goes as base64 in JSON: the API takes no other body for a change. */
	uploadLogo: async (file: File) =>
		api.put<{ organization: Organization }>('/admin/organization/logo', {
			data: await base64Of(file)
		}),

	removeLogo: () => api.delete<{ organization: Organization }>('/admin/organization/logo')
};

function base64Of(file: File): Promise<string> {
	return new Promise((resolve, reject) => {
		const reader = new FileReader();
		reader.onload = () =>
			resolve(String(reader.result).slice(String(reader.result).indexOf(',') + 1));
		reader.onerror = () => reject(reader.error);
		reader.readAsDataURL(file);
	});
}

export const mailApi = {
	get: (fetcher?: Fetch) => api.get<MailResponse>('/admin/mail', fetcher),

	update: (input: MailSettingsInput) => api.patch<MailResponse>('/admin/mail', input),

	test: (input: MailTestInput) =>
		api.post<{ sent: boolean; to: string }>('/admin/mail/test', input),

	content: (fetcher?: Fetch) => api.get<MailContent>('/admin/mail/content', fetcher),

	saveContent: (code: string, messages: Record<string, string>) =>
		api.put<{ language: MailLanguageContent }>(`/admin/mail/content/${code}`, { messages })
};

export const otpApi = {
	get: (fetcher?: Fetch) => api.get<OTPResponse>('/admin/otp', fetcher),

	update: (input: OTPSettingsInput) => api.patch<OTPResponse>('/admin/otp', input)
};

export const socialApi = {
	list: (fetcher?: Fetch) =>
		api.get<{ providers: SocialProvider[]; kinds: SocialSpec[] }>(
			'/admin/social-providers',
			fetcher
		),

	create: (input: SocialProviderInput) =>
		api.post<{ provider: SocialProvider }>('/admin/social-providers', input),

	update: (id: string, input: SocialProviderInput) =>
		api.patch<{ provider: SocialProvider }>(`/admin/social-providers/${id}`, input),

	remove: (id: string) => api.delete<void>(`/admin/social-providers/${id}`),

	secret: (id: string) =>
		api.get<{ secret: string; kind: string }>(`/admin/social-providers/${id}/secret`)
};

export const flowsApi = {
	list: (fetcher?: Fetch) =>
		api.get<{ flows: LoginFlow[]; step_kinds: LoginStepSpec[] }>('/admin/login-flows', fetcher),

	create: (input: LoginFlowInput) => api.post<{ flow: LoginFlow }>('/admin/login-flows', input),

	update: (id: string, input: LoginFlowInput) =>
		api.patch<{ flow: LoginFlow }>(`/admin/login-flows/${id}`, input),

	remove: (id: string) => api.delete<void>(`/admin/login-flows/${id}`)
};

export const sessionsApi = {
	list: (
		params: { search?: string; user?: string; after?: string; limit?: number } = {},
		fetcher?: Fetch
	) => {
		const query = new URLSearchParams();
		if (params.search) query.set('search', params.search);
		if (params.user) query.set('user', params.user);
		if (params.after) query.set('after', params.after);
		if (params.limit) query.set('limit', String(params.limit));

		return api.get<UserSessionPage>(
			query.size ? `/admin/user-sessions?${query}` : '/admin/user-sessions',
			fetcher
		);
	},

	end: (id: string) => api.delete<void>(`/admin/user-sessions/${id}`),

	signOutUser: (userId: string) =>
		api.delete<{ sessions: number; tokens: number }>(`/admin/users/${userId}/sessions`)
};

export const languagesApi = {
	list: (fetcher?: Fetch) => api.get<LanguageList>('/admin/languages', fetcher),

	create: (input: NewLanguageInput) => api.post<{ language: Language }>('/admin/languages', input),

	update: (code: string, input: LanguageInput) =>
		api.patch<{ language: Language }>(`/admin/languages/${encodeURIComponent(code)}`, input),

	remove: (code: string) => api.delete<void>(`/admin/languages/${encodeURIComponent(code)}`),

	translation: (code: string, app: LocaleApp, fetcher?: Fetch) =>
		api.get<Translation>(
			`/admin/languages/${encodeURIComponent(code)}/translations/${app}`,
			fetcher
		),

	saveTranslation: (code: string, app: LocaleApp, messages: Record<string, string>) =>
		api.put<{ language: Language; ignored: number }>(
			`/admin/languages/${encodeURIComponent(code)}/translations/${app}`,
			{ messages }
		)
};

export const ssoApi = {
	list: (fetcher?: Fetch) =>
		api.get<{ connections: SSOConnection[] }>('/admin/sso-connections', fetcher),

	create: (input: SSOConnectionInput) =>
		api.post<{ connection: SSOConnection }>('/admin/sso-connections', input),

	update: (id: string, input: SSOConnectionInput) =>
		api.patch<{ connection: SSOConnection }>(`/admin/sso-connections/${id}`, input),

	remove: (id: string) => api.delete<void>(`/admin/sso-connections/${id}`),

	test: (input: SSOTestInput) => api.post<SSOTestResult>('/admin/sso-connections/test', input),

	refreshMetadata: (id: string) =>
		api.post<{ connection: SSOConnection }>(`/admin/sso-connections/${id}/refresh-metadata`)
};

export const rolesApi = {
	list: (
		params: {
			scope?: 'global' | 'application';
			application?: string;
			search?: string;
			default?: string;
			offset?: number;
			limit?: number;
		},
		fetcher?: Fetch
	) => {
		const query = new URLSearchParams();
		if (params.scope) query.set('scope', params.scope);
		if (params.application) query.set('application', params.application);
		if (params.search) query.set('search', params.search);
		if (params.default === 'true' || params.default === 'false') {
			query.set('default', params.default);
		}
		if (params.offset) query.set('offset', String(params.offset));
		if (params.limit) query.set('limit', String(params.limit));

		return api.get<RolePage>(`/admin/user-roles?${query}`, fetcher);
	},

	create: (input: RoleInput) => api.post<{ role: Role }>('/admin/user-roles', input),

	update: (id: string, input: RoleInput) =>
		api.patch<{ role: Role }>(`/admin/user-roles/${id}`, input),

	remove: (id: string) => api.delete<void>(`/admin/user-roles/${id}`)
};

export const mfaApi = {
	status: (fetcher?: Fetch) => api.get<{ mfa: MfaStatus }>('/admin/mfa', fetcher),

	begin: (currentCode?: string) =>
		api.post<{ enrolment: MfaEnrolment }>(
			'/admin/mfa/totp',
			currentCode ? { code: currentCode } : undefined
		),

	confirm: (code: string) =>
		api.post<{ recovery_codes: string[] }>('/admin/mfa/totp/confirm', { code }),

	disable: (code: string) => api.delete<void>('/admin/mfa/totp', { code }),

	recoveryCodes: (code: string) =>
		api.post<{ recovery_codes: string[] }>('/admin/mfa/recovery-codes', { code })
};

export const adminsApi = {
	list: (
		params: { search?: string; status?: string; role?: string; offset?: number; limit?: number },
		fetcher?: Fetch
	) => {
		const query = new URLSearchParams();
		if (params.search) query.set('search', params.search);
		if (params.status) query.set('status', params.status);
		if (params.role) query.set('role', params.role);
		if (params.offset) query.set('offset', String(params.offset));
		if (params.limit) query.set('limit', String(params.limit));

		return api.get<AdminPage>(`/admin/admins?${query}`, fetcher);
	},

	create: (input: AdminInput) => api.post<{ admin: AdminRecord }>('/admin/admins', input),

	update: (id: string, input: AdminInput) =>
		api.patch<{ admin: AdminRecord }>(`/admin/admins/${id}`, input),

	remove: (id: string) => api.delete<void>(`/admin/admins/${id}`),

	security: (fetcher?: Fetch) => api.get<AdminSecurity>('/admin/security', fetcher),

	updateSecurity: (input: { require_mfa: boolean }) =>
		api.patch<AdminSecurity>('/admin/security', input),

	resetMfa: (id: string) => api.delete<{ admin: AdminRecord }>(`/admin/admins/${id}/mfa`),

	roles: (search = '', fetcher?: Fetch) =>
		api.get<{ roles: AdminRole[]; total: number }>(
			`/admin/admin-roles${search ? `?search=${encodeURIComponent(search)}` : ''}`,
			fetcher
		),

	createRole: (input: AdminRoleInput) => api.post<{ role: AdminRole }>('/admin/admin-roles', input),

	updateRole: (id: string, input: AdminRoleInput) =>
		api.patch<{ role: AdminRole }>(`/admin/admin-roles/${id}`, input),

	removeRole: (id: string) => api.delete<void>(`/admin/admin-roles/${id}`),

	permissions: (fetcher?: Fetch) =>
		api.get<{ permissions: AdminPermission[] }>('/admin/admin-permissions', fetcher)
};

export const applicationsApi = {
	list: (
		params: { search?: string; type?: string; enabled?: string; offset?: number; limit?: number },
		fetcher?: Fetch
	) => {
		const query = new URLSearchParams();
		if (params.search) query.set('search', params.search);
		if (params.type) query.set('type', params.type);
		if (params.enabled === 'true' || params.enabled === 'false')
			query.set('enabled', params.enabled);
		if (params.offset) query.set('offset', String(params.offset));
		if (params.limit) query.set('limit', String(params.limit));

		return api.get<ApplicationPage>(`/admin/applications?${query}`, fetcher);
	},

	create: (input: ApplicationInput) =>
		api.post<ApplicationWithSecret>('/admin/applications', input),

	update: (id: string, input: ApplicationInput) =>
		api.patch<ApplicationWithSecret>(`/admin/applications/${id}`, input),

	apiAccess: (id: string, fetcher?: Fetch) =>
		api.get<{ apis: APIAccess[] }>(`/admin/applications/${id}/apis`, fetcher),

	authorizeAPI: (id: string, apiId: string, scopes: string[]) =>
		api.put<{ apis: APIAccess[] }>(`/admin/applications/${id}/apis/${apiId}`, { scopes }),

	revokeAPI: (id: string, apiId: string) =>
		api.delete<{ apis: APIAccess[] }>(`/admin/applications/${id}/apis/${apiId}`),

	previewToken: (id: string, input: TokenPreviewInput) =>
		api.post<{ preview: TokenPreview }>(`/admin/applications/${id}/token-preview`, input),

	rotateSecret: (id: string) => api.post<ApplicationWithSecret>(`/admin/applications/${id}/secret`),

	remove: (id: string) => api.delete<void>(`/admin/applications/${id}`)
};

export const apisApi = {
	list: (search = '', fetcher?: Fetch) =>
		api.get<{ apis: API[]; total: number }>(
			`/admin/apis${search ? `?search=${encodeURIComponent(search)}` : ''}`,
			fetcher
		),

	get: (id: string, fetcher?: Fetch) => api.get<{ api: API }>(`/admin/apis/${id}`, fetcher),

	applications: (id: string, fetcher?: Fetch) =>
		api.get<{ applications: APIApplication[] }>(`/admin/apis/${id}/applications`, fetcher),

	logs: (id: string, fetcher?: Fetch) =>
		api.get<{ logs: APILogEntry[] }>(`/admin/apis/${id}/logs`, fetcher),

	create: (input: APIInput) => api.post<{ api: API }>('/admin/apis', input),

	update: (id: string, input: APIInput) => api.patch<{ api: API }>(`/admin/apis/${id}`, input),

	remove: (id: string) => api.delete<void>(`/admin/apis/${id}`)
};

export const cacheApi = {
	overview: (fetcher?: Fetch) => api.get<CacheOverview>('/admin/cache', fetcher),

	keys: (database: CacheDatabaseName, query: CacheKeyQuery = {}, fetcher?: Fetch) =>
		api.get<CacheKeyPage>(`/admin/cache/${database}/keys${cacheQuery(query)}`, fetcher),

	key: (database: CacheDatabaseName, name: string, fetcher?: Fetch) =>
		api.get<{ key: CacheKeyValue }>(
			`/admin/cache/${database}/key?name=${encodeURIComponent(name)}`,
			fetcher
		),

	update: (database: CacheDatabaseName, name: string, input: CacheKeyInput) =>
		api.put<{ key: CacheKeyValue }>(
			`/admin/cache/${database}/key?name=${encodeURIComponent(name)}`,
			input
		),

	remove: (database: CacheDatabaseName, name: string) =>
		api.delete<void>(`/admin/cache/${database}/key?name=${encodeURIComponent(name)}`),

	clearGroup: (database: CacheDatabaseName, group: string) =>
		api.post<void>(`/admin/cache/${database}/groups/${encodeURIComponent(group)}/clear`),

	flush: (database: CacheDatabaseName) => api.delete<void>(`/admin/cache/${database}`)
};

function cacheQuery(query: CacheKeyQuery): string {
	const params = new URLSearchParams();
	for (const [name, value] of Object.entries(query)) {
		if (value !== undefined && value !== '') params.set(name, String(value));
	}

	const text = params.toString();
	return text ? `?${text}` : '';
}

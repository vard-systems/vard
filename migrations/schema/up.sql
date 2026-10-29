-- The whole schema, as the first migration makes it. Written once and never
-- changed: a later change is a migration of its own beside 00001_schema.go,
-- and TestLiveTheMigrationsMakeWhatTheModelsDescribe holds the two together.

-- Tables

CREATE TABLE admin_role_assignments (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    admin_id uuid NOT NULL,
    role_id uuid NOT NULL,
    application_id uuid
);

CREATE TABLE admin_roles (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    name character varying(64) NOT NULL,
    description character varying(255) DEFAULT ''::character varying NOT NULL,
    permissions text
);

CREATE TABLE admin_security (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    require_mfa boolean NOT NULL,
    idle_timeout_minutes bigint DEFAULT 30 NOT NULL
);

CREATE TABLE admin_sessions (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    admin_id uuid NOT NULL,
    token_hash character varying(64) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    revoked_at timestamp with time zone,
    is_mfa_passed boolean DEFAULT false NOT NULL,
    passkey_ceremony text DEFAULT ''::text NOT NULL,
    last_seen_at timestamp with time zone NOT NULL,
    user_agent character varying(255) DEFAULT ''::character varying NOT NULL,
    ip character varying(45) DEFAULT ''::character varying NOT NULL
);

CREATE TABLE admins (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    username character varying(100) NOT NULL,
    email character varying(255) NOT NULL,
    first_name character varying(100) NOT NULL,
    last_name character varying(100) NOT NULL,
    avatar_url character varying(512) DEFAULT ''::character varying NOT NULL,
    password_hash character varying(255) NOT NULL,
    status character varying(32) DEFAULT 'invited'::character varying NOT NULL,
    email_verified_at timestamp with time zone,
    last_login_at timestamp with time zone,
    last_login_ip character varying(45) DEFAULT ''::character varying NOT NULL
);

CREATE TABLE api_scopes (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    api_id uuid NOT NULL,
    name character varying(128) NOT NULL,
    description character varying(255) DEFAULT ''::character varying NOT NULL,
    is_default boolean NOT NULL
);

CREATE TABLE apis (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    name character varying(100) NOT NULL,
    identifier character varying(255) NOT NULL,
    description character varying(255) DEFAULT ''::character varying NOT NULL,
    enforce_roles boolean NOT NULL,
    signing_algorithm character varying(16) NOT NULL,
    token_lifetime bigint NOT NULL,
    allow_offline_access boolean NOT NULL,
    system character varying(16) DEFAULT ''::character varying NOT NULL
);

CREATE TABLE application_api_scopes (
    application_id uuid NOT NULL,
    api_scope_id uuid NOT NULL
);

CREATE TABLE application_apis (
    application_id uuid NOT NULL,
    api_id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL
);

CREATE TABLE applications (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    name character varying(100) NOT NULL,
    description character varying(255) DEFAULT ''::character varying NOT NULL,
    type character varying(16) NOT NULL,
    logo_url character varying(512) DEFAULT ''::character varying NOT NULL,
    website_url character varying(512) DEFAULT ''::character varying NOT NULL,
    privacy_url character varying(512) DEFAULT ''::character varying NOT NULL,
    terms_url character varying(512) DEFAULT ''::character varying NOT NULL,
    allow_registration boolean NOT NULL,
    client_id character varying(64) NOT NULL,
    client_secret_hash character varying(64) DEFAULT ''::character varying NOT NULL,
    secret_hint character varying(8) DEFAULT ''::character varying NOT NULL,
    secret_created_at timestamp with time zone,
    token_auth_method character varying(32) NOT NULL,
    grant_types text,
    redirect_uris text,
    post_logout_redirect_uris text,
    allow_development_redirects boolean DEFAULT false NOT NULL,
    scopes text,
    backchannel_logout_uri character varying(512) DEFAULT ''::character varying NOT NULL,
    frontchannel_logout_uri character varying(512) DEFAULT ''::character varying NOT NULL,
    require_pkce boolean NOT NULL,
    require_pushed_requests boolean NOT NULL,
    access_token_lifetime bigint NOT NULL,
    id_token_lifetime bigint NOT NULL,
    refresh_token_lifetime bigint NOT NULL,
    assert_roles boolean NOT NULL,
    require_role_assignment boolean DEFAULT false NOT NULL,
    login_flow_id uuid,
    is_enabled boolean NOT NULL
);

CREATE TABLE audit_logs (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    admin_id uuid,
    actor_email character varying(255) DEFAULT ''::character varying NOT NULL,
    action character varying(128) NOT NULL,
    target_type character varying(64) DEFAULT ''::character varying NOT NULL,
    target_id character varying(64) DEFAULT ''::character varying NOT NULL,
    metadata text,
    ip character varying(45) DEFAULT ''::character varying NOT NULL,
    user_agent character varying(255) DEFAULT ''::character varying NOT NULL,
    request_id character varying(64) DEFAULT ''::character varying NOT NULL
);

CREATE TABLE authorization_codes (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    code_hash character varying(64) NOT NULL,
    application_id uuid NOT NULL,
    user_id uuid NOT NULL,
    session_id uuid,
    redirect_uri character varying(512) NOT NULL,
    scope text NOT NULL,
    nonce text DEFAULT ''::text NOT NULL,
    code_challenge character varying(128) DEFAULT ''::character varying NOT NULL,
    code_challenge_method character varying(8) DEFAULT ''::character varying NOT NULL,
    audience character varying(255) DEFAULT ''::character varying NOT NULL,
    authenticated_at timestamp with time zone NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    used_at timestamp with time zone
);

CREATE TABLE authorization_requests (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    handle_hash character varying(64) NOT NULL,
    application_id uuid NOT NULL,
    redirect_uri character varying(512) NOT NULL,
    scope text NOT NULL,
    state text DEFAULT ''::text NOT NULL,
    nonce text DEFAULT ''::text NOT NULL,
    code_challenge character varying(128) DEFAULT ''::character varying NOT NULL,
    code_challenge_method character varying(8) DEFAULT ''::character varying NOT NULL,
    audience character varying(255) DEFAULT ''::character varying NOT NULL,
    login_hint character varying(255) DEFAULT ''::character varying NOT NULL,
    force_consent boolean NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    completed_at timestamp with time zone
);

CREATE TABLE consents (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    user_id uuid NOT NULL,
    application_id uuid NOT NULL,
    scopes text NOT NULL
);

CREATE TABLE device_authorizations (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    device_code_hash character varying(64) NOT NULL,
    user_code_hash character varying(64) NOT NULL,
    application_id uuid NOT NULL,
    scope text NOT NULL,
    audience character varying(255) DEFAULT ''::character varying NOT NULL,
    status character varying(16) NOT NULL,
    "interval" bigint NOT NULL,
    user_id uuid,
    session_id uuid,
    authenticated_at timestamp with time zone,
    last_polled_at timestamp with time zone,
    expires_at timestamp with time zone NOT NULL,
    used_at timestamp with time zone
);

CREATE TABLE dpop_proofs (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    hash character varying(64) NOT NULL,
    expires_at timestamp with time zone NOT NULL
);

CREATE TABLE email_verifications (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    token_hash character varying(64) NOT NULL,
    user_id uuid NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    used_at timestamp with time zone,
    new_email character varying(255) DEFAULT ''::character varying NOT NULL
);

CREATE TABLE frontchannel_logouts (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    handle_hash character varying(64) NOT NULL,
    frames text NOT NULL,
    next character varying(2048) NOT NULL,
    language character varying(35) DEFAULT ''::character varying NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    used_at timestamp with time zone
);

CREATE TABLE languages (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    code character varying(16) NOT NULL,
    name character varying(64) NOT NULL,
    native_name character varying(64) NOT NULL,
    is_enabled boolean NOT NULL,
    is_default boolean NOT NULL,
    "position" bigint NOT NULL
);

CREATE TABLE login_codes (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    handle_hash character varying(64) NOT NULL,
    code_hash character varying(64) NOT NULL,
    user_id uuid NOT NULL,
    request character varying(64) DEFAULT ''::character varying NOT NULL,
    remember_me boolean NOT NULL,
    attempts bigint NOT NULL,
    sent_at timestamp with time zone NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    used_at timestamp with time zone
);

CREATE TABLE login_flows (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    name character varying(100) NOT NULL,
    slug character varying(64) NOT NULL,
    description character varying(500) DEFAULT ''::character varying NOT NULL,
    is_default boolean NOT NULL,
    is_enabled boolean NOT NULL,
    steps text NOT NULL,
    allow_sign_in boolean NOT NULL,
    allow_registration boolean NOT NULL,
    allow_password_reset boolean NOT NULL,
    allow_remember_me boolean NOT NULL,
    verify_email_on_register boolean NOT NULL,
    require_verified_email boolean NOT NULL,
    allow_email_change boolean NOT NULL,
    session_lifetime_hours bigint NOT NULL
);

CREATE TABLE mail_settings (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    is_enabled boolean NOT NULL,
    host character varying(255) DEFAULT ''::character varying NOT NULL,
    port bigint NOT NULL,
    encryption character varying(16) NOT NULL,
    username character varying(255) DEFAULT ''::character varying NOT NULL,
    password bytea,
    from_address character varying(255) DEFAULT ''::character varying NOT NULL,
    from_name character varying(100) DEFAULT ''::character varying NOT NULL
);

CREATE TABLE mfa_factors (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    admin_id uuid NOT NULL,
    method character varying(32) NOT NULL,
    label character varying(100) DEFAULT ''::character varying NOT NULL,
    secret text NOT NULL,
    credential_id character varying(255) DEFAULT ''::character varying NOT NULL,
    recovery_codes text,
    confirmed_at timestamp with time zone,
    last_used_at timestamp with time zone
);

CREATE TABLE organization_logos (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    content_type character varying(32) DEFAULT ''::character varying NOT NULL,
    data bytea NOT NULL,
    version character varying(16) DEFAULT ''::character varying NOT NULL
);

CREATE TABLE organizations (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    name character varying(100) NOT NULL,
    slug character varying(64) NOT NULL,
    logo_version character varying(16) DEFAULT ''::character varying NOT NULL,
    support_email character varying(255) DEFAULT ''::character varying NOT NULL,
    support_phone character varying(32) DEFAULT ''::character varying NOT NULL,
    terms_url character varying(512) DEFAULT ''::character varying NOT NULL,
    privacy_url character varying(512) DEFAULT ''::character varying NOT NULL,
    timezone character varying(64) DEFAULT 'UTC'::character varying NOT NULL,
    agreements_updated_at timestamp with time zone
);

CREATE TABLE otp_settings (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    code_length bigint NOT NULL,
    lifetime_minutes bigint NOT NULL,
    max_attempts bigint NOT NULL,
    resend_seconds bigint NOT NULL
);

CREATE TABLE passkey_logins (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    handle_hash character varying(64) NOT NULL,
    request character varying(64) DEFAULT ''::character varying NOT NULL,
    remember_me boolean NOT NULL,
    ceremony text NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    used_at timestamp with time zone
);

CREATE TABLE password_resets (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    token_hash character varying(64) NOT NULL,
    user_id uuid NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    used_at timestamp with time zone
);

CREATE TABLE pending_sign_ins (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    handle_hash character varying(64) NOT NULL,
    user_id uuid NOT NULL,
    request character varying(64) DEFAULT ''::character varying NOT NULL,
    remember_me boolean NOT NULL,
    passkey text DEFAULT ''::text NOT NULL,
    link_connection_id uuid,
    link_subject character varying(512) DEFAULT ''::character varying NOT NULL,
    link_email character varying(255) DEFAULT ''::character varying NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    used_at timestamp with time zone
);

CREATE TABLE pushed_requests (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    handle_hash character varying(64) NOT NULL,
    application_id uuid NOT NULL,
    params text NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    used_at timestamp with time zone
);

CREATE TABLE refresh_tokens (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    token_hash character varying(64) NOT NULL,
    family_id uuid NOT NULL,
    application_id uuid NOT NULL,
    user_id uuid NOT NULL,
    code_id uuid,
    session_id uuid,
    dpop_key character varying(64) DEFAULT ''::character varying NOT NULL,
    scope text NOT NULL,
    audience character varying(255) DEFAULT ''::character varying NOT NULL,
    authenticated_at timestamp with time zone NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    revoked_at timestamp with time zone,
    successor_id uuid,
    successor_sealed bytea
);

CREATE TABLE saml_assertions (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    hash character varying(64) NOT NULL,
    expires_at timestamp with time zone NOT NULL
);

CREATE TABLE server_secrets (
    name character varying(32) NOT NULL,
    value bytea NOT NULL,
    created_at timestamp with time zone NOT NULL
);

CREATE TABLE session_applications (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    session_id uuid NOT NULL,
    application_id uuid NOT NULL
);

CREATE TABLE setup_tokens (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    token_hash character varying(64) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    used_at timestamp with time zone
);

CREATE TABLE sign_in_throttles (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    key character varying(160) NOT NULL,
    failures bigint NOT NULL,
    blocked_until timestamp with time zone,
    expires_at timestamp with time zone NOT NULL
);

CREATE TABLE signing_keys (
    kid character varying(64) NOT NULL,
    algorithm character varying(16) NOT NULL,
    private_key bytea NOT NULL,
    created_at timestamp with time zone NOT NULL,
    retired_at timestamp with time zone
);

CREATE TABLE social_identities (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    user_id uuid NOT NULL,
    provider_id uuid NOT NULL,
    subject character varying(255) NOT NULL,
    email character varying(255) DEFAULT ''::character varying NOT NULL,
    last_login_at timestamp with time zone
);

CREATE TABLE social_logins (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    state_hash character varying(64) NOT NULL,
    provider_id uuid NOT NULL,
    verifier character varying(128) DEFAULT ''::character varying NOT NULL,
    request character varying(64) DEFAULT ''::character varying NOT NULL,
    next character varying(512) DEFAULT ''::character varying NOT NULL,
    expires_at timestamp with time zone NOT NULL
);

CREATE TABLE social_providers (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    kind character varying(32) NOT NULL,
    slug character varying(64) NOT NULL,
    name character varying(100) NOT NULL,
    client_id character varying(255) NOT NULL,
    client_secret bytea,
    scopes text,
    is_enabled boolean NOT NULL,
    "position" bigint NOT NULL,
    link_verified_emails boolean NOT NULL,
    allow_registration boolean NOT NULL,
    authorize_url character varying(512) DEFAULT ''::character varying NOT NULL,
    token_url character varying(512) DEFAULT ''::character varying NOT NULL,
    userinfo_url character varying(512) DEFAULT ''::character varying NOT NULL,
    token_auth character varying(32) DEFAULT ''::character varying NOT NULL,
    team_id character varying(64) DEFAULT ''::character varying NOT NULL,
    key_id character varying(64) DEFAULT ''::character varying NOT NULL,
    private_key bytea
);

CREATE TABLE sso_connections (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    slug character varying(64) NOT NULL,
    name character varying(100) NOT NULL,
    protocol character varying(8) NOT NULL,
    is_enabled boolean NOT NULL,
    domains jsonb NOT NULL,
    enforce_domains boolean NOT NULL,
    show_on_login boolean NOT NULL,
    issuer character varying(512) DEFAULT ''::character varying NOT NULL,
    client_id character varying(255) DEFAULT ''::character varying NOT NULL,
    client_secret bytea,
    scopes text,
    metadata_url character varying(1024) DEFAULT ''::character varying NOT NULL,
    metadata text DEFAULT ''::text NOT NULL,
    name_id_format character varying(16) DEFAULT ''::character varying NOT NULL,
    sign_requests boolean NOT NULL,
    sp_key bytea,
    sp_certificate text DEFAULT ''::text NOT NULL,
    matching character varying(16) NOT NULL,
    create_users boolean NOT NULL,
    sync_profile boolean NOT NULL,
    email_attribute character varying(255) DEFAULT ''::character varying NOT NULL,
    first_name_attribute character varying(255) DEFAULT ''::character varying NOT NULL,
    last_name_attribute character varying(255) DEFAULT ''::character varying NOT NULL,
    groups_attribute character varying(255) DEFAULT ''::character varying NOT NULL,
    role_mappings jsonb NOT NULL,
    sync_roles boolean NOT NULL
);

CREATE TABLE sso_domain_verifications (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    connection_id uuid NOT NULL,
    domain character varying(253) NOT NULL,
    token character varying(64) NOT NULL,
    method character varying(16) DEFAULT ''::character varying NOT NULL,
    verified_at timestamp with time zone,
    checked_at timestamp with time zone
);

CREATE TABLE sso_identities (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    user_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    subject character varying(512) NOT NULL,
    email character varying(255) DEFAULT ''::character varying NOT NULL,
    last_login_at timestamp with time zone
);

CREATE TABLE sso_logins (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    state_hash character varying(64) NOT NULL,
    connection_id uuid NOT NULL,
    verifier character varying(128) DEFAULT ''::character varying NOT NULL,
    nonce character varying(128) DEFAULT ''::character varying NOT NULL,
    saml_request_id character varying(128) DEFAULT ''::character varying NOT NULL,
    relay_hash character varying(64) DEFAULT ''::character varying NOT NULL,
    saml_response text DEFAULT ''::text NOT NULL,
    request character varying(64) DEFAULT ''::character varying NOT NULL,
    next character varying(512) DEFAULT ''::character varying NOT NULL,
    expires_at timestamp with time zone NOT NULL
);

CREATE TABLE token_revocations (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    key character varying(160) NOT NULL,
    revoked_at timestamp with time zone NOT NULL,
    expires_at timestamp with time zone NOT NULL
);

CREATE TABLE translations (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    language_id uuid NOT NULL,
    app character varying(16) NOT NULL,
    messages jsonb NOT NULL
);

CREATE TABLE user_factors (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    user_id uuid NOT NULL,
    method character varying(32) NOT NULL,
    label character varying(100) DEFAULT ''::character varying NOT NULL,
    secret text NOT NULL,
    credential_id character varying(255) DEFAULT ''::character varying NOT NULL,
    confirmed_at timestamp with time zone,
    last_used_at timestamp with time zone
);

CREATE TABLE user_fields (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    name character varying(64) NOT NULL,
    label character varying(100) NOT NULL,
    type character varying(16) NOT NULL,
    is_required boolean DEFAULT false NOT NULL,
    is_unique boolean DEFAULT false NOT NULL,
    min numeric,
    max numeric,
    starts_with character varying(64) DEFAULT ''::character varying NOT NULL,
    "position" bigint DEFAULT 0 NOT NULL
);

CREATE TABLE user_recovery_codes (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    user_id uuid NOT NULL,
    code_hash character varying(64) NOT NULL,
    used_at timestamp with time zone
);

CREATE TABLE user_role_api_scopes (
    user_role_id uuid NOT NULL,
    api_scope_id uuid NOT NULL
);

CREATE TABLE user_role_inherits (
    role_id uuid NOT NULL,
    inherited_role_id uuid NOT NULL
);

CREATE TABLE user_role_members (
    user_id uuid NOT NULL,
    user_role_id uuid NOT NULL
);

CREATE TABLE user_roles (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    application_id uuid,
    name character varying(64) NOT NULL,
    description character varying(255) DEFAULT ''::character varying NOT NULL,
    is_default boolean DEFAULT false NOT NULL
);

CREATE TABLE user_sessions (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    token_hash character varying(64) NOT NULL,
    user_id uuid NOT NULL,
    authenticated_at timestamp with time zone NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    revoked_at timestamp with time zone,
    ip character varying(45) DEFAULT ''::character varying NOT NULL,
    user_agent character varying(255) DEFAULT ''::character varying NOT NULL
);

CREATE TABLE users (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    email character varying(255) NOT NULL,
    is_email_verified boolean DEFAULT false NOT NULL,
    first_name character varying(100) DEFAULT ''::character varying NOT NULL,
    last_name character varying(100) DEFAULT ''::character varying NOT NULL,
    is_active boolean NOT NULL,
    password_hash character varying(255) DEFAULT ''::character varying NOT NULL,
    is_password_temporary boolean DEFAULT false NOT NULL,
    last_login_at timestamp with time zone,
    agreements_accepted_at timestamp with time zone,
    data text
);

-- Primary keys and unique constraints

ALTER TABLE admin_role_assignments
    ADD CONSTRAINT admin_role_assignments_pkey PRIMARY KEY (id);

ALTER TABLE admin_roles
    ADD CONSTRAINT admin_roles_pkey PRIMARY KEY (id);

ALTER TABLE admin_security
    ADD CONSTRAINT admin_security_pkey PRIMARY KEY (id);

ALTER TABLE admin_sessions
    ADD CONSTRAINT admin_sessions_pkey PRIMARY KEY (id);

ALTER TABLE admins
    ADD CONSTRAINT admins_pkey PRIMARY KEY (id);

ALTER TABLE api_scopes
    ADD CONSTRAINT api_scopes_pkey PRIMARY KEY (id);

ALTER TABLE apis
    ADD CONSTRAINT apis_pkey PRIMARY KEY (id);

ALTER TABLE application_api_scopes
    ADD CONSTRAINT application_api_scopes_pkey PRIMARY KEY (application_id, api_scope_id);

ALTER TABLE application_apis
    ADD CONSTRAINT application_apis_pkey PRIMARY KEY (application_id, api_id);

ALTER TABLE applications
    ADD CONSTRAINT applications_pkey PRIMARY KEY (id);

ALTER TABLE audit_logs
    ADD CONSTRAINT audit_logs_pkey PRIMARY KEY (id);

ALTER TABLE authorization_codes
    ADD CONSTRAINT authorization_codes_pkey PRIMARY KEY (id);

ALTER TABLE authorization_requests
    ADD CONSTRAINT authorization_requests_pkey PRIMARY KEY (id);

ALTER TABLE consents
    ADD CONSTRAINT consents_pkey PRIMARY KEY (id);

ALTER TABLE device_authorizations
    ADD CONSTRAINT device_authorizations_pkey PRIMARY KEY (id);

ALTER TABLE dpop_proofs
    ADD CONSTRAINT dpop_proofs_pkey PRIMARY KEY (id);

ALTER TABLE email_verifications
    ADD CONSTRAINT email_verifications_pkey PRIMARY KEY (id);

ALTER TABLE frontchannel_logouts
    ADD CONSTRAINT frontchannel_logouts_pkey PRIMARY KEY (id);

ALTER TABLE languages
    ADD CONSTRAINT languages_pkey PRIMARY KEY (id);

ALTER TABLE login_codes
    ADD CONSTRAINT login_codes_pkey PRIMARY KEY (id);

ALTER TABLE login_flows
    ADD CONSTRAINT login_flows_pkey PRIMARY KEY (id);

ALTER TABLE mail_settings
    ADD CONSTRAINT mail_settings_pkey PRIMARY KEY (id);

ALTER TABLE mfa_factors
    ADD CONSTRAINT mfa_factors_pkey PRIMARY KEY (id);

ALTER TABLE organization_logos
    ADD CONSTRAINT organization_logos_pkey PRIMARY KEY (id);

ALTER TABLE organizations
    ADD CONSTRAINT organizations_pkey PRIMARY KEY (id);

ALTER TABLE otp_settings
    ADD CONSTRAINT otp_settings_pkey PRIMARY KEY (id);

ALTER TABLE passkey_logins
    ADD CONSTRAINT passkey_logins_pkey PRIMARY KEY (id);

ALTER TABLE password_resets
    ADD CONSTRAINT password_resets_pkey PRIMARY KEY (id);

ALTER TABLE pending_sign_ins
    ADD CONSTRAINT pending_sign_ins_pkey PRIMARY KEY (id);

ALTER TABLE pushed_requests
    ADD CONSTRAINT pushed_requests_pkey PRIMARY KEY (id);

ALTER TABLE refresh_tokens
    ADD CONSTRAINT refresh_tokens_pkey PRIMARY KEY (id);

ALTER TABLE saml_assertions
    ADD CONSTRAINT saml_assertions_pkey PRIMARY KEY (id);

ALTER TABLE server_secrets
    ADD CONSTRAINT server_secrets_pkey PRIMARY KEY (name);

ALTER TABLE session_applications
    ADD CONSTRAINT session_applications_pkey PRIMARY KEY (id);

ALTER TABLE setup_tokens
    ADD CONSTRAINT setup_tokens_pkey PRIMARY KEY (id);

ALTER TABLE sign_in_throttles
    ADD CONSTRAINT sign_in_throttles_pkey PRIMARY KEY (id);

ALTER TABLE signing_keys
    ADD CONSTRAINT signing_keys_pkey PRIMARY KEY (kid);

ALTER TABLE social_identities
    ADD CONSTRAINT social_identities_pkey PRIMARY KEY (id);

ALTER TABLE social_logins
    ADD CONSTRAINT social_logins_pkey PRIMARY KEY (id);

ALTER TABLE social_providers
    ADD CONSTRAINT social_providers_pkey PRIMARY KEY (id);

ALTER TABLE sso_connections
    ADD CONSTRAINT sso_connections_pkey PRIMARY KEY (id);

ALTER TABLE sso_domain_verifications
    ADD CONSTRAINT sso_domain_verifications_pkey PRIMARY KEY (id);

ALTER TABLE sso_identities
    ADD CONSTRAINT sso_identities_pkey PRIMARY KEY (id);

ALTER TABLE sso_logins
    ADD CONSTRAINT sso_logins_pkey PRIMARY KEY (id);

ALTER TABLE token_revocations
    ADD CONSTRAINT token_revocations_pkey PRIMARY KEY (id);

ALTER TABLE translations
    ADD CONSTRAINT translations_pkey PRIMARY KEY (id);

ALTER TABLE user_factors
    ADD CONSTRAINT user_factors_pkey PRIMARY KEY (id);

ALTER TABLE user_fields
    ADD CONSTRAINT user_fields_pkey PRIMARY KEY (id);

ALTER TABLE user_recovery_codes
    ADD CONSTRAINT user_recovery_codes_pkey PRIMARY KEY (id);

ALTER TABLE user_role_api_scopes
    ADD CONSTRAINT user_role_api_scopes_pkey PRIMARY KEY (user_role_id, api_scope_id);

ALTER TABLE user_role_inherits
    ADD CONSTRAINT user_role_inherits_pkey PRIMARY KEY (role_id, inherited_role_id);

ALTER TABLE user_role_members
    ADD CONSTRAINT user_role_members_pkey PRIMARY KEY (user_id, user_role_id);

ALTER TABLE user_roles
    ADD CONSTRAINT user_roles_pkey PRIMARY KEY (id);

ALTER TABLE user_sessions
    ADD CONSTRAINT user_sessions_pkey PRIMARY KEY (id);

ALTER TABLE users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);

-- Indexes

CREATE INDEX idx_admin_role_assignments_admin_id ON admin_role_assignments USING btree (admin_id);

CREATE INDEX idx_admin_role_assignments_application_id ON admin_role_assignments USING btree (application_id);

CREATE UNIQUE INDEX idx_admin_role_assignments_global ON admin_role_assignments USING btree (admin_id, role_id) WHERE (application_id IS NULL);

CREATE INDEX idx_admin_role_assignments_role_id ON admin_role_assignments USING btree (role_id);

CREATE UNIQUE INDEX idx_admin_role_assignments_scoped ON admin_role_assignments USING btree (admin_id, role_id, application_id) WHERE (application_id IS NOT NULL);

CREATE UNIQUE INDEX idx_admin_roles_name ON admin_roles USING btree (name);

CREATE INDEX idx_admin_sessions_admin_id ON admin_sessions USING btree (admin_id);

CREATE INDEX idx_admin_sessions_expires_at ON admin_sessions USING btree (expires_at);

CREATE UNIQUE INDEX idx_admin_sessions_token_hash ON admin_sessions USING btree (token_hash);

CREATE UNIQUE INDEX idx_admins_email ON admins USING btree (email);

CREATE INDEX idx_admins_status ON admins USING btree (status);

CREATE UNIQUE INDEX idx_admins_username ON admins USING btree (username);

CREATE UNIQUE INDEX idx_api_scopes_api_name ON api_scopes USING btree (api_id, name);

CREATE UNIQUE INDEX idx_apis_identifier ON apis USING btree (identifier);

CREATE INDEX idx_application_api_scopes_scope ON application_api_scopes USING btree (api_scope_id);

CREATE INDEX idx_application_apis_api ON application_apis USING btree (api_id);

CREATE UNIQUE INDEX idx_applications_client_id ON applications USING btree (client_id);

CREATE INDEX idx_applications_is_enabled ON applications USING btree (is_enabled);

CREATE INDEX idx_applications_login_flow_id ON applications USING btree (login_flow_id);

CREATE INDEX idx_applications_type ON applications USING btree (type);

CREATE INDEX idx_audit_logs_action_created ON audit_logs USING btree (action, created_at DESC);

CREATE INDEX idx_audit_logs_actor_created ON audit_logs USING btree (actor_email, created_at DESC);

CREATE INDEX idx_audit_logs_admin_id ON audit_logs USING btree (admin_id);

-- Entries about an API name it in their metadata, not as their target.
CREATE INDEX idx_audit_logs_api ON audit_logs USING btree ((((metadata)::jsonb ->> 'api_id'::text))) WHERE (metadata ~~ '%"api_id"%'::text);

CREATE INDEX idx_audit_logs_created_id ON audit_logs USING btree (created_at DESC, id DESC);

CREATE INDEX idx_audit_logs_request_id ON audit_logs USING btree (request_id);

CREATE INDEX idx_audit_logs_target ON audit_logs USING btree (target_type, target_id, created_at DESC);

CREATE INDEX idx_audit_logs_target_id ON audit_logs USING btree (target_id);

CREATE INDEX idx_authorization_codes_application_id ON authorization_codes USING btree (application_id);

CREATE UNIQUE INDEX idx_authorization_codes_code_hash ON authorization_codes USING btree (code_hash);

CREATE INDEX idx_authorization_codes_expires_at ON authorization_codes USING btree (expires_at);

CREATE INDEX idx_authorization_codes_user_id ON authorization_codes USING btree (user_id);

CREATE INDEX idx_authorization_requests_application_id ON authorization_requests USING btree (application_id);

CREATE INDEX idx_authorization_requests_expires_at ON authorization_requests USING btree (expires_at);

CREATE UNIQUE INDEX idx_authorization_requests_handle_hash ON authorization_requests USING btree (handle_hash);

CREATE INDEX idx_consents_application_id ON consents USING btree (application_id);

CREATE UNIQUE INDEX idx_consents_pair ON consents USING btree (user_id, application_id);

CREATE INDEX idx_device_authorizations_application_id ON device_authorizations USING btree (application_id);

CREATE UNIQUE INDEX idx_device_authorizations_device_code_hash ON device_authorizations USING btree (device_code_hash);

CREATE INDEX idx_device_authorizations_expires_at ON device_authorizations USING btree (expires_at);

CREATE UNIQUE INDEX idx_device_authorizations_user_code_hash ON device_authorizations USING btree (user_code_hash);

CREATE INDEX idx_dpop_proofs_expires_at ON dpop_proofs USING btree (expires_at);

CREATE UNIQUE INDEX idx_dpop_proofs_hash ON dpop_proofs USING btree (hash);

CREATE INDEX idx_email_verifications_expires_at ON email_verifications USING btree (expires_at);

CREATE UNIQUE INDEX idx_email_verifications_token_hash ON email_verifications USING btree (token_hash);

CREATE INDEX idx_email_verifications_user_id ON email_verifications USING btree (user_id);

CREATE INDEX idx_frontchannel_logouts_expires_at ON frontchannel_logouts USING btree (expires_at);

CREATE UNIQUE INDEX idx_frontchannel_logouts_handle_hash ON frontchannel_logouts USING btree (handle_hash);

CREATE UNIQUE INDEX idx_languages_code ON languages USING btree (code);

CREATE INDEX idx_login_codes_expires_at ON login_codes USING btree (expires_at);

CREATE UNIQUE INDEX idx_login_codes_handle_hash ON login_codes USING btree (handle_hash);

CREATE INDEX idx_login_codes_user_id ON login_codes USING btree (user_id);

CREATE UNIQUE INDEX idx_login_flows_slug ON login_flows USING btree (slug);

CREATE INDEX idx_mfa_factors_admin_id ON mfa_factors USING btree (admin_id);

CREATE INDEX idx_mfa_factors_credential_id ON mfa_factors USING btree (credential_id);

CREATE INDEX idx_mfa_factors_method ON mfa_factors USING btree (method);

CREATE UNIQUE INDEX idx_organizations_slug ON organizations USING btree (slug);

CREATE INDEX idx_passkey_logins_expires_at ON passkey_logins USING btree (expires_at);

CREATE UNIQUE INDEX idx_passkey_logins_handle_hash ON passkey_logins USING btree (handle_hash);

CREATE INDEX idx_password_resets_expires_at ON password_resets USING btree (expires_at);

CREATE UNIQUE INDEX idx_password_resets_token_hash ON password_resets USING btree (token_hash);

CREATE INDEX idx_password_resets_user_id ON password_resets USING btree (user_id);

CREATE INDEX idx_pending_sign_ins_expires_at ON pending_sign_ins USING btree (expires_at);

CREATE UNIQUE INDEX idx_pending_sign_ins_handle_hash ON pending_sign_ins USING btree (handle_hash);

CREATE INDEX idx_pending_sign_ins_user_id ON pending_sign_ins USING btree (user_id);

CREATE INDEX idx_pushed_requests_application_id ON pushed_requests USING btree (application_id);

CREATE INDEX idx_pushed_requests_expires_at ON pushed_requests USING btree (expires_at);

CREATE UNIQUE INDEX idx_pushed_requests_handle_hash ON pushed_requests USING btree (handle_hash);

CREATE INDEX idx_refresh_tokens_application_id ON refresh_tokens USING btree (application_id);

CREATE INDEX idx_refresh_tokens_code_id ON refresh_tokens USING btree (code_id);

CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens USING btree (expires_at);

CREATE INDEX idx_refresh_tokens_family_id ON refresh_tokens USING btree (family_id);

-- Only the rows still holding a sealed successor, which the sweep clears.
CREATE INDEX idx_refresh_tokens_sealed ON refresh_tokens USING btree (revoked_at) WHERE (successor_sealed IS NOT NULL);

CREATE UNIQUE INDEX idx_refresh_tokens_token_hash ON refresh_tokens USING btree (token_hash);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens USING btree (user_id);

CREATE INDEX idx_saml_assertions_expires_at ON saml_assertions USING btree (expires_at);

CREATE UNIQUE INDEX idx_saml_assertions_hash ON saml_assertions USING btree (hash);

CREATE INDEX idx_session_applications_application_id ON session_applications USING btree (application_id);

CREATE UNIQUE INDEX idx_session_applications_pair ON session_applications USING btree (session_id, application_id);

CREATE INDEX idx_setup_tokens_expires_at ON setup_tokens USING btree (expires_at);

CREATE UNIQUE INDEX idx_setup_tokens_token_hash ON setup_tokens USING btree (token_hash);

CREATE INDEX idx_sign_in_throttles_expires_at ON sign_in_throttles USING btree (expires_at);

CREATE UNIQUE INDEX idx_sign_in_throttles_key ON sign_in_throttles USING btree (key);

CREATE INDEX idx_signing_keys_algorithm ON signing_keys USING btree (algorithm);

CREATE UNIQUE INDEX idx_social_identities_subject ON social_identities USING btree (provider_id, subject);

CREATE INDEX idx_social_identities_user_id ON social_identities USING btree (user_id);

CREATE INDEX idx_social_logins_expires_at ON social_logins USING btree (expires_at);

CREATE INDEX idx_social_logins_provider_id ON social_logins USING btree (provider_id);

CREATE UNIQUE INDEX idx_social_logins_state_hash ON social_logins USING btree (state_hash);

CREATE UNIQUE INDEX idx_social_providers_slug ON social_providers USING btree (slug);

CREATE UNIQUE INDEX idx_sso_connections_slug ON sso_connections USING btree (slug);

CREATE UNIQUE INDEX idx_sso_domain_verifications_domain ON sso_domain_verifications USING btree (connection_id, domain);

CREATE UNIQUE INDEX idx_sso_identities_subject ON sso_identities USING btree (connection_id, subject);

CREATE INDEX idx_sso_identities_user_id ON sso_identities USING btree (user_id);

CREATE INDEX idx_sso_logins_connection_id ON sso_logins USING btree (connection_id);

CREATE INDEX idx_sso_logins_expires_at ON sso_logins USING btree (expires_at);

CREATE INDEX idx_sso_logins_relay_hash ON sso_logins USING btree (relay_hash);

CREATE UNIQUE INDEX idx_sso_logins_state_hash ON sso_logins USING btree (state_hash);

CREATE INDEX idx_token_revocations_expires_at ON token_revocations USING btree (expires_at);

CREATE UNIQUE INDEX idx_token_revocations_key ON token_revocations USING btree (key);

CREATE UNIQUE INDEX idx_translations_language_app ON translations USING btree (language_id, app);

CREATE INDEX idx_user_factors_credential_id ON user_factors USING btree (credential_id);

CREATE INDEX idx_user_factors_user_id ON user_factors USING btree (user_id);

CREATE UNIQUE INDEX idx_user_fields_name ON user_fields USING btree (name);

CREATE UNIQUE INDEX idx_user_recovery_codes_code ON user_recovery_codes USING btree (user_id, code_hash);

CREATE INDEX idx_user_role_api_scopes_scope ON user_role_api_scopes USING btree (api_scope_id);

CREATE INDEX idx_user_role_inherits_inherited ON user_role_inherits USING btree (inherited_role_id);

-- The other side of each join table: their primary keys lead with the first
-- column only.
CREATE INDEX idx_user_role_members_role ON user_role_members USING btree (user_role_id);

CREATE UNIQUE INDEX idx_user_roles_application_name ON user_roles USING btree (application_id, name);

-- A null application means a global role (a user role) or the whole panel
-- (an admin role's assignment). Postgres treats every null as different, so a
-- unique index over a nullable column needs a partial one for the null case.
CREATE UNIQUE INDEX idx_user_roles_global_name ON user_roles USING btree (name) WHERE (application_id IS NULL);

CREATE INDEX idx_user_roles_is_default ON user_roles USING btree (is_default);

CREATE INDEX idx_user_sessions_expires_at ON user_sessions USING btree (expires_at);

-- The lists that page newest first, and the activity log's filters, read in
-- these orders.
CREATE INDEX idx_user_sessions_recent ON user_sessions USING btree (created_at DESC, id DESC);

CREATE UNIQUE INDEX idx_user_sessions_token_hash ON user_sessions USING btree (token_hash);

CREATE INDEX idx_user_sessions_user_id ON user_sessions USING btree (user_id);

CREATE INDEX idx_users_created ON users USING btree (created_at DESC);

CREATE UNIQUE INDEX idx_users_email ON users USING btree (email);

-- The users page searches addresses by prefix, which the unique index on
-- email, compared under the database's collation, cannot serve.
CREATE INDEX idx_users_email_prefix ON users USING btree (email text_pattern_ops);

-- Foreign keys

ALTER TABLE admin_role_assignments
    ADD CONSTRAINT fk_admin_role_assignments_application FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE;

ALTER TABLE admin_role_assignments
    ADD CONSTRAINT fk_admin_role_assignments_role FOREIGN KEY (role_id) REFERENCES admin_roles(id) ON DELETE CASCADE;

ALTER TABLE admin_role_assignments
    ADD CONSTRAINT fk_admins_assignments FOREIGN KEY (admin_id) REFERENCES admins(id) ON DELETE CASCADE;

ALTER TABLE mfa_factors
    ADD CONSTRAINT fk_admins_mfa FOREIGN KEY (admin_id) REFERENCES admins(id) ON DELETE CASCADE;

ALTER TABLE admin_sessions
    ADD CONSTRAINT fk_admins_sessions FOREIGN KEY (admin_id) REFERENCES admins(id) ON DELETE CASCADE;

ALTER TABLE api_scopes
    ADD CONSTRAINT fk_apis_scopes FOREIGN KEY (api_id) REFERENCES apis(id) ON DELETE CASCADE;

ALTER TABLE application_api_scopes
    ADD CONSTRAINT fk_application_api_scopes_api_scope FOREIGN KEY (api_scope_id) REFERENCES api_scopes(id) ON DELETE CASCADE;

ALTER TABLE application_api_scopes
    ADD CONSTRAINT fk_application_api_scopes_application FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE;

ALTER TABLE application_apis
    ADD CONSTRAINT fk_application_apis_api FOREIGN KEY (api_id) REFERENCES apis(id) ON DELETE CASCADE;

ALTER TABLE application_apis
    ADD CONSTRAINT fk_application_apis_application FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE;

ALTER TABLE authorization_codes
    ADD CONSTRAINT fk_authorization_codes_application FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE;

ALTER TABLE authorization_codes
    ADD CONSTRAINT fk_authorization_codes_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE authorization_requests
    ADD CONSTRAINT fk_authorization_requests_application FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE;

ALTER TABLE consents
    ADD CONSTRAINT fk_consents_application FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE;

ALTER TABLE consents
    ADD CONSTRAINT fk_consents_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE device_authorizations
    ADD CONSTRAINT fk_device_authorizations_application FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE;

ALTER TABLE email_verifications
    ADD CONSTRAINT fk_email_verifications_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE translations
    ADD CONSTRAINT fk_languages_translations FOREIGN KEY (language_id) REFERENCES languages(id) ON DELETE CASCADE;

ALTER TABLE login_codes
    ADD CONSTRAINT fk_login_codes_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE password_resets
    ADD CONSTRAINT fk_password_resets_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE pending_sign_ins
    ADD CONSTRAINT fk_pending_sign_ins_link_connection FOREIGN KEY (link_connection_id) REFERENCES sso_connections(id) ON DELETE CASCADE;

ALTER TABLE pending_sign_ins
    ADD CONSTRAINT fk_pending_sign_ins_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE pushed_requests
    ADD CONSTRAINT fk_pushed_requests_application FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE;

ALTER TABLE refresh_tokens
    ADD CONSTRAINT fk_refresh_tokens_application FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE;

ALTER TABLE refresh_tokens
    ADD CONSTRAINT fk_refresh_tokens_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE session_applications
    ADD CONSTRAINT fk_session_applications_application FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE;

ALTER TABLE session_applications
    ADD CONSTRAINT fk_session_applications_session FOREIGN KEY (session_id) REFERENCES user_sessions(id) ON DELETE CASCADE;

ALTER TABLE social_identities
    ADD CONSTRAINT fk_social_identities_provider FOREIGN KEY (provider_id) REFERENCES social_providers(id) ON DELETE CASCADE;

ALTER TABLE social_identities
    ADD CONSTRAINT fk_social_identities_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE sso_domain_verifications
    ADD CONSTRAINT fk_sso_domain_verifications_connection FOREIGN KEY (connection_id) REFERENCES sso_connections(id) ON DELETE CASCADE;

ALTER TABLE sso_identities
    ADD CONSTRAINT fk_sso_identities_connection FOREIGN KEY (connection_id) REFERENCES sso_connections(id) ON DELETE CASCADE;

ALTER TABLE sso_identities
    ADD CONSTRAINT fk_sso_identities_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE sso_logins
    ADD CONSTRAINT fk_sso_logins_connection FOREIGN KEY (connection_id) REFERENCES sso_connections(id) ON DELETE CASCADE;

ALTER TABLE user_factors
    ADD CONSTRAINT fk_user_factors_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE user_recovery_codes
    ADD CONSTRAINT fk_user_recovery_codes_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE user_role_api_scopes
    ADD CONSTRAINT fk_user_role_api_scopes_api_scope FOREIGN KEY (api_scope_id) REFERENCES api_scopes(id) ON DELETE CASCADE;

ALTER TABLE user_role_api_scopes
    ADD CONSTRAINT fk_user_role_api_scopes_user_role FOREIGN KEY (user_role_id) REFERENCES user_roles(id) ON DELETE CASCADE;

ALTER TABLE user_role_inherits
    ADD CONSTRAINT fk_user_role_inherits_inherits FOREIGN KEY (inherited_role_id) REFERENCES user_roles(id) ON DELETE CASCADE;

ALTER TABLE user_role_inherits
    ADD CONSTRAINT fk_user_role_inherits_user_role FOREIGN KEY (role_id) REFERENCES user_roles(id) ON DELETE CASCADE;

ALTER TABLE user_role_members
    ADD CONSTRAINT fk_user_role_members_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE user_role_members
    ADD CONSTRAINT fk_user_role_members_user_role FOREIGN KEY (user_role_id) REFERENCES user_roles(id) ON DELETE CASCADE;

ALTER TABLE user_roles
    ADD CONSTRAINT fk_user_roles_application FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE;

ALTER TABLE user_sessions
    ADD CONSTRAINT fk_user_sessions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

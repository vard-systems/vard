import type {
	Application,
	ApplicationInput,
	ApplicationType,
	GrantType,
	Scope
} from '#lib/api/index.js';
import { blank, lines, types } from './applications';

const MINUTE = 60;
const DAY = 24 * 60 * MINUTE;

/** The application dialog's fields. Lists and lifetimes are edited as text
    (one URI a line, minutes and days) and turned back into the API's shape
    by `toInput`. */
export class ApplicationForm {
	input = $state<ApplicationInput>(blank('web'));
	redirects = $state('');
	logoutRedirects = $state('');
	accessMinutes = $state('60');
	idMinutes = $state('60');
	refreshDays = $state('30');

	kind = $derived(types[this.input.type]);
	/** A machine-to-machine application has no user: nothing to redirect, no ID token. */
	machine = $derived(this.input.type === 'm2m');
	ready = $derived(this.input.name.trim() !== '');

	fill(from: Application | ApplicationInput | null): void {
		const input: ApplicationInput = from
			? { ...from, login_flow_id: from.login_flow_id ?? '' }
			: blank('web');

		this.input = {
			...input,
			grant_types: [...input.grant_types],
			scopes: [...input.scopes],
			redirect_uris: [...input.redirect_uris],
			post_logout_redirect_uris: [...input.post_logout_redirect_uris]
		};
		this.redirects = input.redirect_uris.join('\n');
		this.logoutRedirects = input.post_logout_redirect_uris.join('\n');
		this.accessMinutes = String(Math.round(input.access_token_lifetime / MINUTE));
		this.idMinutes = String(Math.round(input.id_token_lifetime / MINUTE));
		this.refreshDays = String(Math.round(input.refresh_token_lifetime / DAY));
	}

	/** Starts again from the new type's defaults, keeping how it is presented. */
	changeType(type: ApplicationType): void {
		const { name, description, logo_url, website_url, privacy_url, terms_url } = this.input;
		this.fill({ ...blank(type), name, description, logo_url, website_url, privacy_url, terms_url });
	}

	setGrant(grant: GrantType, on: boolean): void {
		this.input.grant_types = toggled(this.input.grant_types, grant, on);
	}

	setScope(scope: Scope, on: boolean): void {
		this.input.scopes = toggled(this.input.scopes, scope, on);
	}

	toInput(): ApplicationInput {
		return {
			...this.input,
			name: this.input.name.trim(),
			description: this.input.description.trim(),
			backchannel_logout_uri: this.input.backchannel_logout_uri.trim(),
			frontchannel_logout_uri: this.input.frontchannel_logout_uri.trim(),
			redirect_uris: lines(this.redirects),
			post_logout_redirect_uris: lines(this.logoutRedirects),
			access_token_lifetime: Number(this.accessMinutes) * MINUTE,
			id_token_lifetime: Number(this.idMinutes) * MINUTE,
			refresh_token_lifetime: Number(this.refreshDays) * DAY
		};
	}
}

function toggled<T extends string>(list: T[], value: T, on: boolean): T[] {
	if (on) return list.includes(value) ? list : [...list, value];
	return list.filter((item) => item !== value);
}

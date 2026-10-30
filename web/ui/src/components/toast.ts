import { createToaster } from '@ark-ui/svelte/toast';

/** What an error says, for a toast about it: an app's own reading (its API's sentence) once it has
 *  given one to `readErrorsWith`, the error's message until then. */
let messageOf = (err: unknown, fallback: string): string =>
	err instanceof Error && err.message ? err.message : fallback;

export function readErrorsWith(read: (err: unknown, fallback: string) => string) {
	messageOf = read;
}

export const toaster = createToaster({
	placement: 'bottom',
	offsets: '32px',
	gap: 10,
	max: 4,
	duration: 5000,
	pauseOnPageIdle: true
});

const ERROR_DURATION = 8000;

export const notify = {
	success(title: string, description?: string) {
		toaster.success({ title, description });
	},

	/** Something failed: the server's sentence when it sent one, the
	    fallback otherwise — never a stack trace or a status code. */
	error(err: unknown, fallback: string) {
		toaster.error({ title: messageOf(err, fallback), duration: ERROR_DURATION });
	},

	problem(message: string) {
		toaster.error({ title: message, duration: ERROR_DURATION });
	}
};

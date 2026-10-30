import { notify as shared } from 'ui';

/** The sign-in pages' toasts, drawn by the shared Toaster: a page passes the
 *  sentence it has already put in the reader's language. */
export const notify = {
	success: (title: string, description?: string) => shared.success(title, description),
	error: (title: string) => shared.problem(title)
};

import { ApiError } from './client';

export function messageOf(err: unknown, fallback = 'Something went wrong. Try again.'): string {
	if (!(err instanceof ApiError)) return fallback;

	return err.message || fallback;
}

import { describe, expect, it } from 'vitest';
import { datedName } from './download';

describe('datedName', () => {
	it('names the subject and the day, in UTC', () =>
		expect(datedName('roles', new Date('2026-10-24T23:30:00Z'))).toBe('roles-2026-10-24.json'));
});

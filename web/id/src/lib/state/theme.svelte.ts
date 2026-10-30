import { Theme } from 'ui';

import { COOKIES } from '#lib/brand.js';

/** The sign-in pages' theme, in their own cookie (static/theme.js reads it before the first paint). */
export const theme = new Theme(COOKIES.theme);

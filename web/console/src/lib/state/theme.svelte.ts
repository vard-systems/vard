import { Theme } from 'ui';

import { COOKIES } from '#lib/brand.js';

export type { ThemeName as Theme } from 'ui';

/** The panel's theme, in its own cookie (static/theme.js reads it before the first paint). */
export const theme = new Theme(COOKIES.theme);

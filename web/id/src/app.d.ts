declare global {
	namespace App {
		interface Locals {
			/** The page's language, decided by the root layout and written into <html lang>. */
			language?: string;
		}
	}
}

export {};

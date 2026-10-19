export type TokenName = string;

export type Tokens<Name extends string = TokenName> = Record<Name, string>;

export type FontFace = {
	family: string;
	style: 'normal' | 'italic';
	weights: number[];
	subsets: string[];
	file: (subset: string, weight: number) => string;
};

export type Theme<ColorName extends string = string> = {
	fonts: {
		baseUrl: string;
		subsets: Record<string, string>;
		faces: FontFace[];
	};

	scales: { prefix: string; tokens: Tokens }[];

	/** The window widths the layout changes at. CSS cannot read a variable
	    in a media query, so these are for TypeScript (a MediaQuery) and for
	    reading; a stylesheet writes the number and names the breakpoint in a
	    comment. */
	breakpoints: Record<string, string>;

	responsive: { minWidth: string; tokens: Partial<Tokens> }[];

	colors: {
		light: Tokens<ColorName>;
		dark: Partial<Tokens<ColorName>>;
	};

	layers: {
		swapped: Record<ColorName, string> | Partial<Record<ColorName, string>>;
		sets: { prefix: string; selectors: string[] }[];
	};
};

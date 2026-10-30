import { messageOf } from '#lib/api/index.js';
import { readErrorsWith } from 'ui';

// The shared design (web/ui), with the panel's own code views beside it.
export * from 'ui';
export { default as CodeBlock } from './CodeBlock.svelte';
export { default as CodeEditor } from './CodeEditor.svelte';

// A toast about a failed call says what the API said.
readErrorsWith(messageOf);

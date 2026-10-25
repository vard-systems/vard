import { defineEnvVars } from '@sveltejs/kit/env';

// API_URL is where the provider and account API really listens, which the browser never sees:
// this server passes the API's paths on to it. Development runs it on this
// machine.
//
// CLIENT_CERT_HEADER is the header the TLS proxy in front names a client's certificate in, on
// its mutual TLS host (deploy/nginx). Empty — `make dev`, `make docker-local`, anything with no
// such proxy — no certificate is passed on, so no caller can claim one.
export const variables = defineEnvVars({
	API_URL: { schema: (input) => input || 'http://localhost:8080' },
	CLIENT_CERT_HEADER: { schema: (input) => (input ?? '').trim().toLowerCase() }
});

# Mutual TLS certificate authorities

Put the PEM file of the certificate authorities that `tls_client_auth`
clients' certificates chain to here, intermediates included, and name it in
`deploy/.env` as the API sees it:

```
MTLS_URL=https://mtls.id.example.com
MTLS_CA_FILE=/etc/vard/mtls/clients-ca.pem
```

`deploy/compose.yaml` mounts this directory read-only into the API. Only
public certificates belong here; nothing in it is secret. See
`deploy/README.md`, "Mutual TLS".

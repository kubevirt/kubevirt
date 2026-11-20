# perfscale-audit

## Prometheus TLS verification

For an HTTPS `prometheusURL`, the audit client verifies the server's certificate
chain and hostname using the system's trusted certificates. Omit
`prometheusVerifyTLS` or set it to `true` to use this default.
Setting `"prometheusVerifyTLS": false` explicitly disables verification.

Earlier versions interpreted explicit boolean values in reverse. If an existing
configuration sets `prometheusVerifyTLS` to `false`, remove the setting or set
it to `true` to keep verifying the server.

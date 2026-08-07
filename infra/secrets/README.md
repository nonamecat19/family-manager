# Local secrets

Files here are **development only** and are gitignored (`*.pem`).

Generate the auth signing key:

```sh
openssl ecparam -name prime256v1 -genkey -noout -out infra/secrets/auth-signing-key.pem
chmod 0444 infra/secrets/auth-signing-key.pem
```

The `chmod` is needed because the service image runs as `nonroot` (uid 65532) while the file
is created by your user: a 0600 key is unreadable inside the container, and the service exits
with `permission denied` rather than starting with no key.

**In a real deployment do not do this.** Mount the secret owned by the service user with mode
0400, or inject it from the platform's secret store. A world-readable private key is
acceptable on a laptop and nowhere else.

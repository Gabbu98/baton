<!-- baton:context:start -->
## Baton Context Bridge
_Carried over: 2026-04-24 08:36_

[user]: Give me information about how revolut can be better set up briefly

[assistant]: Let me quickly check the existing docs and config to give you relevant advice.

[assistant]: Here's a concise breakdown of what's missing or could be improved for a solid Revolut setup:

**Security**
- `revolut-platform.merchant-api.secret-key: "secretKey"` is a plaintext placeholder — must come from a secrets manager or config server (Vault, GCP Secret Manager) in all non-local profiles.
- HTTP basic auth is wired (`quarkus.http.auth.basic: true`) but the user credentials block is commented out — webhook endpoint is currently unauthenticated.
- Revolut webhooks should also be verified using the **HMAC signature** header (`Revolut-Signature`) — no validation exists yet.

**Config gaps**
- `revolut-platform.merchant-api.base-url` is hardcoded to sandbox — you need a profile switch (`sandbox` / `prod`) so it doesn't get accidentally deployed to production pointing at the sandbox.
- No OAuth2 / API key rotation strategy — Revolut uses a long-lived secret key; consider injecting it via env var (`REVOLUT_SECRET_KEY`) rather than config file.

**Missing infrastructure**
- RabbitMQ channels (exchange, queue names, routing keys) are not declared — the dependency is there but inert.
- MongoDB is commented out — decide whether to persist order IDs/idempotency keys locally, which Revolut's API strongly encourages to avoid duplicate charges.

**Idempotency**
- Revolut's `/api/orders` endpoint requires an `merchant_order_ext_ref` for idempotency. Without persistent storage there's no way to deduplicate retried commands.

The highest-priority items are **secret key handling**, **webhook HMAC validation**, and **profile-based URL switching** — those are the ones that could cause security or prod/sandbox confusion soonest.
<!-- baton:context:end -->

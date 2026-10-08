# OpenRouter provider

genifer generates images through OpenRouter's dedicated images API
(`POST /api/v1/images`) and discovers models via `GET /api/v1/images/models`.

## Credentials

An OpenRouter API key is required. By default genifer resolves it from the
`OPENROUTER_API_KEY` environment variable:

```sh
export OPENROUTER_API_KEY="sk-or-..."
```

This matches the default `config.yaml`:

```yaml
openrouter:
  api_key: "{env:OPENROUTER_API_KEY}"
```

You can point `api_key` at any value directive (see [config.md](config.md)) —
for example, fetch it from a secrets manager at the moment it's needed:

```yaml
openrouter:
  api_key: "{cmd:op read op://vault/openrouter/key}"
```

The key is resolved **lazily** — only when a request is actually made — so a
command-backed key never runs on startup.

The key is sent as `Authorization: Bearer <key>`. An empty or rejected key
surfaces as an authentication error in the TUI.

## Account requirements

- Paid image generation requires more than $1 of account credit, otherwise the
  API returns 402 and genifer reports an "insufficient credit" error.
- Failed generations return 502 (and are not billed); genifer reports a
  retryable "generation failed" error.

## Attribution headers (optional)

OpenRouter accepts optional attribution headers. genifer can send
`HTTP-Referer` and `X-Title`; a title only counts when a referer is also set.
These are optional and off by default.

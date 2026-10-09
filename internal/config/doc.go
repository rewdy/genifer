// Package config loads user configuration (config.yaml), persists app-owned
// last-used state (state.json), caches pricing (pricing-cache.json), and
// resolves setting values that may be literals, environment-variable
// references, or external commands. It can also create a commented starter
// config.yaml when none exists (never overwriting an existing one) and resolve
// the user's text editor for the `genifer config` command.
//
// Providers are configured as a list of keyed, typed instances under the
// top-level `providers:` key. Each instance has a unique `key` (its identity in
// the UI and in persisted state), a `type` (which selects the implementation,
// e.g. "openrouter" or "a1111"), and the type-specific fields it needs
// (`api_key` for OpenRouter, `base_url` for a1111). Two instances may share a
// type while differing by key. A legacy single-provider file (a top-level
// `provider:` string plus a type-named block) is detected on load and reported
// as ErrLegacyConfig so callers can back it up and rebuild it via onboarding
// rather than loading it.
package config

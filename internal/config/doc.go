// Package config loads user configuration (config.yaml), writes the in-tool
// config view's edited settings back to it, persists app-owned last-used state
// (state.json), caches pricing (pricing-cache.json), records prompt history
// (history-cache.json), and resolves setting values that may be literals,
// environment-variable references, or external commands. It can also create a
// commented starter config.yaml when none exists (never overwriting an existing
// one) and resolve the user's text editor for the `genifer config` command.
package config

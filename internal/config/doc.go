// Package config loads user configuration (config.yaml), persists app-owned
// last-used state (state.json), caches pricing (pricing-cache.json), and
// resolves setting values that may be literals, environment-variable
// references, or external commands. It can also create a commented starter
// config.yaml when none exists (never overwriting an existing one) and resolve
// the user's text editor for the `genifer config` command.
package config

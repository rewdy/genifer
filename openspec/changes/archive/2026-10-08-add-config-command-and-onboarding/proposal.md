# Proposal

## Why

New users have no `config.yaml` and no obvious way to create or edit one: the app
never writes the file, so first-run users land in the TUI on defaults with only a
stderr note, and anyone wanting to change settings must hand-create the file from
the docs. We want a `genifer config` command to open (and bootstrap) the config,
and a guided first-run experience that captures the one thing generation actually
needs — how to supply the API key.

## What Changes

- Add a `genifer config` subcommand that opens `config.yaml` in the user's editor
  (`$VISUAL` → `$EDITOR` → OS default), creating a commented starter file first if
  none exists.
- Add a `genifer config path` subcommand that prints the resolved config path and
  exits (for scripting / manual editing), without launching an editor.
- Introduce a shared "write commented starter `config.yaml`" capability used by
  both the command and onboarding. This is the first time the app writes
  `config.yaml`; it only ever **creates** the file when absent and never
  overwrites an existing one, preserving the user-owned invariant.
- Add an in-TUI first-run onboarding flow: when no `config.yaml` exists, show a
  WELCOME banner and prompt the user to choose how to provide the OpenRouter API
  key — paste a literal (with a plaintext-storage warning), reference an
  environment variable (`{env:NAME}`), or run a command (`{cmd:...}`). The choice
  is written into the newly created `config.yaml` as the matching value directive.
- After onboarding writes the config, continue into the normal TUI session.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities

- `config`: add the `genifer config` and `genifer config path` commands and a
  create-starter-config-if-missing behavior, refining the existing "config file
  missing" and "user-owned, never overwrite" requirements.
- `tui`: add a first-run onboarding flow (welcome banner + API-key provider
  choice) that runs when no config exists, before the normal shell.

## Impact

- `main.go`: argument dispatch gains `config` (and `config path`) subcommands
  alongside the existing `version` handling; first-run detection feeds onboarding
  into `tui.Run`.
- `internal/config`: new starter-file writer and editor-resolution helper;
  `config.yaml` becomes app-creatable (create-only).
- `internal/tui`: new onboarding phase/model state ahead of the picker.
- Docs: `docs/config.md` and `README.md` updated to describe the command and
  first-run flow.
- No new third-party dependencies; editor/open resolution reuses existing
  `exec`/OS-default patterns.

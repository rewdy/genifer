# config Specification

## Purpose

Defines how genifer loads user configuration, persists last-used selections separately from hand-edited settings, and resolves setting values that may come from literals, environment variables, or external commands.

## Requirements

### Requirement: Configuration file location and loading

The system SHALL read configuration from a `config.yaml` file in the user's config directory (`~/.config/genifer/config.yaml` on Unix-like systems, resolved via the platform config dir). The system SHALL treat this file as user-owned: it MUST NOT overwrite or modify an existing `config.yaml` during normal operation. The system MAY create `config.yaml` only when it is absent — via the explicit `genifer config` command or the first-run onboarding flow — and when creating it SHALL write a commented starter file; it MUST NOT create or alter the file as a side effect of a normal session launch.

#### Scenario: Config file present

- **WHEN** genifer starts and `config.yaml` exists and is valid
- **THEN** its settings are loaded and used for the session

#### Scenario: Config file missing

- **WHEN** genifer starts and no `config.yaml` exists
- **THEN** genifer starts with built-in defaults and reports, in the TUI, that no config was found and where it is expected

#### Scenario: Config file malformed

- **WHEN** `config.yaml` exists but cannot be parsed
- **THEN** genifer surfaces a clear error identifying the file and does not silently fall back to defaults

#### Scenario: Existing config never overwritten on create

- **WHEN** a create path (the `config` command or onboarding) runs and `config.yaml` already exists
- **THEN** the existing file is left byte-for-byte unchanged and is used as-is

### Requirement: Starter config generation

The system SHALL be able to generate a commented starter `config.yaml` that documents the available settings and their defaults, written only when no config file exists at the resolved path. The starter file SHALL be valid YAML that loads without error, and SHALL use the recommended API-key default (`{env:OPENROUTER_API_KEY}`) unless a different API-key choice is supplied by the caller (e.g. onboarding).

#### Scenario: Starter written when absent

- **WHEN** a starter config is requested and no `config.yaml` exists
- **THEN** a commented `config.yaml` is written to the resolved config path, creating the config directory if needed, and the written file loads successfully as valid configuration

#### Scenario: Starter not written when present

- **WHEN** a starter config is requested and `config.yaml` already exists
- **THEN** no file is written and the existing file is preserved

### Requirement: Config command opens the config file

The system SHALL provide a `genifer config` command that resolves the user's editor in the order `$VISUAL`, then `$EDITOR`, then an OS-appropriate default, creates a commented starter `config.yaml` first if none exists, and opens the config file in that editor. The command SHALL exit after the editor exits and SHALL NOT launch the TUI.

#### Scenario: Open existing config

- **WHEN** the user runs `genifer config` and `config.yaml` exists
- **THEN** the resolved editor is launched on the existing config file and the TUI is not started

#### Scenario: Create then open on first use

- **WHEN** the user runs `genifer config` and no `config.yaml` exists
- **THEN** a commented starter `config.yaml` is created and the resolved editor is launched on it

#### Scenario: Editor resolution order

- **WHEN** `genifer config` resolves which editor to use
- **THEN** it uses `$VISUAL` if set, else `$EDITOR` if set, else an OS-appropriate default editor/open command

#### Scenario: No editor available

- **WHEN** no editor can be resolved and no OS default is available
- **THEN** genifer reports the resolved config path and an actionable error instead of opening an editor

### Requirement: Config path command

The system SHALL provide a `genifer config path` command that prints the resolved absolute path to `config.yaml` and exits, without creating the file, launching an editor, or starting the TUI.

#### Scenario: Print path

- **WHEN** the user runs `genifer config path`
- **THEN** genifer prints the absolute path to `config.yaml` and exits, whether or not the file exists, without modifying anything

### Requirement: State persistence separate from config

The system SHALL persist last-used selections (at minimum, the last selected model and the last selected aspect ratio) in a separate app-written file (`state.json` in the same config directory). The system SHALL write to `state.json` only, never to `config.yaml`. When persisting one selection, the system SHALL preserve the other previously persisted selections rather than overwriting them.

#### Scenario: Remember last model

- **WHEN** the user selects a model and generates, then restarts genifer
- **THEN** the previously selected model is pre-selected on next launch

#### Scenario: Remember last aspect ratio

- **WHEN** the user generates with an aspect ratio selected, then restarts genifer
- **THEN** the previously selected aspect ratio is the remembered last-used value and is preferred as the default when a model offers it

#### Scenario: Persisting one selection preserves the other

- **WHEN** the user changes the model after an aspect ratio has already been remembered
- **THEN** the remembered aspect ratio is retained in `state.json` and not cleared by persisting the new model

#### Scenario: State file missing or invalid

- **WHEN** `state.json` is absent or cannot be parsed
- **THEN** genifer starts without a remembered selection and continues normally, recreating the file on next write

#### Scenario: Remembered model no longer available

- **WHEN** the remembered model is not present in the current provider model list
- **THEN** genifer does not pre-select it, falls back to no selection (or a default), and does not error

### Requirement: Generic value resolution for literals, env vars, and commands

The system SHALL support resolving any string configuration value as one of: a literal string, an environment-variable reference of the form `{env:NAME}`, or a command reference of the form `{cmd:...}` whose trimmed standard output becomes the value. Resolution SHALL be lazy: a referenced command runs only when its value is actually needed.

#### Scenario: Literal value

- **WHEN** a config value contains no recognized directive
- **THEN** the value is used verbatim

#### Scenario: Environment variable reference

- **WHEN** a config value is `{env:OPENROUTER_API_KEY}` and that variable is set
- **THEN** the resolved value is the variable's contents

#### Scenario: Environment variable missing

- **WHEN** a config value references an environment variable that is not set
- **THEN** resolution fails with an error naming the missing variable, surfaced when the value is needed

#### Scenario: Command reference

- **WHEN** a config value is `{cmd:op read op://vault/openrouter/key}` and the command exits zero
- **THEN** the resolved value is the command's standard output with surrounding whitespace trimmed

#### Scenario: Command failure

- **WHEN** a referenced command exits non-zero or cannot be executed
- **THEN** resolution fails with an error that includes the command's failure detail, surfaced when the value is needed

#### Scenario: Lazy evaluation

- **WHEN** a command-backed value is configured but never required during a session
- **THEN** the command is not executed

#### Scenario: Single-level resolution

- **WHEN** a resolved value itself contains text that looks like a directive
- **THEN** that text is treated as a literal and is not resolved again

### Requirement: Output directory resolution

The system SHALL determine where generated images are written from an optional `output_dir` setting, resolved through the generic value resolver first, then interpreted as follows: unset uses a built-in default location; `.` or `pwd` means the current working directory; a relative path is resolved against the current working directory; an absolute path is used as-is.

#### Scenario: Default when unset

- **WHEN** `output_dir` is not set
- **THEN** images are written to the built-in default directory

#### Scenario: Current working directory

- **WHEN** `output_dir` is `.` or `pwd`
- **THEN** images are written to the directory genifer was launched from

#### Scenario: Relative path

- **WHEN** `output_dir` is a relative path
- **THEN** it is resolved against the current working directory

#### Scenario: Absolute path

- **WHEN** `output_dir` is an absolute path
- **THEN** that path is used unchanged

### Requirement: Pricing cache

The system SHALL cache per-model pricing in a separate app-written file (`pricing-cache.json` in the config directory), reused for a bounded freshness window (24 hours) to avoid refetching pricing on every launch. The cache SHALL be app-owned (never `config.yaml`) and SHALL degrade gracefully: a missing, unparsable, or wrong-version cache is ignored and refetched rather than causing an error.

#### Scenario: Fresh cache reused

- **WHEN** the pricing cache was written within the freshness window
- **THEN** genifer uses the cached prices instead of refetching

#### Scenario: Stale cache refetched

- **WHEN** the pricing cache is older than the freshness window
- **THEN** genifer refetches pricing and updates the cache

#### Scenario: Missing or invalid cache

- **WHEN** the pricing cache is absent, unparsable, or an unrecognized version
- **THEN** genifer ignores it, starts normally, and rebuilds it on the next successful fetch

#### Scenario: No prices to persist

- **WHEN** no pricing could be fetched during a session
- **THEN** genifer does not overwrite the cache with empty data

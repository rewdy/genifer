# Spec Delta

## MODIFIED Requirements

### Requirement: Configuration file location and loading

The system SHALL read configuration from a `config.yaml` file in the user's
config directory (`~/.config/genifer/config.yaml` on Unix-like systems,
resolved via the platform config dir). Configuration SHALL define providers as a
list of keyed, typed instances: each entry SHALL have a unique `key` (its
identity in the UI and in persisted state), a `type` (which determines how the
instance is treated), and the fields that type requires. Two entries MAY share a
`type` while differing by `key`. The system SHALL treat this file as user-owned:
during normal operation it MUST NOT overwrite or modify an existing valid
`config.yaml`. The system MAY create `config.yaml` only when it is absent — via
the explicit `genifer config` command or the first-run onboarding flow — and
when creating it SHALL write a commented starter file; it MUST NOT create or
alter the file as a side effect of a normal session launch.

A `config.yaml` written in the legacy single-provider shape (a top-level
`provider:` string with a type-named settings block) SHALL NOT be loaded as-is.
The system SHALL detect that shape, preserve the user's file by renaming it to a
backup alongside the original (`config.yaml.bak`), and route the user into the
first-run onboarding flow to rebuild configuration in the current shape. This is
the one create path that may replace an existing `config.yaml`, and it SHALL do
so only after the original has been preserved as a backup.

#### Scenario: Config file present

- **WHEN** genifer starts and `config.yaml` exists in the current provider-list shape and is valid
- **THEN** its settings are loaded and used for the session

#### Scenario: Config file missing

- **WHEN** genifer starts and no `config.yaml` exists
- **THEN** genifer starts with built-in defaults and reports, in the TUI, that no config was found and where it is expected

#### Scenario: Config file malformed

- **WHEN** `config.yaml` exists but cannot be parsed
- **THEN** genifer surfaces a clear error identifying the file and does not silently fall back to defaults

#### Scenario: Legacy single-provider config recovered

- **WHEN** genifer starts and `config.yaml` is in the legacy single-provider shape
- **THEN** genifer renames the existing file to `config.yaml.bak`, does not load it as current configuration, and enters the first-run onboarding flow to rebuild the config

#### Scenario: Legacy backup does not clobber an existing backup

- **WHEN** a legacy `config.yaml` is being backed up and a `config.yaml.bak` already exists
- **THEN** the system preserves the existing backup rather than overwriting it, using a non-colliding backup name

#### Scenario: Existing config never overwritten on create

- **WHEN** a create path (the `config` command or onboarding) runs and a valid current-shape `config.yaml` already exists
- **THEN** the existing file is left byte-for-byte unchanged and is used as-is

### Requirement: Starter config generation

The system SHALL be able to generate a commented starter `config.yaml` that
documents the available settings and their defaults, written only when no valid
config file exists at the resolved path (or after a legacy file has been backed
up). The starter SHALL express configuration in the provider-list shape with at
least one provider instance, SHALL be valid YAML that loads without error, and
SHALL use the recommended API-key default (`{env:OPENROUTER_API_KEY}`) for an
OpenRouter instance unless a different choice is supplied by the caller (e.g.
onboarding).

#### Scenario: Starter written when absent

- **WHEN** a starter config is requested and no `config.yaml` exists
- **THEN** a commented `config.yaml` in the provider-list shape is written to the resolved config path, creating the config directory if needed, and the written file loads successfully as valid configuration

#### Scenario: Starter not written when present

- **WHEN** a starter config is requested and a valid current-shape `config.yaml` already exists
- **THEN** no file is written and the existing file is preserved

### Requirement: State persistence separate from config

The system SHALL persist last-used selections (at minimum, the last selected
model and the last selected aspect ratio) in a separate app-written file
(`state.json` in the same config directory). The remembered model SHALL be
recorded together with the provider key that owns it, so pre-selection resolves
to the correct provider's model when more than one provider is configured. The
system SHALL write to `state.json` only, never to `config.yaml`. When persisting
one selection, the system SHALL preserve the other previously persisted
selections rather than overwriting them.

#### Scenario: Remember last model

- **WHEN** the user selects a model from a provider and generates, then restarts genifer
- **THEN** the previously selected model under its provider is pre-selected on next launch

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

- **WHEN** the remembered model under its provider key is not present in the current merged model list
- **THEN** genifer does not pre-select it, falls back to no selection (or a default), and does not error

### Requirement: Pricing cache

The system SHALL cache per-model pricing in a separate app-written file
(`pricing-cache.json` in the config directory), reused for a bounded freshness
window (24 hours) to avoid refetching pricing on every launch. Cache entries
SHALL be keyed per provider instance (by provider key together with model id) so
that models sharing an id across providers do not overwrite each other's price.
The cache SHALL be app-owned (never `config.yaml`) and SHALL degrade gracefully:
a missing, unparsable, or wrong-version cache is ignored and refetched rather
than causing an error.

#### Scenario: Fresh cache reused

- **WHEN** the pricing cache was written within the freshness window
- **THEN** genifer uses the cached prices instead of refetching

#### Scenario: Stale cache refetched

- **WHEN** the pricing cache is older than the freshness window
- **THEN** genifer refetches pricing and updates the cache

#### Scenario: Prices distinguished across providers

- **WHEN** two configured providers each offer a model with the same identifier
- **THEN** the cache stores and returns a distinct price per provider instance rather than conflating them

#### Scenario: Missing or invalid cache

- **WHEN** the pricing cache is absent, unparsable, or an unrecognized version
- **THEN** genifer ignores it, starts normally, and rebuilds it on the next successful fetch

#### Scenario: No prices to persist

- **WHEN** no pricing could be fetched during a session
- **THEN** genifer does not overwrite the cache with empty data

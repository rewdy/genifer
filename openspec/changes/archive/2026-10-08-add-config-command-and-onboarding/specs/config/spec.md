# Spec Delta

## MODIFIED Requirements

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

## ADDED Requirements

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

# Spec Delta

## ADDED Requirements

### Requirement: First-run onboarding flow

When genifer launches and no `config.yaml` exists, the TUI SHALL present a first-run onboarding flow before the normal shell: a prominent WELCOME banner and a prompt to choose how the OpenRouter API key will be provided. The flow SHALL offer three choices — paste a literal key, reference an environment variable, or run a command — and SHALL write the resulting value into a newly created `config.yaml` as the matching directive before continuing into the normal session. The onboarding flow SHALL run only when no config file exists; when a config file is present, genifer SHALL skip onboarding and go straight to the normal shell.

#### Scenario: Onboarding shown on first run

- **WHEN** genifer launches and no `config.yaml` exists
- **THEN** the TUI displays a WELCOME banner and the API-key provider prompt instead of starting directly in the model picker

#### Scenario: Onboarding skipped when config exists

- **WHEN** genifer launches and `config.yaml` exists
- **THEN** the onboarding flow is not shown and the normal shell starts

#### Scenario: Choose environment variable

- **WHEN** the user chooses to provide the key via an environment variable and supplies the variable name
- **THEN** genifer writes the API key as `{env:NAME}` into the newly created `config.yaml` and continues into the normal session

#### Scenario: Choose command

- **WHEN** the user chooses to provide the key via a command and supplies the command
- **THEN** genifer writes the API key as `{cmd:...}` into the newly created `config.yaml` and continues into the normal session

#### Scenario: Choose paste with warning

- **WHEN** the user chooses to paste a literal key
- **THEN** genifer warns that the key will be stored as plain text in `config.yaml`, and on confirmation writes the pasted key as a literal value and continues

#### Scenario: Writing the config honors create-only ownership

- **WHEN** onboarding completes and writes `config.yaml`
- **THEN** it creates the file only because none existed, writing a commented starter populated with the chosen API-key directive, and never overwrites an existing config

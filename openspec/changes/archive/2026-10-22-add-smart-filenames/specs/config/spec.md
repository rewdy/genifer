# Spec Delta

## ADDED Requirements

### Requirement: Smart-filename settings

The system SHALL support two top-level configuration settings that control the
smart-filenames feature: a boolean `smart_filenames` that enables the feature
(default disabled; an unset value means disabled) and a string `filename_model`
naming the provider text model used to derive filenames (default empty, meaning
no model selected). The `filename_model` value SHALL be resolvable through the
generic value resolver like other string settings. The starter `config.yaml`
SHALL document both settings, their defaults, and SHALL remain valid YAML that
loads without error.

#### Scenario: Defaults when unset

- **WHEN** neither `smart_filenames` nor `filename_model` is set
- **THEN** the feature is disabled and no filename model is selected

#### Scenario: Enabled with a model

- **WHEN** `smart_filenames` is `true` and `filename_model` names a model id
- **THEN** both values are loaded and available to the smart-filenames feature

#### Scenario: Documented in starter config

- **WHEN** a starter `config.yaml` is generated
- **THEN** it documents `smart_filenames` and `filename_model` with their defaults and loads successfully as valid configuration

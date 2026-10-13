# config-ui Specification

## Purpose

Provides an in-tool graphical editor for the user's `config.yaml` covering the
common settings — save location, history enablement, and
auto-open-on-complete — writing edits back to that file and applying them to the
running session, without the user hand-editing the file.

## Requirements

### Requirement: In-tool config view exposes editable settings

The TUI SHALL provide a configuration view that displays the current value of
the save location, history-enabled, and auto-open-on-complete settings, and
allows the user to edit each.

#### Scenario: Config view shows current values
- **WHEN** the user opens the config view
- **THEN** it shows the current save location, whether history is enabled, and whether auto-open-on-complete is on

#### Scenario: User edits a setting
- **WHEN** the user changes a setting in the config view and confirms
- **THEN** the new value becomes the current value

### Requirement: Edits are written back to the user config file

Confirmed edits SHALL be written to the user's `config.yaml` so they persist
across restarts. The config view edits that file directly; there is no separate
override store. Writing the edited settings SHALL preserve the user's other
settings so that only the changed settings are affected.

#### Scenario: Edited setting survives restart
- **WHEN** the user changes a setting in-tool and restarts the app
- **THEN** the edited value is loaded from `config.yaml` and is in effect

#### Scenario: Other settings are preserved on write
- **WHEN** the user changes one setting in-tool
- **THEN** `config.yaml` reflects the change while the user's other settings (such as the provider and API-key configuration) remain intact

#### Scenario: Written file remains loadable
- **WHEN** the config view writes `config.yaml`
- **THEN** the resulting file parses successfully on the next load

### Requirement: Applicable edits take effect in the current session

Where it is safe to do so, a changed setting SHALL take effect for the running
session without requiring a restart. At minimum, a changed save location and a
changed auto-open preference SHALL apply to the next generation in the same
session.

#### Scenario: New save location applies to next generation
- **WHEN** the user changes the save location in-tool and then generates an image
- **THEN** the image is written to the newly chosen location

#### Scenario: Auto-open change applies immediately
- **WHEN** the user toggles auto-open-on-complete and then completes a generation
- **THEN** the new auto-open behavior is used for that generation

### Requirement: Invalid setting values are rejected

The config view SHALL reject a save-location value it cannot use (for example a
path that cannot be created or written) and SHALL keep the prior value rather
than writing an unusable setting to the file.

#### Scenario: Unusable save location is rejected
- **WHEN** the user enters a save location that cannot be created or written
- **THEN** the config view reports the problem, the prior save location remains in effect, and `config.yaml` is not changed

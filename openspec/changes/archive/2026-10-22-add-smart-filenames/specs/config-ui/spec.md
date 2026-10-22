# Spec Delta

## MODIFIED Requirements

### Requirement: In-tool config view exposes editable settings

The TUI SHALL provide a configuration view that displays the current value of
the save location, history-enabled, auto-open-on-complete, smart-filenames, and
filename-model settings, and allows the user to edit each. The smart-filenames
setting SHALL be presented as a toggle and the filename model as a free-text
field. The view SHALL show a reference to the OpenRouter models page so a user
who does not know which model id to enter can find one.

#### Scenario: Config view shows current values
- **WHEN** the user opens the config view
- **THEN** it shows the current save location, whether history is enabled, whether auto-open-on-complete is on, whether smart filenames is on, and the configured filename model

#### Scenario: User edits a setting
- **WHEN** the user changes a setting in the config view and confirms
- **THEN** the new value becomes the current value

#### Scenario: Smart-filename settings are editable
- **WHEN** the user toggles smart filenames and enters a filename model id in the config view and confirms
- **THEN** the new toggle state and model id become the current values

#### Scenario: OpenRouter models link is shown
- **WHEN** the user views the filename-model field in the config view
- **THEN** a reference to the OpenRouter models page is shown so the user can find a model id

### Requirement: Applicable edits take effect in the current session

Where it is safe to do so, a changed setting SHALL take effect for the running
session without requiring a restart. At minimum, a changed save location, a
changed auto-open preference, and a changed smart-filenames toggle or filename
model SHALL apply to the next generation in the same session.

#### Scenario: New save location applies to next generation
- **WHEN** the user changes the save location in-tool and then generates an image
- **THEN** the image is written to the newly chosen location

#### Scenario: Auto-open change applies immediately
- **WHEN** the user toggles auto-open-on-complete and then completes a generation
- **THEN** the new auto-open behavior is used for that generation

#### Scenario: Smart-filename change applies immediately
- **WHEN** the user changes the smart-filenames toggle or the filename model in-tool and then completes a generation
- **THEN** the new smart-filenames behavior is used when naming that generation's file

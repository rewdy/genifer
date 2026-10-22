# Spec Delta

## MODIFIED Requirements

### Requirement: Saving generated images

The system SHALL save a successfully generated image to the configured output
directory using a non-colliding file name whose extension matches the image's
media type, and SHALL report the saved file path to the user. The base name
SHALL be a timestamp by default. When the smart-filenames feature is enabled,
the system SHALL instead derive the base name from the generation prompt via the
smart-filenames feature, falling back to the timestamped base name whenever
smart naming does not yield a usable name. Deriving the smart name MUST NOT
block or fail the save: a successfully generated image is always saved.

#### Scenario: Save on success

- **WHEN** a generation succeeds
- **THEN** the image is written to the output directory with an extension matching its media type, and the resolved path is shown

#### Scenario: Smart name used when enabled

- **WHEN** a generation succeeds, smart filenames is enabled, and a usable name is derived from the prompt
- **THEN** the image is saved under the derived base name rather than the timestamp

#### Scenario: Fallback to timestamp

- **WHEN** a generation succeeds and smart filenames is disabled or does not produce a usable name
- **THEN** the image is saved under the timestamped base name

#### Scenario: Output directory does not exist

- **WHEN** the configured output directory does not yet exist at save time
- **THEN** genifer creates it before writing the file

#### Scenario: Write failure

- **WHEN** the image cannot be written (e.g. permissions)
- **THEN** genifer reports a clear error and does not claim success

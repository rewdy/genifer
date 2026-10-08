# Spec Delta

## MODIFIED Requirements

### Requirement: Output directory resolution

The system SHALL determine where generated images are written from an optional `output_dir` setting, resolved through the generic value resolver first, then interpreted as follows: unset uses a built-in default location; `.` or `pwd` means the current working directory; a value of `~` or beginning with `~/` has the leading `~` expanded to the user's home directory, yielding an absolute path; a relative path is resolved against the current working directory; an absolute path is used as-is.

#### Scenario: Default when unset

- **WHEN** `output_dir` is not set
- **THEN** images are written to the built-in default directory

#### Scenario: Current working directory

- **WHEN** `output_dir` is `.` or `pwd`
- **THEN** images are written to the directory genifer was launched from

#### Scenario: Home-relative path

- **WHEN** `output_dir` is `~/Downloads/genifer`
- **THEN** the leading `~` is expanded to the user's home directory and images are written to `<home>/Downloads/genifer`

#### Scenario: Bare home

- **WHEN** `output_dir` is `~`
- **THEN** images are written to the user's home directory

#### Scenario: Relative path

- **WHEN** `output_dir` is a relative path
- **THEN** it is resolved against the current working directory

#### Scenario: Absolute path

- **WHEN** `output_dir` is an absolute path
- **THEN** that path is used unchanged

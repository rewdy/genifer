# Spec Delta

## MODIFIED Requirements

### Requirement: Provider abstraction

The system SHALL define a provider interface that exposes, at minimum, the
ability to list available image models and to generate an image from a request.
Code above the provider SHALL depend only on this interface and MUST NOT depend
on provider-specific request or response formats, so that an additional provider
can be added without changing callers.

The system SHALL support more than one configured provider at once, holding a
registry of provider instances keyed by their configured identity. Each instance
is constructed from its configured `type`; an unrecognized type SHALL be a clear
configuration error naming the offending instance. A generation or discovery
request SHALL be routed to the provider instance that owns the selected model.

#### Scenario: Multiple providers configured

- **WHEN** genifer runs with more than one provider instance configured
- **THEN** each instance is constructed from its type and model listing and generation are served through the provider interface by the owning instance

#### Scenario: Single provider configured

- **WHEN** genifer runs with exactly one provider instance configured
- **THEN** model listing and generation are served through the provider interface by that instance

#### Scenario: Unknown provider type

- **WHEN** a configured provider instance declares a type the system does not recognize
- **THEN** genifer reports a clear configuration error identifying that instance rather than starting with it silently ignored

#### Scenario: Provider-agnostic request

- **WHEN** the workflow issues a generation request
- **THEN** it supplies model, prompt, optional reference images, and generation parameters without referencing any provider-specific field names

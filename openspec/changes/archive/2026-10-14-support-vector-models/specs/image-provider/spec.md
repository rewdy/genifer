# Spec Delta

## MODIFIED Requirements

### Requirement: Model discovery

The provider SHALL return the list of models usable for image generation, each with a stable identifier, a human-readable name, and a capability description sufficient to drive the UI (at minimum: whether the model accepts reference images, and which generation parameters such as aspect ratio and seed it supports). It SHALL additionally expose advertised reference-count bounds and output formats without requiring callers to interpret provider-specific metadata. An absent bound SHALL remain unknown rather than becoming a zero-reference maximum.

#### Scenario: List models

- **WHEN** the provider's model listing is requested with valid credentials
- **THEN** it returns the available image models with their identifiers, names, and capabilities

#### Scenario: Capability reporting

- **WHEN** a model supports reference images and an aspect-ratio parameter
- **THEN** the returned capability description reflects both, and omits parameters the model does not support

#### Scenario: Discovery failure

- **WHEN** the model list cannot be retrieved (network, auth, or server error)
- **THEN** the provider returns an error distinguishing authentication failures from other failures

#### Scenario: Required reference bounds
- **WHEN** discovery advertises a reference range of 1 through 10
- **THEN** capabilities report that at least one and at most ten references are accepted

#### Scenario: Optional references and SVG output
- **WHEN** discovery advertises a reference range starting at zero and only SVG output
- **THEN** capabilities permit text-only generation and identify SVG output

#### Scenario: Legacy reference capability
- **WHEN** discovery indicates reference support without numeric bounds
- **THEN** references remain optional and no numeric maximum is inferred

## ADDED Requirements

### Requirement: Reference input validation
The provider SHALL reject requests outside known reference-count bounds before sending a generation request, including requests with no references when references are required. It SHALL reject unsupported references using the existing unsupported-reference error, and distinguish invalid reference count, format, or dimensions with an actionable validation error. Recraft V4 Styles models (raster and vector, including Pro variants) SHALL accept only PNG, JPEG, or WebP references whose width and height are each at least 256 pixels. Other models SHALL retain their existing input rules except for advertised count bounds. Validation and discovery SHALL honor cancellation.

#### Scenario: Missing required reference
- **WHEN** generation is requested with zero references for a model requiring at least one
- **THEN** the provider returns a reference validation error and sends no generation request

#### Scenario: Too many references
- **WHEN** the supplied reference count exceeds the advertised maximum
- **THEN** the provider returns a reference validation error and sends no generation request

#### Scenario: Invalid style reference
- **WHEN** a Recraft V4 Styles request includes a GIF or an image whose shortest edge is below 256 pixels
- **THEN** generation is prevented with a message describing the accepted format or minimum dimensions

#### Scenario: Valid style reference
- **WHEN** a Recraft V4 Styles request includes one valid 256-by-256 PNG and an otherwise valid prompt
- **THEN** it passes reference validation and the image is submitted as a reference input

### Requirement: Native SVG output preservation
The provider SHALL decode SVG output using the same generation response contract as raster output, return the original decoded bytes with the reported SVG media type, and leave the model's output format at its provider default. It MUST NOT convert SVG to a raster image or infer SVG support from the model name.

#### Scenario: SVG response
- **WHEN** generation returns base64-encoded SVG bytes and media type `image/svg+xml`
- **THEN** the returned bytes are identical to the decoded response and the media type is preserved

#### Scenario: Default output selection
- **WHEN** the selected model advertises SVG output
- **THEN** generation uses the dedicated images endpoint without forcing a raster output format

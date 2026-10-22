# image-provider Specification

## Purpose

Defines the provider abstraction for discovering image models and generating images, and the OpenRouter implementation targeting its dedicated images API, including capability reporting, reference-image input, and error mapping.

## Requirements

### Requirement: Provider abstraction

The system SHALL define a provider interface that exposes, at minimum, the ability to list available image models and to generate an image from a request. Code above the provider SHALL depend only on this interface and MUST NOT depend on provider-specific request or response formats, so that an additional provider can be added without changing callers.

#### Scenario: Single provider configured

- **WHEN** genifer runs with OpenRouter configured
- **THEN** model listing and generation are served through the provider interface by the OpenRouter implementation

#### Scenario: Provider-agnostic request

- **WHEN** the workflow issues a generation request
- **THEN** it supplies model, prompt, optional reference images, and generation parameters without referencing any provider-specific field names

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

### Requirement: Model pricing

The provider SHALL expose, per model, a classified output-image price sufficient for the UI to display it: a per-image cost, a per-token indication, free, or unknown. Pricing retrieval SHALL honor cancellation and SHALL be treated as non-fatal by callers — an error means "price unknown", never a failure of discovery or generation.

#### Scenario: Per-image price

- **WHEN** a model bills a flat amount per output image
- **THEN** the provider reports a per-image classification with that amount

#### Scenario: Tiered per-image price

- **WHEN** a model bills per output image but the amount varies by resolution tier
- **THEN** the provider reports a per-image classification using the lowest tier's amount

#### Scenario: Per-token price

- **WHEN** a model bills per token for output images
- **THEN** the provider reports a per-token classification without a per-image amount

#### Scenario: Free model

- **WHEN** every output-image price for a model is zero
- **THEN** the provider reports the model as free

#### Scenario: Pricing retrieval fails

- **WHEN** pricing for a model cannot be retrieved
- **THEN** the provider returns an error that the caller treats as "price unknown" without aborting discovery or generation

### Requirement: Image generation via OpenRouter images API

The OpenRouter provider SHALL generate images by calling the dedicated images endpoint with a Bearer-token `Authorization` header, requesting a single image (`n=1`) for the MVP. It SHALL decode the returned base64 image data and determine the file type from the response's reported media type.

#### Scenario: Successful generation

- **WHEN** a generation request with a valid model and prompt succeeds
- **THEN** the provider returns the decoded image bytes and a file-type indication derived from the response media type (e.g. png, jpeg, webp, svg+xml)

#### Scenario: Actual cost reported

- **WHEN** a generation succeeds and the service reports a cost for it
- **THEN** the provider includes that actual cost (in US dollars) in the result

#### Scenario: Reference images for image-to-image

- **WHEN** the selected model accepts reference images and the request includes one or more
- **THEN** the provider submits them as reference inputs alongside the prompt

#### Scenario: Reference images on unsupported model

- **WHEN** the selected model does not accept reference images and the request includes one
- **THEN** generation is prevented with a clear error before the request is sent

### Requirement: Generation error mapping

The provider SHALL map known failure conditions to distinct, actionable errors — at minimum: insufficient account credit, generation failure (not billed), and authentication failure — so the UI can present a helpful message rather than a raw error.

#### Scenario: Insufficient credit

- **WHEN** generation is rejected for insufficient account credit
- **THEN** the provider returns an error identifying it as a credit problem

#### Scenario: Generation failed

- **WHEN** the service reports the generation itself failed
- **THEN** the provider returns a retryable "generation failed" error

#### Scenario: Authentication failure

- **WHEN** the credentials are missing or rejected
- **THEN** the provider returns an authentication error

### Requirement: Cancellable requests

The provider SHALL accept a cancellation signal for in-flight generation and model-discovery requests, and SHALL abort the underlying network call promptly when cancellation occurs.

#### Scenario: Cancel in-flight generation

- **WHEN** a generation request is cancelled before completion
- **THEN** the provider stops waiting for the response and returns a cancellation outcome rather than a result

### Requirement: Provider text completion for short derived text

The provider interface SHALL expose a text-completion operation that, given a
model identifier and a prompt, returns a short piece of generated text (used by
the system to derive a filename from a generation prompt). Code above the
provider SHALL depend only on this interface and MUST NOT depend on
provider-specific request or response formats. The operation SHALL honor
cancellation and SHALL map failures to the same actionable error categories as
image generation (at minimum authentication failure, insufficient credit, and a
general request failure) so callers can treat them uniformly. The OpenRouter
implementation SHALL fulfill this by calling the provider's chat/text
completions endpoint with a Bearer-token `Authorization` header.

#### Scenario: Short text returned

- **WHEN** the text-completion operation is called with a configured model and a prompt
- **THEN** it returns the model's generated text, trimmed of surrounding whitespace

#### Scenario: Cancellation aborts the request

- **WHEN** the caller cancels the context during an in-flight text-completion request
- **THEN** the underlying network call is aborted promptly and the operation returns a cancellation error

#### Scenario: Failures mapped to actionable errors

- **WHEN** the text-completion request is rejected for authentication or insufficient credit, or otherwise fails
- **THEN** the operation returns the corresponding actionable error category rather than a raw provider error

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

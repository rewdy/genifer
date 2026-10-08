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

The provider SHALL return the list of models usable for image generation, each with a stable identifier, a human-readable name, and a capability description sufficient to drive the UI (at minimum: whether the model accepts reference images, and which generation parameters such as aspect ratio and seed it supports).

#### Scenario: List models

- **WHEN** the provider's model listing is requested with valid credentials
- **THEN** it returns the available image models with their identifiers, names, and capabilities

#### Scenario: Capability reporting

- **WHEN** a model supports reference images and an aspect-ratio parameter
- **THEN** the returned capability description reflects both, and omits parameters the model does not support

#### Scenario: Discovery failure

- **WHEN** the model list cannot be retrieved (network, auth, or server error)
- **THEN** the provider returns an error distinguishing authentication failures from other failures

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

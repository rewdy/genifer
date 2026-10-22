# Spec Delta

## ADDED Requirements

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

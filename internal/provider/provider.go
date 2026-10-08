package provider

import (
	"context"
	"errors"
)

// Provider is the narrow, provider-agnostic seam the rest of genifer depends
// on. A new backend is added by implementing this interface; callers never see
// provider-specific request or response formats.
type Provider interface {
	// Models returns the image models available for generation. It honors ctx
	// cancellation.
	Models(ctx context.Context) ([]Model, error)

	// Pricing returns the classified output-image price for a single model. It
	// honors ctx cancellation. Pricing is decorative: callers treat an error as
	// "price unknown" rather than fatal.
	Pricing(ctx context.Context, modelID string) (Price, error)

	// Generate produces a single image for the request. It honors ctx
	// cancellation, aborting the underlying call promptly.
	Generate(ctx context.Context, req GenerateRequest) (GenerateResult, error)
}

// Model describes an image model offered by a provider.
type Model struct {
	// ID is the stable identifier used in a GenerateRequest.
	ID string
	// Name is a human-readable display name.
	Name string
	// Capabilities describes what the model supports, enough to drive the UI.
	Capabilities Capabilities
}

// Capabilities describes, in provider-agnostic terms, what a model supports.
type Capabilities struct {
	// AcceptsReferenceImages reports whether the model supports image-to-image
	// via reference images.
	AcceptsReferenceImages bool
	// AspectRatios lists selectable aspect ratios (e.g. "1:1", "16:9"). Empty
	// means the model does not expose an aspect-ratio control.
	AspectRatios []string
	// SupportsSeed reports whether the model accepts a seed parameter.
	SupportsSeed bool
}

// ReferenceImage is an input image supplied for image-to-image generation.
type ReferenceImage struct {
	// Data is the raw image bytes.
	Data []byte
	// MediaType is the IANA media type (e.g. "image/png").
	MediaType string
}

// GenerateRequest is a provider-agnostic request for a single image.
type GenerateRequest struct {
	// Model is the id of the model to use.
	Model string
	// Prompt is the text prompt.
	Prompt string
	// ReferenceImages are optional inputs for image-to-image. Supplying any for
	// a model whose Capabilities.AcceptsReferenceImages is false is an error.
	ReferenceImages []ReferenceImage
	// AspectRatio, when non-empty, requests a specific aspect ratio.
	AspectRatio string
	// Seed, when non-nil, requests deterministic generation.
	Seed *int64
}

// GenerateResult is the outcome of a successful generation.
type GenerateResult struct {
	// Data is the decoded image bytes.
	Data []byte
	// MediaType is the IANA media type of the image (e.g. "image/png"), used to
	// choose a file extension.
	MediaType string
	// CostUSD is the actual cost of this generation in US dollars, as reported
	// by the provider. Zero when the provider did not report a cost.
	CostUSD float64
}

// PriceUnit classifies how a model bills for output images.
type PriceUnit int

const (
	// PriceUnknown means pricing could not be determined.
	PriceUnknown PriceUnit = iota
	// PriceFree means output images cost nothing.
	PriceFree
	// PricePerImage means a flat per-image price (see Price.USD).
	PricePerImage
	// PricePerImageTiered means per-image pricing that varies by resolution;
	// Price.USD holds the lowest tier.
	PricePerImageTiered
	// PricePerToken means token-based pricing (no honest per-image figure).
	PricePerToken
)

// Price is a provider-agnostic, classified output-image price for a model.
type Price struct {
	Unit PriceUnit
	// USD is the per-image cost for PricePerImage, or the lowest tier for
	// PricePerImageTiered. Unused for other units.
	USD float64
}

// Sentinel errors let the UI present actionable messages instead of raw errors.
// Implementations wrap these with %w.
var (
	// ErrAuth indicates missing or rejected credentials.
	ErrAuth = errors.New("authentication failed")
	// ErrNoAPIKey indicates no API key is configured (the resolved key is
	// empty or its config directive could not be resolved). It is distinct from
	// ErrAuth, which indicates the service rejected a key that was sent. The UI
	// uses the distinction to tell the user to *set* a key versus *fix* one.
	ErrNoAPIKey = errors.New("no API key configured")
	// ErrInsufficientCredit indicates the account lacks the credit to generate.
	ErrInsufficientCredit = errors.New("insufficient account credit")
	// ErrGenerationFailed indicates the service failed to produce an image; it
	// is typically retryable and not billed.
	ErrGenerationFailed = errors.New("generation failed")
	// ErrReferenceImagesUnsupported indicates reference images were supplied for
	// a model that does not accept them.
	ErrReferenceImagesUnsupported = errors.New("model does not accept reference images")
)

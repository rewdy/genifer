package openrouter

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rewdy/genifer/internal/provider"
)

const defaultBaseURL = "https://openrouter.ai/api/v1"

// KeyFunc lazily supplies the API key. It is called on each request so a
// command-backed key is only resolved when a call is actually made.
type KeyFunc func(ctx context.Context) (string, error)

// Client implements provider.Provider against OpenRouter's images API.
type Client struct {
	baseURL string
	key     KeyFunc
	http    *http.Client

	// Attribution headers (optional). Referer must be set for Title to count.
	referer string
	title   string
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL overrides the API base URL (useful for tests).
func WithBaseURL(u string) Option {
	return func(c *Client) {
		if u != "" {
			c.baseURL = strings.TrimRight(u, "/")
		}
	}
}

// WithHTTPClient sets the underlying HTTP client.
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.http = h } }

// WithAttribution sets the optional HTTP-Referer / X-Title attribution headers.
func WithAttribution(referer, title string) Option {
	return func(c *Client) { c.referer, c.title = referer, title }
}

// New creates a Client. key is called lazily to obtain the bearer token.
func New(key KeyFunc, opts ...Option) *Client {
	c := &Client{
		baseURL: defaultBaseURL,
		key:     key,
		http:    &http.Client{Timeout: 120 * time.Second},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

var _ provider.Provider = (*Client)(nil)

func (c *Client) authHeader(ctx context.Context, req *http.Request) error {
	key, err := c.key(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", provider.ErrNoAPIKey, err)
	}
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("%w: resolved key is empty", provider.ErrNoAPIKey)
	}
	req.Header.Set("Authorization", "Bearer "+key)
	if c.referer != "" {
		req.Header.Set("HTTP-Referer", c.referer)
	}
	if c.title != "" {
		req.Header.Set("X-Title", c.title)
	}
	return nil
}

// --- Model discovery -------------------------------------------------------

// imageModelsResponse mirrors GET /images/models.
type imageModelsResponse struct {
	Data []imageModel `json:"data"`
}

type imageModel struct {
	ID                  string                        `json:"id"`
	Name                string                        `json:"name"`
	SupportedParameters map[string]supportedParameter `json:"supported_parameters"`
}

// supportedParameter is a capability descriptor. A missing key means the model
// does not support that parameter.
type supportedParameter struct {
	Type   string   `json:"type"`             // "enum" | "range" | "boolean"
	Values []string `json:"values,omitempty"` // for enum
}

func (c *Client) Models(ctx context.Context) ([]provider.Model, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/images/models", nil)
	if err != nil {
		return nil, err
	}
	if err := c.authHeader(ctx, req); err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, mapTransportError(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, mapHTTPError(resp.StatusCode, body)
	}
	var parsed imageModelsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("openrouter: decoding models: %w", err)
	}
	models := make([]provider.Model, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		models = append(models, provider.Model{
			ID:           m.ID,
			Name:         orName(m),
			Capabilities: capsFrom(m.SupportedParameters),
		})
	}
	return models, nil
}

func orName(m imageModel) string {
	if m.Name != "" {
		return m.Name
	}
	return m.ID
}

func capsFrom(params map[string]supportedParameter) provider.Capabilities {
	caps := provider.Capabilities{}
	if _, ok := params["input_references"]; ok {
		caps.AcceptsReferenceImages = true
	}
	if p, ok := params["aspect_ratio"]; ok {
		caps.AspectRatios = p.Values
	}
	if _, ok := params["seed"]; ok {
		caps.SupportsSeed = true
	}
	return caps
}

// --- Pricing ---------------------------------------------------------------

// endpointsResponse mirrors GET /images/models/{author}/{slug}/endpoints.
type endpointsResponse struct {
	Endpoints []struct {
		Pricing []pricingLine `json:"pricing"`
	} `json:"endpoints"`
}

type pricingLine struct {
	Billable string  `json:"billable"` // e.g. "output_image", "input_image"
	Unit     string  `json:"unit"`     // "image" | "megapixel" | "token"
	CostUSD  float64 `json:"cost_usd"`
	Variant  string  `json:"variant,omitempty"` // resolution tier, e.g. "768", "1k"
}

// Pricing fetches and classifies the output-image price for a single model.
func (c *Client) Pricing(ctx context.Context, modelID string) (provider.Price, error) {
	url := c.baseURL + "/images/models/" + modelID + "/endpoints"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return provider.Price{}, err
	}
	if err := c.authHeader(ctx, req); err != nil {
		return provider.Price{}, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return provider.Price{}, mapTransportError(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return provider.Price{}, mapHTTPError(resp.StatusCode, body)
	}
	var parsed endpointsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return provider.Price{}, fmt.Errorf("openrouter: decoding endpoints: %w", err)
	}
	return classifyPricing(parsed), nil
}

// classifyPricing reduces all endpoints' output_image pricing lines into a
// single provider.Price. Across multiple endpoints and resolution tiers it
// reports the lowest per-image cost. A model whose every output_image line is
// zero is Free; a model priced only per token is PerToken.
func classifyPricing(r endpointsResponse) provider.Price {
	var (
		haveImageLine bool
		tiered        bool
		haveToken     bool
		minUSD        float64
		haveMin       bool
	)
	for _, ep := range r.Endpoints {
		for _, line := range ep.Pricing {
			if line.Billable != "output_image" {
				continue
			}
			switch line.Unit {
			case "image", "megapixel":
				haveImageLine = true
				if line.Variant != "" {
					tiered = true
				}
				if !haveMin || line.CostUSD < minUSD {
					minUSD, haveMin = line.CostUSD, true
				}
			case "token":
				haveToken = true
			}
		}
	}

	switch {
	case haveImageLine && minUSD == 0:
		return provider.Price{Unit: provider.PriceFree}
	case haveImageLine && tiered:
		return provider.Price{Unit: provider.PricePerImageTiered, USD: minUSD}
	case haveImageLine:
		return provider.Price{Unit: provider.PricePerImage, USD: minUSD}
	case haveToken:
		return provider.Price{Unit: provider.PricePerToken}
	default:
		return provider.Price{Unit: provider.PriceUnknown}
	}
}

// --- Generation ------------------------------------------------------------

type generatePayload struct {
	Model           string           `json:"model"`
	Prompt          string           `json:"prompt"`
	N               int              `json:"n"`
	AspectRatio     string           `json:"aspect_ratio,omitempty"`
	Seed            *int64           `json:"seed,omitempty"`
	InputReferences []inputReference `json:"input_references,omitempty"`
}

type inputReference struct {
	Type     string        `json:"type"`
	ImageURL imageURLField `json:"image_url"`
}

type imageURLField struct {
	URL string `json:"url"`
}

type generateResponse struct {
	Data []struct {
		B64JSON   string `json:"b64_json"`
		MediaType string `json:"media_type"`
	} `json:"data"`
	Usage struct {
		Cost float64 `json:"cost"`
	} `json:"usage"`
}

func (c *Client) Generate(ctx context.Context, r provider.GenerateRequest) (provider.GenerateResult, error) {
	if len(r.ReferenceImages) > 0 {
		// Guard: reject reference images for a model that cannot accept them.
		// Capability is known to the caller; we also need the model's caps here.
		caps, err := c.capabilitiesFor(ctx, r.Model)
		if err != nil {
			return provider.GenerateResult{}, err
		}
		if !caps.AcceptsReferenceImages {
			return provider.GenerateResult{}, fmt.Errorf("%w: %s", provider.ErrReferenceImagesUnsupported, r.Model)
		}
	}

	payload := generatePayload{
		Model:       r.Model,
		Prompt:      r.Prompt,
		N:           1,
		AspectRatio: r.AspectRatio,
		Seed:        r.Seed,
	}
	for _, ref := range r.ReferenceImages {
		payload.InputReferences = append(payload.InputReferences, inputReference{
			Type:     "image_url",
			ImageURL: imageURLField{URL: dataURL(ref)},
		})
	}

	buf, err := json.Marshal(payload)
	if err != nil {
		return provider.GenerateResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/images", bytes.NewReader(buf))
	if err != nil {
		return provider.GenerateResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if err := c.authHeader(ctx, req); err != nil {
		return provider.GenerateResult{}, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return provider.GenerateResult{}, mapTransportError(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return provider.GenerateResult{}, mapHTTPError(resp.StatusCode, body)
	}
	var parsed generateResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return provider.GenerateResult{}, fmt.Errorf("openrouter: decoding generation: %w", err)
	}
	if len(parsed.Data) == 0 || parsed.Data[0].B64JSON == "" {
		return provider.GenerateResult{}, fmt.Errorf("%w: empty response", provider.ErrGenerationFailed)
	}
	data, err := base64.StdEncoding.DecodeString(parsed.Data[0].B64JSON)
	if err != nil {
		return provider.GenerateResult{}, fmt.Errorf("openrouter: decoding image bytes: %w", err)
	}
	return provider.GenerateResult{Data: data, MediaType: parsed.Data[0].MediaType, CostUSD: parsed.Usage.Cost}, nil
}

// capabilitiesFor looks up a single model's capabilities via Models.
func (c *Client) capabilitiesFor(ctx context.Context, id string) (provider.Capabilities, error) {
	models, err := c.Models(ctx)
	if err != nil {
		return provider.Capabilities{}, err
	}
	for _, m := range models {
		if m.ID == id {
			return m.Capabilities, nil
		}
	}
	return provider.Capabilities{}, fmt.Errorf("openrouter: unknown model %q", id)
}

func dataURL(ref provider.ReferenceImage) string {
	mt := ref.MediaType
	if mt == "" {
		mt = "image/png"
	}
	return "data:" + mt + ";base64," + base64.StdEncoding.EncodeToString(ref.Data)
}

// --- Error mapping ---------------------------------------------------------

func mapTransportError(err error) error {
	// Preserve context cancellation/deadline so callers can detect it with
	// errors.Is, even though net/http wraps the cause in a *url.Error.
	switch {
	case errors.Is(err, context.Canceled):
		return context.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		return context.DeadlineExceeded
	default:
		return fmt.Errorf("openrouter: request failed: %w", err)
	}
}

func mapHTTPError(status int, body []byte) error {
	detail := extractMessage(body)
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("%w: %s", provider.ErrAuth, detail)
	case http.StatusPaymentRequired:
		return fmt.Errorf("%w: %s", provider.ErrInsufficientCredit, detail)
	case http.StatusBadGateway:
		return fmt.Errorf("%w: %s", provider.ErrGenerationFailed, detail)
	default:
		return fmt.Errorf("openrouter: unexpected status %d: %s", status, detail)
	}
}

// extractMessage pulls a human message from an OpenRouter error body if present.
func extractMessage(body []byte) string {
	var env struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &env); err == nil && env.Error.Message != "" {
		return env.Error.Message
	}
	s := strings.TrimSpace(string(body))
	if s == "" {
		return "(no detail)"
	}
	return s
}

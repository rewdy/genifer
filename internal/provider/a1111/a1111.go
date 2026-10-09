package a1111

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

// DefaultBaseURL is the WebUI address assumed when none is configured.
const DefaultBaseURL = "http://127.0.0.1:7860"

// aspectDimensions maps each offered aspect ratio to the output pixel
// dimensions sent to the WebUI. It is the single source of truth for both the
// advertised capability list and the txt2img width/height.
var aspectDimensions = []struct {
	ratio         string
	width, height int
}{
	{"1:1", 1024, 1024},
	{"3:2", 1216, 832},
	{"2:3", 832, 1216},
	{"16:9", 1344, 768},
	{"9:16", 768, 1344},
}

// offeredAspectRatios returns the aspect-ratio keys in advertised order.
func offeredAspectRatios() []string {
	out := make([]string, len(aspectDimensions))
	for i, d := range aspectDimensions {
		out[i] = d.ratio
	}
	return out
}

// dimsFor returns the width/height for an aspect ratio, falling back to the
// first (square) entry when the ratio is empty or unknown.
func dimsFor(ratio string) (int, int) {
	for _, d := range aspectDimensions {
		if d.ratio == ratio {
			return d.width, d.height
		}
	}
	return aspectDimensions[0].width, aspectDimensions[0].height
}

// localCapabilities is the fixed capability description for every local model.
func localCapabilities() provider.Capabilities {
	return provider.Capabilities{
		AcceptsReferenceImages: false,
		AspectRatios:           offeredAspectRatios(),
		SupportsSeed:           true,
	}
}

// Client implements provider.Provider against an A1111-compatible WebUI.
type Client struct {
	baseURL string
	http    *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets the underlying HTTP client (useful for tests).
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.http = h } }

// New creates a Client for the given WebUI base URL. An empty baseURL falls
// back to DefaultBaseURL.
func New(baseURL string, opts ...Option) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	c := &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 300 * time.Second},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

var _ provider.Provider = (*Client)(nil)

// --- Model discovery -------------------------------------------------------

// sdModel mirrors an entry of GET /sdapi/v1/sd-models.
type sdModel struct {
	Title     string `json:"title"`
	ModelName string `json:"model_name"`
}

func (c *Client) Models(ctx context.Context) ([]provider.Model, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/sdapi/v1/sd-models", nil)
	if err != nil {
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
	var parsed []sdModel
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("a1111: decoding models: %w", err)
	}
	models := make([]provider.Model, 0, len(parsed))
	for _, m := range parsed {
		id := m.Title
		if id == "" {
			id = m.ModelName
		}
		name := m.ModelName
		if name == "" {
			name = m.Title
		}
		models = append(models, provider.Model{
			ID:           id,
			Name:         name,
			Capabilities: localCapabilities(),
		})
	}
	return models, nil
}

// --- Pricing ---------------------------------------------------------------

// Pricing reports every local model as free: generation runs on the user's own
// hardware. It honors ctx but makes no network call.
func (c *Client) Pricing(ctx context.Context, modelID string) (provider.Price, error) {
	if err := ctx.Err(); err != nil {
		return provider.Price{}, err
	}
	return provider.Price{Unit: provider.PriceFree}, nil
}

// --- Generation ------------------------------------------------------------

// txt2imgRequest mirrors the subset of POST /sdapi/v1/txt2img we send.
type txt2imgRequest struct {
	Prompt string `json:"prompt"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Seed   *int64 `json:"seed,omitempty"`
}

// txt2imgResponse mirrors the fields of the txt2img response we read.
type txt2imgResponse struct {
	Images []string `json:"images"`
}

func (c *Client) Generate(ctx context.Context, r provider.GenerateRequest) (provider.GenerateResult, error) {
	// Reject reference images before any call: local models cannot accept them.
	if len(r.ReferenceImages) > 0 {
		return provider.GenerateResult{}, fmt.Errorf("%w: %s", provider.ErrReferenceImagesUnsupported, r.Model)
	}

	w, h := dimsFor(r.AspectRatio)
	payload := txt2imgRequest{
		Prompt: r.Prompt,
		Width:  w,
		Height: h,
		Seed:   r.Seed,
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		return provider.GenerateResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/sdapi/v1/txt2img", bytes.NewReader(buf))
	if err != nil {
		return provider.GenerateResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return provider.GenerateResult{}, mapTransportError(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return provider.GenerateResult{}, mapHTTPError(resp.StatusCode, body)
	}
	var parsed txt2imgResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return provider.GenerateResult{}, fmt.Errorf("a1111: decoding generation: %w", err)
	}
	if len(parsed.Images) == 0 || parsed.Images[0] == "" {
		return provider.GenerateResult{}, fmt.Errorf("%w: empty response", provider.ErrGenerationFailed)
	}
	// A1111 may prefix the base64 with a data URL; strip it if present.
	raw := parsed.Images[0]
	if i := strings.Index(raw, ","); strings.HasPrefix(raw, "data:") && i >= 0 {
		raw = raw[i+1:]
	}
	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return provider.GenerateResult{}, fmt.Errorf("a1111: decoding image bytes: %w", err)
	}
	return provider.GenerateResult{Data: data, MediaType: "image/png"}, nil
}

// --- Error mapping ---------------------------------------------------------

// mapTransportError preserves cancellation/deadline and maps every other
// transport failure (connection refused, no route, DNS) to ErrUnreachable so
// the UI can mark the instance offline.
func mapTransportError(err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return context.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		return context.DeadlineExceeded
	default:
		return fmt.Errorf("%w: %v", provider.ErrUnreachable, err)
	}
}

func mapHTTPError(status int, body []byte) error {
	detail := strings.TrimSpace(string(body))
	if detail == "" {
		detail = "(no detail)"
	}
	switch status {
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return fmt.Errorf("%w: WebUI returned %d", provider.ErrUnreachable, status)
	case http.StatusInternalServerError:
		return fmt.Errorf("%w: WebUI error: %s", provider.ErrGenerationFailed, detail)
	default:
		return fmt.Errorf("a1111: unexpected status %d: %s", status, detail)
	}
}

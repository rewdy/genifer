package a1111

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rewdy/genifer/internal/provider"
)

func newTestClient(h http.Handler) (*Client, *httptest.Server) {
	srv := httptest.NewServer(h)
	c := New(srv.URL, WithHTTPClient(srv.Client()))
	return c, srv
}

const sampleModels = `[
  {"title":"sd_xl_base_1.0.safetensors [abc123]","model_name":"sd_xl_base_1.0"},
  {"title":"dreamshaper_8.safetensors [def456]","model_name":"dreamshaper_8"}
]`

// 6.2 model discovery
func TestModelsMapping(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/sdapi/v1/sd-models") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Write([]byte(sampleModels))
	}))
	defer srv.Close()

	models, err := c.Models(context.Background())
	if err != nil {
		t.Fatalf("Models: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("got %d models, want 2", len(models))
	}
	if models[0].ID != "sd_xl_base_1.0.safetensors [abc123]" {
		t.Errorf("ID = %q", models[0].ID)
	}
	if models[0].Name != "sd_xl_base_1.0" {
		t.Errorf("Name = %q", models[0].Name)
	}
	if !models[0].Capabilities.SupportsSeed || models[0].Capabilities.AcceptsReferenceImages {
		t.Errorf("caps = %+v", models[0].Capabilities)
	}
}

// 6.2 cancellation
func TestModelsCancelled(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sampleModels))
	}))
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.Models(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

// 6.1 unreachable mapping (discovery)
func TestModelsUnreachable(t *testing.T) {
	// Point at a closed server to force a connection error.
	srv := httptest.NewServer(http.NewServeMux())
	url := srv.URL
	srv.Close()
	c := New(url)
	_, err := c.Models(context.Background())
	if !errors.Is(err, provider.ErrUnreachable) {
		t.Fatalf("err = %v, want ErrUnreachable", err)
	}
	if errors.Is(err, provider.ErrAuth) {
		t.Error("unreachable must not be classified as auth")
	}
}

// 6.3 pricing + capabilities
func TestPricingAndCapabilities(t *testing.T) {
	c := New("")
	price, err := c.Pricing(context.Background(), "any-model")
	if err != nil {
		t.Fatalf("Pricing: %v", err)
	}
	if price.Unit != provider.PriceFree {
		t.Errorf("price = %+v, want PriceFree", price)
	}
	caps := localCapabilities()
	if !caps.SupportsSeed {
		t.Error("want SupportsSeed true")
	}
	if caps.AcceptsReferenceImages {
		t.Error("want AcceptsReferenceImages false")
	}
	if len(caps.AspectRatios) == 0 {
		t.Error("want a non-empty fixed aspect-ratio list")
	}
}

// 6.4 successful generation + aspect ratio -> dimensions
func TestGenerateSuccessAndDimensions(t *testing.T) {
	imgBytes := []byte("fake-png-bytes")
	b64 := base64.StdEncoding.EncodeToString(imgBytes)

	var gotReq txt2imgRequest
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/sdapi/v1/txt2img") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotReq)
		resp, _ := json.Marshal(txt2imgResponse{Images: []string{b64}})
		w.Write(resp)
	}))
	defer srv.Close()

	res, err := c.Generate(context.Background(), provider.GenerateRequest{
		Model:       "sd-xl",
		Prompt:      "a cat",
		AspectRatio: "16:9",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if string(res.Data) != string(imgBytes) {
		t.Errorf("decoded data mismatch")
	}
	if res.MediaType != "image/png" {
		t.Errorf("MediaType = %q", res.MediaType)
	}
	wantW, wantH := dimsFor("16:9")
	if gotReq.Width != wantW || gotReq.Height != wantH {
		t.Errorf("sent dims %dx%d, want %dx%d", gotReq.Width, gotReq.Height, wantW, wantH)
	}
	if gotReq.Prompt != "a cat" {
		t.Errorf("prompt = %q", gotReq.Prompt)
	}
}

// 6.4 reference images rejected before any call
func TestGenerateRejectsReferenceImages(t *testing.T) {
	called := false
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Write([]byte(`{"images":[""]}`))
	}))
	defer srv.Close()

	_, err := c.Generate(context.Background(), provider.GenerateRequest{
		Model:           "sd-xl",
		Prompt:          "a cat",
		ReferenceImages: []provider.ReferenceImage{{Data: []byte("x"), MediaType: "image/png"}},
	})
	if !errors.Is(err, provider.ErrReferenceImagesUnsupported) {
		t.Fatalf("err = %v, want ErrReferenceImagesUnsupported", err)
	}
	if called {
		t.Error("no request should have been sent to the WebUI")
	}
}

// 6.4 generation cancellation
func TestGenerateCancelled(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"images":["AAAA"]}`))
	}))
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.Generate(ctx, provider.GenerateRequest{Model: "sd-xl", Prompt: "a cat"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

// 6.1 unreachable mapping (generation)
func TestGenerateUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.NewServeMux())
	url := srv.URL
	srv.Close()
	c := New(url)
	_, err := c.Generate(context.Background(), provider.GenerateRequest{Model: "sd-xl", Prompt: "a cat"})
	if !errors.Is(err, provider.ErrUnreachable) {
		t.Fatalf("err = %v, want ErrUnreachable", err)
	}
}

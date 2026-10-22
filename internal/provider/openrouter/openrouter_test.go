package openrouter

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rewdy/genifer/internal/provider"
)

func staticKey(string) KeyFunc {
	return func(context.Context) (string, error) { return "test-key", nil }
}

func newTestClient(h http.Handler) (*Client, *httptest.Server) {
	srv := httptest.NewServer(h)
	c := New(staticKey("x"), WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	return c, srv
}

const sampleModels = `{
  "data": [
    {"id":"google/gemini-2.5-flash-image","name":"Gemini 2.5 Flash Image",
     "supported_parameters":{"aspect_ratio":{"type":"enum","values":["1:1","16:9"]},"input_references":{"type":"range","min":0},"seed":{"type":"range"},"background":{"type":"enum","values":["auto","transparent"]},"output_format":{"type":"enum","values":["png","jpeg"]}}},
    {"id":"some/basic-model","name":"Basic",
     "supported_parameters":{}}
  ]
}`

// 4.1 model discovery + capability mapping
func TestModelsMapping(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/images/models") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("auth header = %q", got)
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
	g := models[0]
	if !g.Capabilities.SupportsTransparentBackground {
		t.Error("gemini should support transparent backgrounds")
	}
	if !g.Capabilities.SupportsSeed {
		t.Error("gemini should support seed")
	}
	if len(g.Capabilities.AspectRatios) != 2 {
		t.Errorf("gemini aspect ratios = %v", g.Capabilities.AspectRatios)
	}
	b := models[1]
	if b.Capabilities.AcceptsReferenceImages || b.Capabilities.SupportsSeed || len(b.Capabilities.AspectRatios) != 0 {
		t.Errorf("basic model should expose no capabilities, got %+v", b.Capabilities)
	}
}

// 4.1 auth failure distinguished
func TestModelsAuthError(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"invalid key"}}`))
	}))
	defer srv.Close()
	_, err := c.Models(context.Background())
	if !errors.Is(err, provider.ErrAuth) {
		t.Fatalf("err = %v, want ErrAuth", err)
	}
}

// 4.2 successful generation decoding
func TestGenerateSuccess(t *testing.T) {
	want := []byte("\x89PNG fake bytes")
	b64 := base64.StdEncoding.EncodeToString(want)
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/images/models" {
			w.Write([]byte(`{"data":[{"id":"m"}]}`))
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/images") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Write([]byte(`{"data":[{"b64_json":"` + b64 + `","media_type":"image/png"}]}`))
	}))
	defer srv.Close()

	res, err := c.Generate(context.Background(), provider.GenerateRequest{Model: "m", Prompt: "cat"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if string(res.Data) != string(want) {
		t.Errorf("decoded bytes mismatch")
	}
	if res.MediaType != "image/png" {
		t.Errorf("MediaType = %q", res.MediaType)
	}
}

// 1.3 provider-reported seed is parsed opportunistically from the response,
// from either a per-image or top-level field, and is nil when absent.
func TestGenerateParsesReportedSeed(t *testing.T) {
	b64 := base64.StdEncoding.EncodeToString([]byte("img"))
	cases := []struct {
		name string
		body string
		want *int64
	}{
		{"per-image seed", `{"data":[{"b64_json":"` + b64 + `","media_type":"image/png","seed":42}]}`, ptrInt64(42)},
		{"top-level seed", `{"data":[{"b64_json":"` + b64 + `","media_type":"image/png"}],"seed":7}`, ptrInt64(7)},
		{"no seed", `{"data":[{"b64_json":"` + b64 + `","media_type":"image/png"}]}`, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/images/models" {
					w.Write([]byte(`{"data":[{"id":"m"}]}`))
					return
				}
				w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			res, err := c.Generate(context.Background(), provider.GenerateRequest{Model: "m", Prompt: "cat"})
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			switch {
			case tc.want == nil && res.Seed != nil:
				t.Errorf("Seed = %d, want nil", *res.Seed)
			case tc.want != nil && res.Seed == nil:
				t.Errorf("Seed = nil, want %d", *tc.want)
			case tc.want != nil && *res.Seed != *tc.want:
				t.Errorf("Seed = %d, want %d", *res.Seed, *tc.want)
			}
		})
	}
}

func ptrInt64(v int64) *int64 { return &v }

// 4.3 reference images allowed for a capable model
func TestGenerateWithReferenceSupported(t *testing.T) {
	b64 := base64.StdEncoding.EncodeToString([]byte("img"))
	var sawRefs bool
	mux := http.NewServeMux()
	mux.HandleFunc("/images/models", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sampleModels))
	})
	mux.HandleFunc("/images", func(w http.ResponseWriter, r *http.Request) {
		buf, _ := io.ReadAll(r.Body)
		sawRefs = strings.Contains(string(buf), "input_references")
		w.Write([]byte(`{"data":[{"b64_json":"` + b64 + `","media_type":"image/png"}]}`))
	})
	c, srv := newTestClient(mux)
	defer srv.Close()

	_, err := c.Generate(context.Background(), provider.GenerateRequest{
		Model:           "google/gemini-2.5-flash-image",
		Prompt:          "cat",
		ReferenceImages: []provider.ReferenceImage{{Data: testPNG(t), MediaType: "image/png"}},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !sawRefs {
		t.Error("expected input_references in request body")
	}
}

// 4.3 reference images rejected for an incapable model, before sending
func TestGenerateWithReferenceUnsupported(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/images/models") {
			w.Write([]byte(sampleModels))
			return
		}
		t.Error("generation endpoint must not be called when refs unsupported")
	}))
	defer srv.Close()

	_, err := c.Generate(context.Background(), provider.GenerateRequest{
		Model:           "some/basic-model",
		Prompt:          "cat",
		ReferenceImages: []provider.ReferenceImage{{Data: []byte("x")}},
	})
	if !errors.Is(err, provider.ErrReferenceImagesUnsupported) {
		t.Fatalf("err = %v, want ErrReferenceImagesUnsupported", err)
	}
}

// 4.1 transparent background requested and supported
func TestGenerateWithTransparentBackgroundAndPngOutput(t *testing.T) {
	b64 := base64.StdEncoding.EncodeToString([]byte("img"))
	var sawTransparent bool
	var sawPngOutput bool
	mux := http.NewServeMux()
	mux.HandleFunc("/images/models", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sampleModels))
	})
	mux.HandleFunc("/images", func(w http.ResponseWriter, r *http.Request) {
		buf, _ := io.ReadAll(r.Body)
		sawTransparent = strings.Contains(string(buf), "\"background\":\"transparent\"")
		sawPngOutput = strings.Contains(string(buf), "\"output_format\":\"png\"")
		w.Write([]byte(`{"data":[{"b64_json":"` + b64 + `","media_type":"image/png"}]}`))
	})
	c, srv := newTestClient(mux)
	defer srv.Close()

	_, err := c.Generate(context.Background(), provider.GenerateRequest{
		Model:        "google/gemini-2.5-flash-image",
		Prompt:       "cat",
		Background:   "transparent",
		OutputFormat: "png",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !sawTransparent {
		t.Error("expected transparent background in request body")
	}
	if !sawPngOutput {
		t.Error("expected png output format in request body")
	}
}

// 4.1 transparent background requested but model does not support it
func TestGenerateWithoutTransparentBackgroundIfModelNotSupported(t *testing.T) {
	b64 := base64.StdEncoding.EncodeToString([]byte("img"))
	var sawTransparent bool
	mux := http.NewServeMux()
	mux.HandleFunc("/images/models", func(w http.ResponseWriter, r *http.Request) {
		// A sample model that does NOT support transparent backgrounds
		w.Write([]byte(`{"data":[{"id":"some/basic-model","name":"Basic","supported_parameters":{}}]}`))
	})
	mux.HandleFunc("/images", func(w http.ResponseWriter, r *http.Request) {
		buf, _ := io.ReadAll(r.Body)
		sawTransparent = strings.Contains(string(buf), "\"background\":\"transparent\"")
		w.Write([]byte(`{"data":[{"b64_json":"` + b64 + `","media_type":"image/png"}]}`))
	})
	c, srv := newTestClient(mux)
	defer srv.Close()

	_, err := c.Generate(context.Background(), provider.GenerateRequest{
		Model:        "some/basic-model",
		Prompt:       "dog",
		Background:   "transparent",
		OutputFormat: "png",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if sawTransparent {
		t.Error("did not expect transparent background in request body for unsupported model")
	}
}

// 4.4 error mapping
func TestGenerateErrorMapping(t *testing.T) {
	cases := []struct {
		status int
		want   error
	}{
		{http.StatusPaymentRequired, provider.ErrInsufficientCredit},
		{http.StatusBadGateway, provider.ErrGenerationFailed},
		{http.StatusUnauthorized, provider.ErrAuth},
	}
	for _, tc := range cases {
		c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/images/models" {
				w.Write([]byte(`{"data":[{"id":"m"}]}`))
				return
			}
			w.WriteHeader(tc.status)
			w.Write([]byte(`{"error":{"message":"x"}}`))
		}))
		_, err := c.Generate(context.Background(), provider.GenerateRequest{Model: "m", Prompt: "p"})
		if !errors.Is(err, tc.want) {
			t.Errorf("status %d: err = %v, want %v", tc.status, err, tc.want)
		}
		srv.Close()
	}
}

// 4.4 cancellation aborts the request
func TestGenerateCancellation(t *testing.T) {
	release := make(chan struct{})
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done(): // client cancelled
		case <-release: // test cleanup, unblock the handler
		}
	}))
	defer srv.Close()
	defer close(release)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	_, err := c.Generate(ctx, provider.GenerateRequest{Model: "m", Prompt: "p"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	srv.CloseClientConnections()
}

// auth: empty key surfaces as ErrNoAPIKey (distinct from a rejected key)
// without hitting the network
func TestEmptyKeyIsNoAPIKeyError(t *testing.T) {
	c := New(func(context.Context) (string, error) { return "", nil }, WithBaseURL("http://127.0.0.1:0"))
	_, err := c.Models(context.Background())
	if !errors.Is(err, provider.ErrNoAPIKey) {
		t.Fatalf("err = %v, want ErrNoAPIKey", err)
	}
	if errors.Is(err, provider.ErrAuth) {
		t.Error("a missing key must not also be ErrAuth (service-rejection)")
	}
}

// auth: a key-resolution failure (e.g. unset env var) is ErrNoAPIKey
func TestKeyResolutionFailureIsNoAPIKeyError(t *testing.T) {
	c := New(func(context.Context) (string, error) {
		return "", errors.New("environment variable \"OPENROUTER_API_KEY\" is not set")
	}, WithBaseURL("http://127.0.0.1:0"))
	_, err := c.Models(context.Background())
	if !errors.Is(err, provider.ErrNoAPIKey) {
		t.Fatalf("err = %v, want ErrNoAPIKey", err)
	}
}

func testPNG(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 256, 256))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// 2.2 GenerateText posts a chat completion and returns the trimmed content;
// auth and credit failures map to the provider sentinels.
func TestGenerateText(t *testing.T) {
	var gotPath string
	var gotBody []byte
	var gotAuth string
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotBody, _ = io.ReadAll(r.Body)
		w.Write([]byte(`{"choices":[{"message":{"content":"  a-descriptive-name  "}}]}`))
	}))
	defer srv.Close()

	got, err := c.GenerateText(context.Background(), provider.TextRequest{Model: "some/text-model", Prompt: "a cozy lighthouse at dusk"})
	if err != nil {
		t.Fatalf("GenerateText: %v", err)
	}
	if gotPath != "/chat/completions" {
		t.Errorf("path = %q, want /chat/completions", gotPath)
	}
	if gotAuth != "Bearer test-key" {
		t.Errorf("auth header = %q", gotAuth)
	}
	var payload chatCompletionPayload
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatalf("payload is not valid chat JSON: %v", err)
	}
	if payload.Model != "some/text-model" {
		t.Errorf("payload model = %q", payload.Model)
	}
	if len(payload.Messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(payload.Messages))
	}
	if payload.Messages[1].Content != "a cozy lighthouse at dusk" {
		t.Errorf("user content = %q", payload.Messages[1].Content)
	}
	if got != "a-descriptive-name" {
		t.Errorf("returned text = %q, want trimmed content", got)
	}
}

func TestGenerateTextErrorMapping(t *testing.T) {
	cases := []struct {
		name   string
		status int
		want   error
	}{
		{"auth", http.StatusUnauthorized, provider.ErrAuth},
		{"credit", http.StatusPaymentRequired, provider.ErrInsufficientCredit},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				w.Write([]byte(`{"error":{"message":"boom"}}`))
			}))
			defer srv.Close()
			_, err := c.GenerateText(context.Background(), provider.TextRequest{Model: "m", Prompt: "p"})
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

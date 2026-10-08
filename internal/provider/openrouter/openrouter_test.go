package openrouter

import (
	"context"
	"encoding/base64"
	"errors"
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
     "supported_parameters":{"aspect_ratio":{"type":"enum","values":["1:1","16:9"]},"input_references":{"type":"range","min":0},"seed":{"type":"range"}}},
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
	if !g.Capabilities.AcceptsReferenceImages {
		t.Error("gemini should accept reference images")
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
		ReferenceImages: []provider.ReferenceImage{{Data: []byte("x"), MediaType: "image/png"}},
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

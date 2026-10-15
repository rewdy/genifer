package openrouter

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/rewdy/genifer/internal/provider"
)

func TestGenerateReferenceConstraints(t *testing.T) {
	png := testPNG(t)
	for _, tc := range []struct {
		name, model, params string
		refs                []provider.ReferenceImage
		want                error
	}{
		{"missing", "recraft/recraft-v4-styles-vector", `{"input_references":{"type":"range","min":1,"max":10}}`, nil, provider.ErrInvalidReference},
		{"too many", "m", `{"input_references":{"type":"range","min":0,"max":1}}`, []provider.ReferenceImage{{Data: png}, {Data: png}}, provider.ErrInvalidReference},
		{"bad style", "recraft/recraft-v4-styles-vector", `{"input_references":{"type":"range","min":1,"max":10}}`, []provider.ReferenceImage{{Data: []byte("bad")}}, provider.ErrInvalidReference},
		{"valid style", "recraft/recraft-v4-styles-vector", `{"input_references":{"type":"range","min":1,"max":10}}`, []provider.ReferenceImage{{Data: png, MediaType: "image/png"}}, nil},
		{"ordinary text only", "recraft/recraft-v4-vector", `{"input_references":{"type":"range","min":0,"max":1}}`, nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			posts := 0
			c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/images/models" {
					w.Write([]byte(`{"data":[{"id":"` + tc.model + `","supported_parameters":` + tc.params + `}]}`))
					return
				}
				posts++
				if r.URL.Path != "/images" || r.Method != "POST" {
					t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
				}
				var body generatePayload
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body.N != 1 {
					t.Error(body.N)
				}
				if len(tc.refs) > 0 {
					if len(body.InputReferences) != len(tc.refs) || body.InputReferences[0].ImageURL.URL != "data:image/png;base64,"+base64.StdEncoding.EncodeToString(png) {
						t.Error("invalid reference payload")
					}
				}
				w.Write([]byte(`{"data":[{"b64_json":"aW1n","media_type":"image/svg+xml"}]}`))
			}))
			defer srv.Close()
			_, err := c.Generate(context.Background(), provider.GenerateRequest{Model: tc.model, Prompt: "bird", ReferenceImages: tc.refs})
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
			if (posts == 1) != (tc.want == nil) {
				t.Fatalf("posts %d", posts)
			}
		})
	}
}

func TestGenerateDiscoveryFailureDoesNotPost(t *testing.T) {
	posts := 0
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			posts++
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	_, err := c.Generate(context.Background(), provider.GenerateRequest{Model: "m", Prompt: "bird"})
	if !errors.Is(err, provider.ErrAuth) || posts != 0 {
		t.Fatalf("err %v posts %d", err, posts)
	}
}

func TestDataURLInfersMissingMediaType(t *testing.T) {
	data, err := os.ReadFile("../testdata/reference.webp")
	if err != nil {
		t.Fatal(err)
	}
	if got := dataURL(provider.ReferenceImage{Data: data}); !strings.HasPrefix(got, "data:image/webp;base64,") {
		t.Fatal("missing media type was not inferred from bytes")
	}
}

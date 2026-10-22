package tui

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rewdy/genifer/internal/config"
	"github.com/rewdy/genifer/internal/gen"
	"github.com/rewdy/genifer/internal/provider"
	"github.com/rewdy/genifer/internal/provider/openrouter"
)

func TestSVGGenerationSaveAndOpen(t *testing.T) {
	const model = "recraft/recraft-v4-styles-vector"
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256"><path fill="red" d="M0 0h256v256H0z"/></svg>`)
	var reference bytes.Buffer
	if err := png.Encode(&reference, image.NewRGBA(image.Rect(0, 0, 256, 256))); err != nil {
		t.Fatal(err)
	}
	posts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("missing authorization")
		}
		switch r.URL.Path {
		case "/images/models":
			w.Write([]byte(`{"data":[{"id":"` + model + `","supported_parameters":{"input_references":{"type":"range","min":1,"max":10},"output_format":{"type":"enum","values":["svg"]}}}]}`))
		case "/images":
			posts++
			if r.Method != "POST" || r.Header.Get("Content-Type") != "application/json" {
				t.Error("invalid method/content type")
			}
			var payload map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			if string(payload["n"]) != "1" || string(payload["model"]) != `"`+model+`"` {
				t.Error("invalid model/count")
			}
			if _, ok := payload["output_format"]; ok {
				t.Error("native output overridden")
			}
			if !strings.Contains(string(payload["input_references"]), "data:image/png;base64,") {
				t.Error("reference missing")
			}
			json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"b64_json": base64.StdEncoding.EncodeToString(svg), "media_type": "image/svg+xml"}}, "usage": map[string]any{"cost": 0.055}})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	p := openrouter.New(func(context.Context) (string, error) { return "test-key", nil }, openrouter.WithBaseURL(srv.URL), openrouter.WithHTTPClient(srv.Client()))
	out := gen.Run(context.Background(), p, gen.Draft{Model: model, Prompt: "red graphic", ReferenceImages: []provider.ReferenceImage{{Data: reference.Bytes(), MediaType: "image/png"}}}, t.TempDir(), gen.NameOptions{})
	if out.Failure != gen.FailureNone || out.CostUSD != 0.055 || posts != 1 {
		t.Fatalf("outcome %+v, posts %d", out, posts)
	}
	if filepath.Ext(out.Path) != ".svg" {
		t.Fatal(out.Path)
	}
	saved, err := os.ReadFile(out.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(saved, svg) {
		t.Fatal("SVG bytes changed")
	}
	for _, auto := range []bool{false, true} {
		t.Run(map[bool]string{false: "manual", true: "auto"}[auto], func(t *testing.T) {
			calls := 0
			m := New(Deps{Config: config.Config{AutoOpen: auto}, OpenCommand: "custom-viewer", Open: func(ctx context.Context, override, path string) error {
				if path != out.Path || override != "custom-viewer" || ctx.Err() != nil {
					t.Errorf("wrong open arguments: %q %q", override, path)
				}
				calls++
				return nil
			}})
			m.lastDraft = gen.Draft{Model: model, Prompt: "red graphic"}
			updated, cmd := m.Update(generatedMsg{outcome: out})
			m = updated.(Model)
			if cmd != nil {
				cmd()
			}
			if !auto {
				if calls != 0 {
					t.Fatal("opened without user action")
				}
				_, cmd = m.handleResultKey(key("o"))
				if cmd == nil {
					t.Fatal("no manual open command")
				}
				cmd()
			}
			if calls != 1 {
				t.Fatalf("opener called %d times", calls)
			}
		})
	}
}

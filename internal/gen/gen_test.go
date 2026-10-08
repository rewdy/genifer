package gen

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rewdy/genifer/internal/provider"
)

// --- 5.1 composition / review ---------------------------------------------

func TestDraftSubmitExactText(t *testing.T) {
	d := Draft{Prompt: "a lighthouse at dusk", Model: "m"}
	req, err := d.Request()
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	if req.Prompt != "a lighthouse at dusk" {
		t.Errorf("prompt = %q, want exact text", req.Prompt)
	}
	if d.Review() != "a lighthouse at dusk" {
		t.Errorf("Review = %q", d.Review())
	}
}

func TestDraftEditPreservesText(t *testing.T) {
	// Editing is represented by mutating the draft's Prompt; the value must be
	// preserved across a review round-trip.
	d := Draft{Prompt: "first", Model: "m"}
	if d.Review() != "first" {
		t.Fatal("review mismatch")
	}
	d.Prompt = "first, edited"
	if d.Review() != "first, edited" {
		t.Errorf("edited review = %q", d.Review())
	}
}

func TestDraftEmptyPromptBlocked(t *testing.T) {
	for _, p := range []string{"", "   ", "\n\t "} {
		if _, err := (Draft{Prompt: p, Model: "m"}).Request(); !errors.Is(err, ErrEmptyPrompt) {
			t.Errorf("Request(%q) err = %v, want ErrEmptyPrompt", p, err)
		}
	}
}

// --- 5.2 saving ------------------------------------------------------------

func TestSaveSuccess(t *testing.T) {
	dir := t.TempDir()
	path, err := Save(dir, provider.GenerateResult{Data: []byte("png"), MediaType: "image/png"})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if filepath.Ext(path) != ".png" {
		t.Errorf("ext = %q, want .png", filepath.Ext(path))
	}
	b, _ := os.ReadFile(path)
	if string(b) != "png" {
		t.Errorf("file contents = %q", b)
	}
}

func TestSaveCreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does", "not", "exist")
	_, err := Save(dir, provider.GenerateResult{Data: []byte("x"), MediaType: "image/webp"})
	if err != nil {
		t.Fatalf("Save should create dir: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("dir not created: %v", err)
	}
}

func TestSaveMediaTypeExtensions(t *testing.T) {
	cases := map[string]string{
		"image/png":     ".png",
		"image/jpeg":    ".jpg",
		"image/webp":    ".webp",
		"image/svg+xml": ".svg",
		"weird/type":    ".img",
	}
	for mt, want := range cases {
		if got := extByMediaType(mt); got != want {
			t.Errorf("ext(%q) = %q, want %q", mt, got, want)
		}
	}
}

func TestSaveNoCollision(t *testing.T) {
	dir := t.TempDir()
	// Freeze time so both saves compute the same base name.
	now = func() time.Time { return time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC) }
	defer func() { now = time.Now }()
	p1, _ := Save(dir, provider.GenerateResult{Data: []byte("a"), MediaType: "image/png"})
	p2, _ := Save(dir, provider.GenerateResult{Data: []byte("b"), MediaType: "image/png"})
	if p1 == p2 {
		t.Fatalf("expected distinct paths, both %q", p1)
	}
}

func TestSaveWriteFailure(t *testing.T) {
	// Point dir at a path whose parent is a file, so MkdirAll fails.
	f := filepath.Join(t.TempDir(), "afile")
	os.WriteFile(f, []byte("x"), 0o600)
	_, err := Save(filepath.Join(f, "sub"), provider.GenerateResult{Data: []byte("x"), MediaType: "image/png"})
	if err == nil {
		t.Fatal("expected error when output dir cannot be created")
	}
}

// --- 5.3 open behavior -----------------------------------------------------

func TestOpenCommandDefaults(t *testing.T) {
	cases := map[string]string{"darwin": "open", "linux": "xdg-open", "windows": "start"}
	for goos, want := range cases {
		if got := defaultOpenCommand(goos); got != want {
			t.Errorf("defaultOpenCommand(%q) = %q, want %q", goos, got, want)
		}
	}
}

func TestOpenCommandOverride(t *testing.T) {
	if got := OpenCommand("feh"); got != "feh" {
		t.Errorf("override ignored: got %q", got)
	}
	if OpenCommand("") == "" {
		t.Error("empty override should fall back to an OS default")
	}
}

// --- 5.4 async run + failure surfacing ------------------------------------

type fakeProvider struct {
	res provider.GenerateResult
	err error
}

func (f fakeProvider) Models(context.Context) ([]provider.Model, error) { return nil, nil }
func (f fakeProvider) Pricing(context.Context, string) (provider.Price, error) {
	return provider.Price{}, nil
}
func (f fakeProvider) Generate(ctx context.Context, _ provider.GenerateRequest) (provider.GenerateResult, error) {
	return f.res, f.err
}

func TestRunSuccess(t *testing.T) {
	dir := t.TempDir()
	p := fakeProvider{res: provider.GenerateResult{Data: []byte("img"), MediaType: "image/png"}}
	out := Run(context.Background(), p, Draft{Prompt: "cat", Model: "m"}, dir)
	if out.Failure != FailureNone {
		t.Fatalf("failure = %v (%v)", out.Failure, out.Err)
	}
	if out.Path == "" {
		t.Error("expected a saved path")
	}
}

func TestRunFailureClassification(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		err  error
		want FailureKind
	}{
		{provider.ErrNoAPIKey, FailureNoKey},
		{provider.ErrAuth, FailureAuth},
		{provider.ErrInsufficientCredit, FailureCredit},
		{provider.ErrGenerationFailed, FailureGeneration},
		{context.Canceled, FailureCancelled},
		{errors.New("boom"), FailureOther},
	}
	for _, tc := range cases {
		out := Run(context.Background(), fakeProvider{err: tc.err}, Draft{Prompt: "p", Model: "m"}, dir)
		if out.Failure != tc.want {
			t.Errorf("err %v -> failure %v, want %v", tc.err, out.Failure, tc.want)
		}
		if out.Message() == "" {
			t.Errorf("empty message for %v", tc.err)
		}
	}
}

func TestRunRetryable(t *testing.T) {
	dir := t.TempDir()
	gen := Run(context.Background(), fakeProvider{err: provider.ErrGenerationFailed}, Draft{Prompt: "p", Model: "m"}, dir)
	if !gen.Retryable() {
		t.Error("generation failure should be retryable")
	}
	credit := Run(context.Background(), fakeProvider{err: provider.ErrInsufficientCredit}, Draft{Prompt: "p", Model: "m"}, dir)
	if credit.Retryable() {
		t.Error("credit failure should not be retryable")
	}
}

func TestRunEmptyPrompt(t *testing.T) {
	out := Run(context.Background(), fakeProvider{}, Draft{Prompt: "", Model: "m"}, t.TempDir())
	if !errors.Is(out.Err, ErrEmptyPrompt) {
		t.Fatalf("err = %v, want ErrEmptyPrompt", out.Err)
	}
}

package gen

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rewdy/genifer/internal/provider"
)

// namingProvider is a fake provider with controllable text completion output.
type namingProvider struct {
	res     provider.GenerateResult
	genErr  error
	text    string
	textErr error
}

func (p namingProvider) Models(context.Context) ([]provider.Model, error) { return nil, nil }
func (p namingProvider) Pricing(context.Context, string) (provider.Price, error) {
	return provider.Price{}, nil
}
func (p namingProvider) Generate(ctx context.Context, _ provider.GenerateRequest) (provider.GenerateResult, error) {
	if err := ctx.Err(); err != nil {
		return provider.GenerateResult{}, err
	}
	return p.res, p.genErr
}
func (p namingProvider) GenerateText(context.Context, provider.TextRequest) (string, error) {
	return p.text, p.textErr
}

// 3.2 sanitizeBaseName
func TestSanitizeBaseName(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "a cozy lighthouse", "a cozy lighthouse"},
		{"spaces collapsed", "  foo   bar  ", "foo bar"},
		{"slashes removed", "a/b\\c", "abc"},
		{"unicode letters kept", "château fjord", "château fjord"},
		{"extension stripped", "sunset.png", "sunset"},
		{"overlong truncated", strings.Repeat("a", 200), strings.Repeat("a", maxBaseNameLen)},
		{"junk empty", "///... ,,," , ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sanitizeBaseName(tc.in); got != tc.want {
				t.Errorf("sanitizeBaseName(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// 3.3 Run honors NameOptions when generating a descriptive file name.
func TestRunSmartNaming(t *testing.T) {
	img := provider.GenerateResult{Data: []byte("img"), MediaType: "image/png"}
	timestampBase := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC).Format("2006-01-02-150405")
	now = func() time.Time { return time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC) }
	defer func() { now = time.Now }()

	cases := []struct {
		name     string
		p        namingProvider
		opts     NameOptions
		wantBase string // expected non-empty file name base, or "" for timestamp
	}{
		{
			name:     "enabled good name used",
			p:        namingProvider{res: img, text: "A cozy lighthouse / winter.png"},
			opts:     NameOptions{Enabled: true, Model: "text/m"},
			wantBase: "A cozy lighthouse winter",
		},
		{
			name:     "disabled uses timestamp",
			p:        namingProvider{res: img, text: "any"},
			opts:     NameOptions{Enabled: false, Model: "text/m"},
			wantBase: "",
		},
		{
			name:     "empty model uses timestamp",
			p:        namingProvider{res: img, text: "any"},
			opts:     NameOptions{Enabled: true},
			wantBase: "",
		},
		{
			name:     "naming error falls back to timestamp",
			p:        namingProvider{res: img, textErr: errors.New("naming failed")},
			opts:     NameOptions{Enabled: true, Model: "text/m"},
			wantBase: "",
		},
		{
			name:     "junk name falls back to timestamp",
			p:        namingProvider{res: img, text: "///..."},
			opts:     NameOptions{Enabled: true, Model: "text/m"},
			wantBase: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			out := Run(context.Background(), tc.p, Draft{Prompt: "p", Model: "img/m"}, dir, tc.opts)
			if out.Failure != FailureNone {
				t.Fatalf("failure = %v (%v)", out.Failure, out.Err)
			}
			if out.Path == "" {
				t.Fatal("expected a saved file")
			}
			if _, err := os.Stat(out.Path); err != nil {
				t.Fatalf("saved file missing: %v", err)
			}
			base := strings.TrimSuffix(filepath.Base(out.Path), ".png")
			if base != tc.wantBase {
				if tc.wantBase != "" {
					t.Errorf("file base = %q, want %q", base, tc.wantBase)
				} else if base != timestampBase {
					t.Errorf("file base = %q, want timestamp %q", base, timestampBase)
				}
			}
		})
	}
}
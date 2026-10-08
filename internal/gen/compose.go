package gen

import (
	"errors"
	"strings"

	"github.com/rewdy/genifer/internal/provider"
)

// ErrEmptyPrompt indicates a submit was attempted with no prompt text.
var ErrEmptyPrompt = errors.New("prompt is empty")

// Draft is a prompt being composed. It is the single source of truth that both
// freeform entry and (later) template filling feed into, then reviewed before
// sending.
type Draft struct {
	Prompt          string
	Model           string
	AspectRatio     string
	Seed            *int64
	ReferenceImages []provider.ReferenceImage
}

// Review returns the exact prompt text that will be submitted, for display at
// the review step. It does not mutate the draft.
func (d Draft) Review() string {
	return strings.TrimRight(d.Prompt, "\n")
}

// Request builds the provider request from the draft, rejecting an empty
// prompt so a blank generation is never sent.
func (d Draft) Request() (provider.GenerateRequest, error) {
	if strings.TrimSpace(d.Prompt) == "" {
		return provider.GenerateRequest{}, ErrEmptyPrompt
	}
	return provider.GenerateRequest{
		Model:           d.Model,
		Prompt:          d.Prompt,
		AspectRatio:     d.AspectRatio,
		Seed:            d.Seed,
		ReferenceImages: d.ReferenceImages,
	}, nil
}

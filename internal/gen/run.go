package gen

import (
	"context"
	"errors"

	"github.com/rewdy/genifer/internal/provider"
)

// FailureKind classifies a generation failure so the UI can show an actionable
// message and decide whether a retry is sensible.
type FailureKind int

const (
	FailureNone FailureKind = iota
	FailureCancelled
	FailureNoKey
	FailureAuth
	FailureCredit
	FailureGeneration // retryable
	FailureSave
	FailureOther
)

// Outcome is the result of a generation attempt.
type Outcome struct {
	Path    string      // saved file path on success
	CostUSD float64     // actual cost in USD on success (0 if unreported)
	Failure FailureKind // FailureNone on success
	Err     error       // underlying error when Failure != FailureNone
}

// Message returns a short, human-readable description of the outcome.
func (o Outcome) Message() string {
	switch o.Failure {
	case FailureNone:
		return "Saved " + o.Path
	case FailureCancelled:
		return "Generation cancelled"
	case FailureNoKey:
		return "No API key configured — set your OpenRouter API key (see `genifer config`)"
	case FailureAuth:
		return "Authentication failed — your API key was rejected"
	case FailureCredit:
		return "Insufficient account credit"
	case FailureGeneration:
		return "Generation failed — try again"
	case FailureSave:
		return "Could not save the image: " + errText(o.Err)
	default:
		return "Error: " + errText(o.Err)
	}
}

// Retryable reports whether retrying with the same prompt may succeed.
func (o Outcome) Retryable() bool {
	switch o.Failure {
	case FailureGeneration, FailureOther:
		return true
	default:
		return false
	}
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// Run generates one image for the draft and saves it to dir. It is safe to call
// off the UI loop; cancelling ctx aborts an in-flight request. Run never
// returns a bare error — every failure is classified in the Outcome.
func Run(ctx context.Context, p provider.Provider, d Draft, dir string) Outcome {
	req, err := d.Request()
	if err != nil {
		return Outcome{Failure: FailureOther, Err: err}
	}
	res, err := p.Generate(ctx, req)
	if err != nil {
		return Outcome{Failure: classify(ctx, err), Err: err}
	}
	path, err := Save(dir, res)
	if err != nil {
		return Outcome{Failure: FailureSave, Err: err}
	}
	return Outcome{Path: path, CostUSD: res.CostUSD}
}

func classify(ctx context.Context, err error) FailureKind {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(ctx.Err(), context.Canceled):
		return FailureCancelled
	case errors.Is(err, provider.ErrNoAPIKey):
		return FailureNoKey
	case errors.Is(err, provider.ErrAuth):
		return FailureAuth
	case errors.Is(err, provider.ErrInsufficientCredit):
		return FailureCredit
	case errors.Is(err, provider.ErrGenerationFailed):
		return FailureGeneration
	default:
		return FailureOther
	}
}

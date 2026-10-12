# Proposal

## Why

After a successful generation the result screen offers only "open", "new"
(which returns to compose but silently keeps the prompt and attachments), and
"quit". Users who want to iterate — try the same prompt on a different model,
start fresh, or re-run with identical parameters — have no direct path and must
manually navigate back and re-key values. The seed parameter is plumbed through
the backend but never set or shown, so a user cannot tell a seed is in use,
reproduce a result, or deliberately vary it.

## What Changes

- Replace the single ambiguous "new" action on the result screen with three
  explicit post-generation actions:
  - **Try again with a new model**: return to the model picker while preserving
    the entered prompt, aspect ratio, and reference images, so the next
    generation reuses them against a newly chosen model.
  - **New image**: return to the prompt screen with the prompt and reference
    images cleared, for an unrelated generation.
  - **Re-generate**: immediately re-run generation with the same model and
    parameters as the just-completed generation.
- Handle the **seed** for seed-capable models on the user's behalf: genifer
  auto-generates the seed (the user is never required to type one), shows the
  seed in use so the user knows one is active and what it is, and on re-generate
  lets the user reuse it (reproduce) or use a new one (vary) — communicating
  which is happening rather than changing it silently. Guiding principle: tell
  the user what they need to know to use the tool properly without asking for
  input they don't need to give.
- Keep the existing retry-on-failure behavior for failed generations distinct
  from these success-path actions.

## Capabilities

### New Capabilities
- `post-generation-actions`: the actions available on the result screen after a
  generation completes — try-again-with-new-model, new-image, and re-generate —
  and how each carries or clears the composed parameters.
- `seed-control`: auto-generating, displaying, and reusing-or-varying a seed for
  seed-capable models so a generation can be reproduced or deliberately varied.

### Modified Capabilities
<!-- None: no pre-existing specs. -->

## Impact

- `internal/tui/model.go`: `phaseResult` key handling (`handleResultKey`), the
  "new" transition, draft/seed state carried on `Model`, re-generate reuse/vary
  choice, draft reset logic.
- `internal/tui/view.go`: `resultView` (show the seed used) and `footerView` key
  hints for the new actions.
- `internal/gen/compose.go` + `internal/tui/model.go`: mint a seed for
  seed-capable models and set `Draft.Seed` (field already present but unset).
- `internal/provider/provider.go` + `openrouter`: optionally surface a
  provider-reported seed in `GenerateResult`; not relied upon, since genifer
  mints the seed itself.

# Design

## Context

See proposal.md — Why. The TUI is a single Bubble Tea `Model`
(`internal/tui/model.go`) driven by a `phase` iota: picker → compose → aspect →
review → generating → result. Compose state lives directly on `Model`
(`prompt textarea.Model`, `aspectRatio string`, `refImages []provider.ReferenceImage`,
`selected int`). `generate()` builds a `gen.Draft` via `composeDraft()` and runs
`gen.Run` in a `tea.Cmd`. `handleResultKey` currently maps `enter`/`n` back to
`phaseCompose` without clearing anything. `gen.Draft.Seed *int64` and
`provider.GenerateRequest.Seed` already exist but `composeDraft()` never sets a
seed, and `GenerateResult` does not carry a seed back.

## Goals / Non-Goals

**Goals:**
- Three unambiguous success-path actions on `phaseResult`, with explicit
  carry/clear semantics for prompt, aspect ratio, and reference images.
- Auto-minted seed for seed-capable models, shown to the user, with a
  reuse-or-vary choice on re-generate.
- Keep the existing failure/retry path untouched.

**Non-Goals:**
- Persisting prompts or seeds across sessions (that is the prompt-history
  change — which does persist the seed value for reproducibility).
- A manual seed-entry field. genifer mints the seed; the user is not asked to
  type one. (A manual override could be added later but is out of scope.)
- The larger prompt-centric UI rework (that is the prompt-ui-overhaul change).
  This change stays within the current phase model; the overhaul may later
  re-express these actions as slash commands.

## Decisions

- **Model each action as a `phaseResult` key handler, reusing existing phases.**
  Try-again-with-new-model sets `phase = phasePicker` without clearing compose
  state. New-image clears `prompt` and `refImages`, keeps `selected`, and sets
  `phase = phaseCompose`. Re-generate re-runs the generation directly from
  `phaseResult`. Alternative — a new dedicated post-result phase — rejected as
  unnecessary; reusing picker/compose keeps the state machine small.
- **genifer mints the seed; it does not depend on the provider echoing one.**
  For a seed-capable model (`Capabilities.SupportsSeed`), `composeDraft()` sets
  `Draft.Seed` to a locally generated random int64 when none is already set.
  This makes the seed always known to genifer, so it can be shown, reused, and
  (via the prompt-history change) persisted — none of which is reliable if we
  wait for the provider to report the seed it chose. Alternative — rely on
  `GenerateResult` reporting the seed — rejected as unreliable; we still accept
  a provider-reported seed opportunistically but never require it.
- **Capture the composed draft at generate time.** Store the exact `gen.Draft`
  used for the in-flight generation on `Model` (e.g. `lastDraft gen.Draft`) when
  the generation fires, including the minted seed. Re-generate reuses
  `lastDraft`; new-image/try-again mutate the live fields, so a captured draft
  is what guarantees "same parameters".
- **Re-generate offers reuse vs vary via the seed.** For a seed-capable model,
  re-generate presents a reuse-seed (reproduce) or new-seed (vary) choice and
  indicates which was applied; reuse sends `lastDraft.Seed`, vary mints a fresh
  seed and shows it. For a model without seed support, re-generate just repeats
  `lastDraft`. Keep the choice lightweight (a key, not a typed value) per the
  guiding principle.
- **Show the seed in the result view.** `resultView` displays the seed used for
  seed-capable models (alongside the existing cost line). `gen.Outcome` carries
  the seed that was used (from the minted `Draft.Seed`, or a provider-reported
  value if present) so the view and history both read one source.

## Risks / Trade-offs

- [Provider ignores or overrides the sent seed] → If a model does not honor the
  seed we send, reproduction is best-effort. genifer still shows and stores the
  seed it sent; this is the honest contract and is noted in the seed-control
  spec. Minting locally is still strictly better than relying on an unreported
  provider seed.
- [Reference images + new model that rejects them] → Guarded already by
  `ErrReferenceImagesUnsupported`; try-again-with-new-model must drop
  now-invalid reference images and inform the user rather than erroring at
  generate time.
- [Scope creep into the UI overhaul] → Keep these as result-screen keys within
  the current phases; do not introduce slash commands here.

## Open Questions

- None blocking. Exact key bindings for the three actions and the
  reuse/vary choice are an implementation detail chosen during apply to avoid
  collisions with open/quit.

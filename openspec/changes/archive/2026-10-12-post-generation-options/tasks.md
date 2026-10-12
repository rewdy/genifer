# Tasks

## 1. Seed minting and propagation

- [x] 1.1 In `composeDraft()` (or an equivalent helper), mint a random `int64` seed and set `Draft.Seed` for seed-capable models when no seed is already set, leaving non-seed models untouched; verify a table-driven test asserts a seed is set only for seed-capable models.
- [x] 1.2 Carry the seed that was used forward on `gen.Outcome` (success path) in `internal/gen/run.go`, sourced from the minted `Draft.Seed` (and preferring a provider-reported value if one is present); verify `gen_test.go` asserts the outcome exposes the seed used.
- [x] 1.3 Optionally parse a provider-reported seed into `provider.GenerateResult.Seed` in the OpenRouter path without relying on it; verify a unit test over response fixtures with and without a seed field.

## 2. Show the seed

- [x] 2.1 Update `resultView` to display the seed used for seed-capable models (alongside the cost line) and show nothing for non-seed models; verify a view test asserts the seed line appears only when a seed was used.

## 3. Post-generation actions

- [x] 3.1 Capture the exact draft used for the in-flight generation (`lastDraft`, including the minted seed) when the generation fires; verify a test asserts `lastDraft` equals the submitted draft.
- [x] 3.2 Implement re-generate with a reuse-vs-vary seed choice for seed-capable models (reuse sends `lastDraft.Seed`; vary mints a fresh seed and shows it), repeating `lastDraft` for non-seed models, and indicate which was applied; verify tests for the reuse path (same seed) and the vary path (new seed shown).
- [x] 3.3 Implement try-again-with-new-model: set `phasePicker` preserving prompt, aspect ratio, and reference images; drop reference images and inform the user when the newly selected model rejects them; verify tests for both the preserved-values path and the dropped-images path.
- [x] 3.4 Implement new-image: clear prompt and reference images, keep the selected model, return to `phaseCompose` (the next generation mints a fresh seed); verify a test asserts prompt and reference images are cleared and the model is retained.
- [x] 3.5 Update `resultView`/`footerView` to show key hints for open, try-again, new-image, re-generate (incl. the reuse/vary affordance), and quit on success, leaving the failure/retry path unchanged; verify a view/footer test asserts the success-path hints and that failure hints are unchanged.

## 4. Documentation and integration

- [x] 4.1 Update `README.md` (and `docs/` as needed) to describe the result-screen actions and that genifer auto-generates and shows a seed for seed-capable models; verify the documented keys match `footerView`.
- [x] 4.2 Run `go build ./...`, `go test ./...`, and `go vet ./...` (or `just check`) and verify all pass.

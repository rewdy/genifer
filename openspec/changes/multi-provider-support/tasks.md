# Tasks

## 1. Config shape: provider list

- [x] 1.1 Replace `Config.Provider string` + `OpenRouterConfig` with `Providers []ProviderConfig` (flat union: `Key`, `Type`, `APIKey Value`, `BaseURL string`) in `internal/config/config.go`; update `Default()` to a single OpenRouter instance. Verify with a `config_test.go` case decoding a two-instance `providers:` YAML into the expected structs.
- [x] 1.2 Add validation that provider `key`s are unique and non-empty and `type` is non-empty; return a clear error naming the offending instance otherwise. Verify with table-driven tests for duplicate key, empty key, and empty type.
- [x] 1.3 Update `internal/config/doc.go` package comment to describe the provider-list shape.

## 2. Legacy config detection and recovery

- [x] 2.1 In `Load`, detect the legacy shape (top-level `provider:` and/or a type-named block with no `providers:` list) before strict decode, and return a distinct sentinel (e.g. `ErrLegacyConfig`) instead of a generic parse error. Verify with a test feeding a legacy `config.yaml` and asserting the sentinel.
- [x] 2.2 Add a backup helper that renames `config.yaml` → `config.yaml.bak`, choosing a non-colliding name when `.bak` already exists; never overwrite an existing backup. Verify with tests for the fresh-backup and existing-backup cases.
- [x] 2.3 Wire `main.go` so an `ErrLegacyConfig` result backs up the file and enters the first-run flow (set the onboarding/`FirstRun` path) rather than erroring out. Verify by unit-testing the load-outcome branch (missing → default+FirstRun, legacy → backup+FirstRun, malformed → hard error, valid → load).

## 3. Starter config in list shape

- [x] 3.1 Rewrite `starterTemplate` / `StarterContent` in `internal/config/starter.go` to emit the `providers:` list shape with one instance, parameterized by provider type and its detail (OpenRouter api_key directive, or a1111 base_url). Verify the existing starter round-trip test (write → `Load`) passes against the new template for both an OpenRouter and an a1111 starter.
- [x] 3.2 Ensure `WriteStarter` keeps its create-only (O_EXCL) guarantee for the normal path; the legacy-recovery replacement happens only after task 2.2's backup. Verify with a test that `WriteStarter` still refuses to overwrite an existing file.

## 4. State and pricing cache re-keying

- [x] 4.1 Add `LastProviderKey` to `config.State` alongside `LastModel`; keep read-modify-write preservation of other fields. Verify with a `state_test.go` case that persisting the model keeps the aspect ratio and now also records the provider key.
- [x] 4.2 Re-key the pricing cache (`internal/config/pricing.go`, `pricing-cache.json`) by `providerKey/modelID` and bump its version; an old-version or unparsable cache is ignored and re-fetched. Verify with tests for composite-key round-trip and old-version-ignored.

## 5. Provider registry

- [x] 5.1 Replace `main.buildProvider` with `buildProviders(cfg) (map[string]provider.Provider, error)` plus the ordered instance list, dispatching on `.Type`; unknown type → clear error naming the instance. Verify with a test building a two-instance config and asserting the registry contents and the unknown-type error. (`provider.go` is unchanged.)
- [x] 5.2 Update `tui.Deps` to carry the registry (keyed map + ordered keys) instead of a single `Provider`, and update `main.run` to populate it. Verify `go build ./...` succeeds.

## 6. Local A1111-compatible provider

- [x] 6.1 Create `internal/provider/a1111` with `doc.go` and a `Client` implementing `provider.Provider`; add a connection-refused/unreachable sentinel to `provider.go` (distinct from `ErrAuth`) and map transport failures to it. Verify with a unit test asserting the unreachable mapping.
- [x] 6.2 Implement `Models` against `GET /sdapi/v1/sd-models` (id + name) honoring ctx. Verify with an httptest server returning a sample model list and a cancellation test.
- [x] 6.3 Implement `Pricing` returning `PriceFree` for any model, and hardcoded `Capabilities` (seed true, fixed aspect-ratio list, reference images false). Verify with a unit test on both.
- [x] 6.4 Implement `Generate` against `POST /sdapi/v1/txt2img`: map aspect ratio → width/height via a fixed table, decode the base64 `images[0]`, honor ctx, and reject reference images with a clear error before any call. Verify with httptest success, aspect-ratio→dimensions, reference-image-rejected, and cancellation tests.

## 7. TUI merge, pricing, and generation routing

- [x] 7.1 Change startup model loading to fan out `Models` across all registry instances concurrently with a bounded dial timeout, tag each model with its provider key, merge, and record a per-instance offline marker on failure. Verify with a test using a reachable fake and an unreachable fake, asserting the reachable models list plus the offline marker and no app error.
- [x] 7.2 Re-key `m.prices` by `providerKey/modelID`, update `rebuildItems`/pricing fetches to the composite key, and fetch per instance (skip pricing for a1111 or let it resolve free). Verify with a test that two providers sharing a model id show distinct prices.
- [x] 7.3 Make selection carry `(providerKey, modelID)`: update `currentModelID`/selection helpers, `preselectModel` to match on key+id, and persist both to state on choose. Verify with a selection/preselect test over a merged two-provider list.
- [x] 7.4 Route `generate()` to the owning provider resolved from the registry by the selected model's key, passing that single `provider.Provider` into `gen.Run` (gen contract unchanged). Verify with a test asserting the correct provider instance is used for a selected model.
- [x] 7.5 Update `delegate.go` to render the provider key as a group/tag on each picker row, visually secondary to the name. Verify with a render test asserting the key appears on rows.

## 8. Onboarding: provider-type choice and legacy entry

- [x] 8.1 Add a provider-type selection step (OpenRouter | local) ahead of the existing detail step in `internal/tui/onboarding.go`; for OpenRouter keep the three key methods, for local collect the WebUI address with a default. Verify with onboarding tests for each branch asserting the written instance.
- [x] 8.2 Make `writeOnboardConfig` emit a single provider-list instance via the updated starter (task 3.1) for the chosen type. Verify the written `config.yaml` loads back into the expected `Providers` entry.
- [x] 8.3 Confirm the legacy-recovery path (task 2.3) enters this flow and, on completion, writes the rebuilt config and proceeds to the picker. Verify with a test driving legacy-detected → onboarding → picker.

## 9. Documentation

- [x] 9.1 Rewrite `docs/config.md` for the provider-list shape: `providers:` with `key`/`type`/type-specific fields, an a1111 example, how to add multiple providers by hand, and the legacy `.bak` recovery behavior. Verify any example config in the doc loads via `config.Load` (or is covered by a doc-example test).
- [x] 9.2 Add a short `docs/local-provider.md` (or a section) covering the A1111-compatible contract, supported runtimes (A1111/Forge/ComfyUI-with-shim), and the `base_url` default. Verify links/paths referenced exist.
- [x] 9.3 Update `AGENTS.md` layout/architecture notes to mention the provider registry and the `a1111` provider. Verify the described packages match the tree.

## 10. Integration verification

- [x] 10.1 Run `go build ./...`, `go test ./...`, and `go vet ./...` (or `just check`) and confirm all pass.
- [ ] 10.2 Manually verify end-to-end: a two-provider config (OpenRouter + a1111) shows a merged picker; stopping the local WebUI shows it offline while OpenRouter models remain usable; a legacy config is backed up and routed into onboarding.

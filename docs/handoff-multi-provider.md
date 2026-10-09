# Handoff: multi-provider-support

Status at handoff: **28/29 tasks done.** Only the manual end-to-end check
(task 10.2) remains. Branch: `alt-providers`.

## Why this handoff exists

Task 10.2 needs a **running local A1111-compatible WebUI**. The local models
live on another machine, so the end-to-end verification continues there.

## What's implemented and verified

All automated checks pass (`just check`: build + test + vet, plus `gofmt`).

- **Config** (`internal/config`): `providers:` list of keyed, typed instances
  (flat union `ProviderConfig`); unique-key/non-empty-type validation;
  `ErrLegacyConfig` detection + `BackupConfig` (non-colliding `.bak`); starter
  rewritten to the list shape via `StarterSpec`; `State.LastProviderKey`;
  pricing cache re-keyed `providerKey/modelID` (version bumped to 2).
- **Provider** (`internal/provider`): added `ErrUnreachable`; new
  `internal/provider/a1111` client (`/sdapi/v1/sd-models`, `/sdapi/v1/txt2img`,
  aspect→dimension map, free pricing, fixed caps, reference-image rejection,
  transport→unreachable mapping).
- **Registry + TUI**: `main.buildProviders` (keyed map + ordered keys) and
  `loadConfig` (missing/legacy/malformed/valid); `tui.Deps` carries the
  registry; model identity is `(providerKey, modelID)` across picker, pricing,
  state, and generation routing; concurrent startup fan-out with offline
  isolation; onboarding gained a provider-type step (OpenRouter | local).
- **Docs**: `docs/config.md` rewritten, `docs/local-provider.md` added,
  `AGENTS.md` updated.

## The one remaining task — 10.2 (manual E2E)

Do this on the machine that can reach the local WebUI.

1. **Start the local WebUI** (A1111 or Forge launched with `--api`, or ComfyUI
   with the A1111-compatible shim). Note its address, e.g.
   `http://127.0.0.1:7860`. See `docs/local-provider.md`.
2. **Write a two-provider `config.yaml`** at `~/.config/genifer/config.yaml`:
   ```yaml
   providers:
     - key: or
       type: openrouter
       api_key: "{env:OPENROUTER_API_KEY}"
     - key: local
       type: a1111
       base_url: "http://127.0.0.1:7860"
   ```
   Export a real `OPENROUTER_API_KEY`.
3. **Merged picker**: launch `go run .` — the picker should list OpenRouter
   models tagged `@or` and local models tagged `@local`.
4. **Offline isolation**: stop the local WebUI and relaunch — `local` should
   show offline in the status line, its models omitted, while `@or` models stay
   usable. Restart the WebUI and confirm the local models return.
5. **Generate locally**: select a `@local` model and generate; confirm the
   image is produced by the local backend and saved.
6. **Legacy recovery**: put an old-shape config in place and launch:
   ```yaml
   provider: openrouter
   openrouter:
     api_key: "{env:OPENROUTER_API_KEY}"
   ```
   Confirm genifer renames it to `config.yaml.bak` and enters onboarding; walk
   the local-provider branch and confirm the rebuilt config lands in the picker.

Each of these behaviors is already covered by automated tests
(`TestLoadModelsIsolatesUnreachable`, `TestPricesDistinctAcrossProviders`,
`TestSelectionAcrossProviders`, `TestLoadConfigOutcomes/legacy`,
`TestOnboardingAdvancesToPicker`); step through the manual run to confirm the
live integration, then mark 10.2 done.

## Resuming on the other machine

```sh
git fetch origin
git checkout alt-providers
git pull
just check            # confirm the gate still passes
```

Then run the OpenSpec apply flow to pick up where this left off:

```sh
openspec instructions apply --change multi-provider-support --json
```

Only task 10.2 should remain. After verifying it, mark it done in
`openspec/changes/multi-provider-support/tasks.md`, then archive the change
(`/opsx-archive` or `openspec archive`).

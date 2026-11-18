# Design

## Context

See `proposal.md` for motivation. The current shape of the code constrains the approach:

- `provider.Capabilities` already carries flags such as `SupportsTransparentBackground` and a raw `OutputFormats` list; the OpenRouter adapter fills both from `GET /images/models` `supported_parameters` in `capsFrom`. No extra discovery call is needed.
- `provider.GenerateRequest` already has `Background` and `OutputFormat`, `gen.Draft` carries both, and `openrouter.generatePayload` already serializes `output_format`. The transparency guard in `Generate` drops unsupported requests at the provider seam.
- The TUI keeps a single `transparentBackground` boolean plus a `ctrl+t` toggle, shows the setting in the compose line, persists it as `State.LastTransparentBackground`, and gates it on `currentCaps()`.
- History records prompt/model/aspect/seed/path/time in a version-1, append-only `history-cache.json`. `reuseHistoryEntry` restores aspect and seed where supported; inline Up/Down recall snapshots only the prompt text.
- The picker is a bubbles v1 `list.Model`. It filters by a `FilterValue` of model name plus ID, sorts favorites first, and draws capability hints via `capHint`. Its `Filter` field is a `func(string, []string) []Rank`, which can be extended without replacing the list widget.
- `gen.extByMediaType` maps known media types to extensions but has no `image/gif` case. The preview package already imports the GIF decoder, so a GIF can be displayed as a still image.
- OpenRouter documents `output_format: "gif"` (advertised in discovery for models such as the GPT Image 2.6 variants), optional `frame_count`/`frame_duration_ms`, and that transparent backgrounds require PNG or WebP output.

This change assumes the in-progress `enable-transparent-images` change has landed; it extends that capability rather than duplicating it.

## Goals / Non-Goals

**Goals:**

- Reuse the existing capability/request/UI pipeline for GIF output instead of creating a parallel animation path.
- Keep discovery driven by advertised capability, with no model-ID allowlist.
- Preserve existing history files and add settings capture in a backward-compatible way.
- Make capability filters compose cleanly with the existing text filter and favorites ordering.

**Non-Goals:**

- No controls for `frame_count` or `frame_duration_ms`; GIF animation length and timing stay at provider defaults.
- No animated playback in the terminal preview; the first frame is sufficient.
- No new `config.yaml` option or `state.json` field for a last-used GIF default.
- No image conversion, background removal, or support for providers other than OpenRouter.
- No change to the history store version or pruning policy.

## Decisions

1. **Derive GIF support from advertised output formats.** Add `SupportsAnimatedGIF bool` to `provider.Capabilities`, set it in `capsFrom` when the `output_format` enum contains `gif`, and leave it false otherwise. Alternative considered: a separate `frame_count`/`animation` descriptor in discovery. Rejected because current discovery does not advertise those keys for GIF-capable models, while `output_format` reliably does. This also avoids any hand-maintained model list.

2. **Reuse `OutputFormat` as the request field.** Do not add an `Animated` boolean to `provider.GenerateRequest`. When the TUI's GIF toggle is on and the model supports it, `buildDraft` sets `OutputFormat = "gif"`; otherwise it stays empty. The provider guard drops `gif` for models that do not advertise it (mirroring the existing transparency guard) and omits `background: transparent` when GIF output is requested, because transparency requires PNG/WebP. Alternative considered: a separate `Animated` request field. Rejected as redundant state that could disagree with `OutputFormat`.

3. **Hide unsupported compose controls.** Extend `FormControls` with `ShowAnimatedGIF` derived from the capability, add a `ctrl+g` toggle and an `Animated: on/off` entry in the compose settings line, and add a `gif` marker to `capHint`. Keep the mutual-exclusion rule in the key handler and re-assert it when building the draft. Alternative: always show a disabled toggle. Rejected because the existing form hides any parameter the selected model does not support.

4. **Apply capability filters before models reach the list.** Keep active filters as a small struct of booleans on `Model`, and have the picker's item-building step skip models that fail any active filter before calling `list.SetItems`. The typed text query continues to run through the list's native filter over the reduced item set, and favorites-first ordering is applied before filtering. Toggle keys 1-5 select transparency, GIF, references, seed, and SVG; key 0 clears all; the status area names the active filters and result count. The selected model ID is recorded before rebuilding and re-selected if it survives. Alternative considered: a custom `list.Filter` closure reading capability tokens from `FilterValue`. Rejected after inspecting bubbles v1: `filterItems` short-circuits when the filter input is empty, so capability-only filters would never run unless the text input were forced non-empty, which would corrupt the existing text-filter UX.

5. **Extend history records without invalidating them.** Add optional `background` and `output_format` fields to `config.HistoryEntry` and keep `historyCacheVersion` at 1, so existing files load unchanged and missing values mean "use the model default". Record them from the same `lastDraft` used for the generation. Introduce one shared restore helper used by both `reuseHistoryEntry` and `applyRecalled`: it applies aspect ratio, seed, transparency, and GIF output only where `currentCaps()` allows, resolves GIF/transparent conflicts in favor of GIF, and updates the compose state. Extend the recall snapshot to hold the in-progress settings alongside the prompt text so stepping past the newest entry restores both. Alternative: bump the cache version to 2 and discard old records. Rejected because history is explicitly unbounded and never pruned.

6. **Save GIFs by media type.** Add `image/gif` to `extByMediaType` so a GIF response is saved as `.gif`; all other media types keep their current mapping and the `.img` fallback. No special preview path is needed because the GIF decoder is already registered.

## Risks / Trade-offs

- **Discovery may omit GIF support for a capable model** → Rely on advertised capabilities only; no allowlist. A future config override can be added if real-world gaps appear, without changing this design.
- **GIF size and frame timing are provider-controlled** → Document that `frame_count`/`frame_duration_ms` are not exposed; consider a follow-up change if users need control.
- **Filter closure holds pointer state across value copies** → Keep the filter state in one allocation created in `New`, and cover filter combinations, ordering, and selection in unit tests.
- **Mutual exclusion can be violated by stale or hand-edited history** → Enforce it both when restoring settings and in the provider request guard.
- **Large GIFs increase memory use in the existing decode/save path** → No new buffering is introduced; the current base64 decode already holds the full image in memory.

## Migration Plan

Land this change after `enable-transparent-images`. No user-visible migration is required: history files remain readable, no config keys are added, and defaults are unchanged for models without GIF support. Rollback is a normal revert; records written with the new optional fields remain readable by the previous version because unknown JSON fields are ignored.

## Open Questions

None.

# Proposal

## Why

OpenRouter's image API can request animated GIF output (`output_format: "gif"`)
from a small set of animation-capable models (for example the GPT Image 2.6
variants), but genifer currently has no way to detect that capability or ask for
it. Model discovery does advertise it (`output_format` includes `gif`), so the
app can surface it without guesswork. Separately, the picker only supports text
search, making capability-bearing models hard to find, and history records
prompts but not the settings that produced them, so recalling an old prompt
loses choices like transparency and aspect ratio.

## What Changes

- Detect animated GIF support from OpenRouter model discovery and expose it as a
  provider-agnostic `SupportsAnimatedGIF` capability, alongside the existing
  transparency capability. Never send `output_format: "gif"` to a model that
  does not advertise GIF output.
- Add an "Animated GIF" toggle to the compose form, shown only for
  animation-capable models. GIF output and transparent backgrounds are mutually
  exclusive (OpenRouter requires PNG or WebP for transparency), so enabling one
  clears the other in the UI and in the provider guard. GIF responses are saved
  with a `.gif` extension; the current extension map would otherwise fall back
  to `.img`.
- Add a capability filter panel to the model picker with toggleable filters for
  transparency, animated GIF, reference images (img2img), seed support, and
  vector/SVG output. Filters combine with the existing text query, compose with
  favorites-first ordering, can be cleared in one action, and are discoverable
  from the picker's hints and rows.
- Record the generation settings in history alongside the prompt: aspect ratio,
  seed, background (transparency), and output format (GIF). Restore them when
  selecting an entry or using inline Up/Down recall, applying each setting only
  where the currently selected model supports it. Older entries without these
  fields remain readable and fall back to the current defaults.
- Do not add a new config or `state.json` setting for the animated GIF default;
  GIF starts off unless a recalled history entry turns it on. Persisting a
  last-used GIF preference is a possible follow-up, not part of this change.

## Capabilities

### New Capabilities
- `animated-gif`: end-to-end animated GIF generation, from capability discovery
  through the compose option, guarded provider request, and correctly named
  saved file.
- `model-filter`: capability-based filtering of the model picker (transparency,
  animated GIF, reference images, seed, vector/SVG) that composes with text
  search and favorites ordering.

### Modified Capabilities
- `prompt-history`: history records capture generation settings and reuse/recall
  restore them where the selected model supports them.

## Impact

- `internal/provider/provider.go`, `internal/provider/openrouter/`: new
  capability flag, discovery parsing, and a guard that drops unsupported GIF
  output format requests before the API call.
- `internal/tui/`: modelItem filter values and picker filter state/UI, compose
  form controls and settings line, capability hints, key hints, and state
  restoration for history/recall.
- `internal/config/history.go`: optional history fields for background and
  output format; keep the on-disk version compatible so existing history is
  preserved.
- `internal/gen/save.go`: map `image/gif` to `.gif`.
- `docs/openrouter.md`: document GIF support, discovery, and the transparency
  exclusion.
- Depends on the in-progress `enable-transparent-images` change being landed
  first; it introduced the transparency capability and background request field
  that this change builds on. No new dependencies.

# Design

## Context

See proposal.md for motivation. Research performed 2026-10-14, without issuing paid generation requests.

Observed in the repository:
- `openrouter.Models` uses the dedicated image catalog, but `supportedParameter` keeps only `type` and enum `values`. `capsFrom` collapses references into a boolean.
- `Generate` checks capabilities only if references are present, so a missing required reference escapes validation. Its payload already serializes references as data URLs and requests `n=1`.
- The response decoder already preserves arbitrary base64 bytes and reported media type. `gen.Save` already maps `image/svg+xml` to `.svg`; the existing test covers extension selection, not an entire SVG generation/save round trip.
- Reference loading accepts PNG/JPEG/GIF, lacks a WebP decoder, and discards dimensions. Compose loads `@path` files synchronously inside `Update`; this conflicts with the repo's I/O-in-commands convention. Reference work in this change must move that path processing to a command rather than extend that pattern.
- The form describes every image-capable model as `img2img`. That does not communicate required style inputs. The TUI spec's clipboard-unavailable scenario currently promises generation without references; the delta restricts that promise to optional-reference models.

### Verified API contract

Use these dedicated Image API endpoints, not chat completions or the similarly named general Models API:

| Operation | Path |
| --- | --- |
| Discover | `GET /api/v1/images/models` |
| Inspect an endpoint | `GET /api/v1/images/models/{author}/{slug}/endpoints` |
| Generate | `POST /api/v1/images` |

The [Image API documentation](https://openrouter.ai/docs/guides/overview/multimodal/image-generation) describes base64 output with the actual media type, including SVG. Its model-level capability map is a union; endpoint records are definitive.

The [Styles Vector endpoint record](https://openrouter.ai/api/v1/images/models/recraft/recraft-v4-styles-vector/endpoints) currently has one Recraft endpoint: `input_references` range 1–10, `n` range 1–6, `output_format` enum `["svg"]`, no normalized seed capability, and aspect ratios including `1:1`. Thus the screenshot's concrete provider rejection is the reference count, not evidence that `1:1` or `n=1` is unsupported. It also lists optional passthrough controls; none is required for the initial workflow.

The [Styles Vector model page](https://openrouter.ai/recraft/recraft-v4-styles-vector) documents PNG/JPEG/WebP style inputs of at least 256 pixels on the shortest edge and an extra style-creation fee per request. Ordinary [V4 Vector](https://openrouter.ai/recraft/recraft-v4-vector) supports text-only generation; do not apply Styles requirements to all vector models.

Example JSON body (the reference value below is illustrative, not valid image data):

```json
{
  "model": "recraft/recraft-v4-styles-vector",
  "prompt": "A flat vector illustration of a red bird",
  "n": 1,
  "aspect_ratio": "1:1",
  "input_references": [
    {"type": "image_url", "image_url": {"url": "data:image/png;base64,<encoded-png-bytes>"}}
  ]
}
```

Send Bearer authentication and JSON content type using existing wiring. Leave `output_format` omitted to use the model's native default. A successful SVG response uses `data[0].b64_json` and `data[0].media_type = "image/svg+xml"`; existing cost handling remains applicable.

## Goals / Non-Goals

**Goals:** Extend the capability contract without coupling the TUI to Recraft model IDs; reuse the existing transport, draft, save, and opener flow; reject correctable input problems before a generation POST.

**Non-Goals:** Rendering SVG inside the terminal, accepting SVG as a style reference, converting raster to vector, adding a direct Recraft client, endpoint pinning, batching, advanced palettes/style IDs, output-format selection, or redesigning pricing and its cache.

## Decisions

### 1. Extend capabilities with optional constraints

Add provider-neutral optional minimum/maximum reference counts and advertised output formats. Retain `AcceptsReferenceImages` for compatibility with current callers and test fakes. Use optional numeric values so unknown maximum differs from zero. Parse range `min`/`max` in OpenRouter metadata. A reference descriptor without numeric bounds remains supported and optional; do not invent limits. Empty output formats means unknown, not raster.

Use discovered bounds for every model, not a vector-name check. Output guidance identifies SVG only when it is the sole advertised format; multiple formats must not be described as a guaranteed SVG result. A selector would require draft/history/state changes with no benefit for SVG-only models, so defer it.

### 2. Keep documented Styles rules inside OpenRouter

The catalog does not expose accepted reference media types or minimum dimensions. Enrich the four exact known Recraft V4 Styles IDs (Styles, Styles Pro, Styles Vector, Styles Pro Vector) with provider-neutral reference constraints: allowed media types and minimum width/height. Isolate this small, documented compatibility table in the adapter; no broad substring matching and no model-specific knowledge in `tui` or `gen`.

Put shared, pure validation of reference bytes/count against capabilities in `provider`, using header decoding rather than full raster decoding. Register WebP through existing `golang.org/x/image/webp`; move decoder ownership out of the TUI if needed. Validate actual bytes, not a filename or untrusted MIME label. Keep the existing 20 MiB input cap and retain GIF support for other models. No resizing, conversion, or silent attachment dropping.

Reusing just the boolean leaves the reported failure intact. Applying the Styles restrictions globally would break other models. Parsing prose descriptions would be brittle and is rejected.

### 3. Validate both the draft and provider boundary

The TUI uses known capabilities for immediate feedback. The adapter resolves capabilities for every Generate call, including zero-reference calls, then validates before POST. Initially reuse `capabilitiesFor`; its additional catalog GET for text-only calls is acceptable for correctness and keeps discovery failure/cancellation explicit. Do not add an unrelated cache. No generation is submitted if this lookup fails.

Reuse `ErrReferenceImagesUnsupported` for unsupported inputs and add one wrapped invalid-reference sentinel with useful count/format/dimension detail. Do not turn invalid inputs into `ErrGenerationFailed`. Handle validation errors as correctable input failures, with an edit path and without encouraging unchanged retries. Preserve auth/credit/cancellation mappings and thread context through any added work.

Guard compose submission, review confirmation, retry, and re-generate through a common pure validation path. After model changes, preserve references for capable models and block invalid drafts until the user removes/replaces them; retain existing dropping behavior for models accepting no references.

### 4. Make attachment updates atomic and asynchronous

Resolve and validate `@path` batches in a `tea.Cmd`; commit attachments and remove path lines only on a successful batch. On failure preserve the original draft and all prior attachments. Combine existing pasted attachments with candidate path images when checking the maximum. A successful path batch followed by an empty-prompt warning must not cause duplicate attachment on a later submit.

Carry a draft/model revision with asynchronous attachment messages; discard obsolete results after edits, model changes, or navigation. Enforce maximum and per-image rules on paste completion too. Minimum counts block submission, not intermediate attachment/removal steps. Show current count, limits, and actionable messages; distinguish a required reference from optional image editing. Compose and review show `SVG` for a sole SVG output format. Keep layout bounded at narrow terminal widths.

### 5. Preserve native output and existing billing semantics

Add a mocked HTTP-to-save regression proving SVG byte identity, `.svg` extension, actual cost, and reuse of the existing opener path. Do not add a renderer or request raster output. A missing media type keeps the existing `.img` fallback rather than guessing from model ID.

The picker currently means output-image price, not total request price. Keep that contract and cache unchanged; document the Styles surcharge and show the API's actual total cost after success. A later pricing change can model request-level fees explicitly.

## Risks / Trade-offs

- [Model metadata can change] → fixtures capture research-time bounds; consume live discovery for counts/formats and keep the small Styles enrichment table documented.
- [Model unions can combine incompatible endpoint constraints] → this release uses existing routing and validates catalog-level requirements; the target Styles Vector currently has one endpoint. Do not claim catalog validation guarantees eligibility across arbitrary multiple endpoints.
- [Extra discovery call adds latency/failure opportunity] → perform it cancellably off the UI loop; do not submit potentially invalid requests on lookup failure.
- [WebP decoding changes module ownership] → use the already present module, promote to a direct dependency if necessary, and preserve unrelated image-format behavior.
- [No real SVG generation was run during planning] → mock the documented contract; an optional paid smoke test during implementation must explicitly record its model, input, saved MIME/extension, and cost.

## Migration Plan

No config, history, state, or pricing-cache schema migration. Implement provider metadata/validation first, then command-based attachment handling and form guidance, then regression tests and user documentation. Run the repository gate before completion. Rollback is a normal code revert; generated SVGs remain ordinary user files.

# Proposal

## Why

Recraft vector models already appear in genifer, but Styles variants fail on text-only requests because genifer discards the API's required reference count. SVG decoding and saving already exist; users need accurate model requirements, validated style inputs, and a verified path from generation to a saved SVG.

## What Changes

- Preserve advertised reference-count bounds and output formats in provider-neutral capabilities.
- Show SVG output and required-reference guidance in the form and review; enforce reference limits before generation, including when switching models or repeating a request.
- Support the documented Recraft V4 Styles input rules: PNG/JPEG/WebP references with a shortest edge of at least 256 pixels. Preserve existing input behavior for other models.
- Verify native SVG responses are saved unchanged as `.svg` and opened through the existing file opener.
- Document the dedicated Image API request, text-only versus Styles workflows, and additional style-creation billing.
- Keep provider-default output selection, one output per request, and existing output-image pricing semantics. Format selectors, vectorization, advanced style controls, and pricing redesign are out of scope.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `image-provider`: expose reference bounds/output formats, validate reference requirements, and define native SVG response behavior.
- `tui`: communicate output and input requirements and prevent invalid drafts from proceeding to generation.

## Impact

Touches `internal/provider`, `internal/provider/openrouter`, reference loading and form/review flow in `internal/tui`, focused generation/save tests, and `docs/openrouter.md`. Existing `generation-workflow` saving/opening requirements already cover SVG and need no delta. No config migration or new dependency is expected: WebP decoding is available through the existing indirect `golang.org/x/image` module (promote it to direct if imported). Provider remains the only backend seam; pricing failures remain non-fatal.

# Tasks

## 1. Provider capabilities and validation

- [x] 1.1 Extend provider-neutral capabilities and OpenRouter descriptor parsing with optional reference-count bounds, output formats, and reference media/dimension constraints; add the exact four-model Styles enrichment table described in design.md. Verify table-driven discovery tests cover 1–10, 0–1, missing bounds, no reference support, SVG-only, and unknown output formats.
- [x] 1.2 Add shared reference validation and WebP header decoding using the existing x/image module; retain other models' PNG/JPEG/GIF behavior and the existing size cap. Verify boundary tests for 255/256 pixels, valid WebP, GIF on Styles versus other models, malformed bytes, MIME mismatch, and maximum counts without requiring minimum count during attachment.
- [x] 1.3 Validate every adapter Generate call before POST, including zero-reference calls; reuse the unsupported-reference sentinel and add a wrapped invalid-reference sentinel. Verify HTTP tests prove no POST for invalid inputs, valid Styles payloads contain a data URL with `n=1`, ordinary vector text-only requests pass zero-minimum validation, discovery failures are returned, and cancellation survives the additional lookup.
- [x] 1.4 Update provider/OpenRouter package comments as needed and document the capability enrichment's source and limits near the table. Verify documentation matches the metadata fixtures and `go test ./internal/provider/...` passes.

## 2. Attachment commands and form behavior

- [x] 2.1 Move `@path` reads and validation into a command with a draft/model revision guard, committing each batch atomically and rejecting stale completions; apply the same constraints on paste completion. Verify tests cover mixed pasted/path counts, an invalid member in a batch, preserved prompt lines on rejection, empty cleaned prompt without duplicate attachments, and stale results after edit/model switch/navigation.
- [x] 2.2 Guard compose submission, review confirmation, retry, and re-generate with shared input validation; preserve incompatible references after switching between capable models and offer correction. Give provider-side validation failures an actionable edit path without unchanged retry guidance. Verify tests cover missing required references, over-limit carryover, invalid format/dimensions after switching, optional text-only generation, and existing unsupported-model dropping behavior.
- [x] 2.3 Show required/optional reference counts and attachment guidance in compose/review and SVG output guidance for sole-SVG models. Verify view tests at normal and narrow widths, including optional-reference and unknown-format models; update TUI package comments to describe asynchronous attachment handling.
- [x] 2.4 Update `docs/openrouter.md` with text-only vector versus Styles examples using current paste/`@path` controls, required formats/dimensions, and recovery from invalid input. Verify examples match the implemented keybindings and `go test ./internal/tui/...` passes.

## 3. Native SVG workflow and API documentation

- [x] 3.1 Add a mocked OpenRouter generation-to-save regression with real SVG fixture bytes and a valid style reference. Verify the outgoing request uses `/images` and does not force raster output, saved bytes match the response exactly, filename ends in `.svg`, and actual cost is retained; cover existing opener/auto-open behavior using an injected opener without launching a desktop application.
- [x] 3.2 Extend `docs/openrouter.md` with dedicated discovery/endpoint/generation paths, a valid request shape, native SVG response handling, links to official sources, and the distinction between output-image price and additional Styles request fees. Verify against design.md and current response fixtures; record that no paid smoke test was required.

## 4. Integration gate

- [x] 4.1 Run `go build ./...`, `go test ./...`, and `go vet ./...` as separate commands, plus `openspec validate support-vector-models --strict`; verify all succeed and review the diff for config/history compatibility and absence of unrelated changes.

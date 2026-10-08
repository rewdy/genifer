# Tasks

## 1. Clipboard seam and dependency

- [x] 1.1 Add `golang.design/x/clipboard` at an exact pinned version to `go.mod`; verify `go build ./...` and `go mod tidy` leave a clean module graph
- [x] 1.2 Define an `imagePaster` interface in `internal/tui` (e.g. `ReadImage(ctx) ([]byte, error)`) with a default implementation that lazily runs `clipboard.Init()` and `clipboard.Read(ctx, clipboard.FmtImage)`, mapping init/read failure and empty data to a sentinel "no image / unavailable" error; verify a unit test with a fake paster covers the success, empty, and unavailable paths
- [x] 1.3 Add the paster to `tui.Deps` and wire a concrete instance in `main.go`'s construction; verify `go build ./...` succeeds and the TUI constructs without calling `clipboard.Init()` at startup

## 2. Image validation and attachment state

- [x] 2.1 Add a validation helper that decodes clipboard bytes with the standard library, derives the IANA media type, and enforces a size cap, returning a `provider.ReferenceImage` or an error; verify table-driven tests cover a valid PNG, an oversized image, and non-image bytes
- [x] 2.2 Add a `refImages []provider.ReferenceImage` field to the TUI `Model` plus helpers to append and remove-most-recent; verify unit tests cover append, remove, and remove-to-empty transitions

## 3. Compose paste, indicator, and removal

- [x] 3.1 In `handleComposeKey`, bind `ctrl+v` and `alt+v` to a paste command that runs only when the current model is reference-capable (otherwise fall through to the textarea); verify a model-level test that a paste key on an incapable model does not attach and does not consume the key
- [x] 3.2 Handle the paste-result message in `Update`: on success append the validated image and clear status; on "no image / unavailable" set a transient non-fatal status; verify tests drive both outcomes through `Update` with a fake paster
- [x] 3.3 Bind a remove key that drops the most-recent attached image when ≥1 is attached; verify a model-level test that the key reduces the count and returns to the no-image state when cleared
- [x] 3.4 Render inline attached-image pills with a count in `composeView`, and show the paste-key hint only when the model is reference-capable; verify a view test asserts the hint and pill text appear for a capable model and are absent for an incapable one
- [x] 3.5 Add footer keybinding hints for paste (and remove when ≥1 attached) in `footerView`; verify a view test asserts the hints render in the compose phase for a capable model

## 4. Carry images into generation

- [x] 4.1 Populate `gen.Draft{ReferenceImages: m.refImages}` in `Model.generate()`; verify a test asserts the draft/request carries the attached images for a reference-capable model
- [x] 4.2 Surface the attached-image count at the review step in `reviewView`; verify a view test shows the count when images are attached and nothing when none are

## 5. Attach by file path

- [x] 5.1 Add a prompt scanner that extracts sigil-prefixed path lines (first non-space char is the reserved sigil), returning the resolved paths and the cleaned prompt with those lines removed; verify table-driven tests cover a single path, multiple paths, `~` expansion, no sigil lines, and a sigil line amid normal text
- [x] 5.2 On compose submit, read and validate each extracted path via the §2.1 helper, append successes to `refImages`, and build the draft from the cleaned prompt; verify a model-level test that submitting a prompt with a valid `@path` attaches the image and strips the line from the sent prompt
- [x] 5.3 On a missing/unreadable path or a file that fails validation, show a transient non-fatal status and keep the user in compose (do not send); verify tests cover a non-existent path and a non-image file
- [x] 5.4 Render path-attached images through the same §3.4 pills and ensure §3.3 removal applies to them uniformly; verify a view/model test that a path-attached image appears as a pill and can be removed

## 6. Integration verification

- [x] 6.1 Run `go test ./...` and `go vet ./...` (or `just check`) and confirm the full suite passes
- [x] 6.2 Manually verify end to end with `OPENROUTER_API_KEY` set and a reference-capable model: (a) copy an image, press the paste key, confirm the pill appears, remove it, re-paste, generate, and confirm the generated image reflects the reference; (b) submit a prompt containing an `@/path/to/image` line, confirm the image attaches, the line is stripped from the prompt, and generation reflects it; note both results in the change notes

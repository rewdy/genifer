# Proposal

## Why

For models that accept reference images, the backend seam already carries them end to end, and the compose form already advertises "reference images supported" — but there is no way for a user to actually supply one. The attach action promised by the `tui` spec does not exist. Pasting an image from the OS clipboard is the ergonomic pattern users expect from Claude Code, Gemini CLI, and similar tools, and it closes the gap on an otherwise half-built feature. Supplying a local file path is a natural terminal companion to paste: it works where clipboard access does not (SSH, tmux, no display) and is the expected way to reference an image that already exists on disk.

## What Changes

- Add a clipboard-image paste action to the compose phase, available only when the selected model accepts reference images.
- Trigger on a dedicated keypress (`ctrl+v` everywhere, plus `alt+v` as a fallback for terminals that reserve `ctrl+v`). On the keypress the app reads the OS clipboard itself and attaches any image found — it does not rely on the terminal delivering image bytes through its own paste.
- Show a persistent hint in the compose view when the model is reference-capable, telling the user the key to press to paste an image.
- After a successful paste, show an inline pill/chip (Claude Code style) listing each attached image (e.g. `[Image 1]`), with a visible count.
- Provide a way to remove attached images (remove the most-recent with a dedicated key; a cleared set returns the compose view to its no-image state).
- Carry the attached images into generation: populate `gen.Draft.ReferenceImages` in the TUI's `generate()` and surface them at the review step.
- Also let the user attach a reference image by entering a local file path on its own line in the prompt, prefixed with a sigil (e.g. `@/path/to/image.png`). At submit, such lines are extracted, resolved (with `~` expansion), validated, and attached, and are stripped from the prompt text sent to the provider. File-path attach works even where clipboard access does not.
- Validate attached data from either source (decode as a supported image, enforce a size cap, derive the media type) before attaching; present a transient, non-fatal status message when the clipboard holds no image, a referenced file is missing or unreadable, or the data is unusable.
- Degrade gracefully where clipboard access is unavailable (no display, SSH/tmux, missing Linux helper): the paste action reports "no image found / clipboard unavailable" rather than erroring the app, and file-path attach remains available as the fallback.
- Explicitly out of scope: drag-and-drop (not meaningful in a terminal) and remote-clipboard protocols (OSC 52).

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `tui`: The "Capability-adaptive generate form" requirement gains concrete behavior for attaching reference images — pasting from the OS clipboard via a keypress and attaching by a sigil-prefixed file path in the prompt, with an on-screen hint, an attached-image indicator, and removal — replacing the currently unimplemented "offers an action to attach" scenario with testable attach/indicate/remove behavior.

## Impact

- **Code:** `internal/tui` — compose keypress handling (`model.go`), compose view rendering and footer hints (`view.go`, `form.go`), new model state holding attached `provider.ReferenceImage` values, and `generate()`/review wiring. A new clipboard-reading helper (likely a small `internal/tui` or `internal/clipboard` seam) that pulls image bytes per-OS, plus sigil-path extraction applied to the prompt at submit.
- **Dependencies:** adds a clipboard-image mechanism. `atotto/clipboard` (already indirect) is text-only and insufficient; this needs either `golang.design/x/clipboard` (cgo-free on desktop, reads PNG image bytes) or per-OS shell-outs (`osascript` / `wl-paste` / `xclip` / PowerShell). The design document decides between them.
- **Provider/gen layers:** no interface changes — `provider.ReferenceImage`, `GenerateRequest.ReferenceImages`, the OpenRouter `input_references` mapping, and `gen.Draft.ReferenceImages` already exist and are reused as-is.
- **Platform:** behavior varies by OS and terminal; the feature is best-effort and never fatal when the clipboard cannot be read.

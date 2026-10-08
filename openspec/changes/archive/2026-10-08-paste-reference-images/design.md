# Design

## Context

See proposal.md — Why. The lower layers already support reference images end to end: `provider.ReferenceImage`, `GenerateRequest.ReferenceImages`, the OpenRouter `input_references` mapping (base64 data URL via `dataURL(ref)`), and `gen.Draft.ReferenceImages` all exist. The only gap is the TUI: the compose phase collects no image and `Model.generate()` builds `gen.Draft{Prompt, Model}` with no reference images. The compose view already shows a passive "reference images supported" line but binds no action to it.

Key constraints from AGENTS.md and the architecture:
- Dependencies flow `main` → `tui` → (`gen`, `provider`, `config`). Clipboard reading is a TUI concern and must not leak into `provider`/`gen`.
- The TUI is a Bubble Tea `Model`; side effects happen in `tea.Cmd`s, not in `Update`/`View`. Reading the clipboard is a side effect and must run in a command.
- Keep changes minimal; reuse the existing provider/gen seam unchanged.

Terminals do not deliver binary clipboard image data through a keystroke. Research (Claude Code, Gemini CLI, Codex) confirms the robust pattern is: a dedicated keypress triggers the app to read the OS clipboard itself, out-of-band. Plain `cmd+v` on macOS is handled by the terminal and the app never sees image bytes, so it cannot be the trigger.

## Goals / Non-Goals

**Goals:**
- A dedicated keypress in the compose phase reads the OS clipboard and attaches any image found, only when the model is reference-capable.
- An alternate attach path: a sigil-prefixed file-path line in the prompt, resolved and attached at submit, working even where the clipboard is unavailable.
- Visible hint (which key to press) and an inline attached-image indicator with removal.
- Attached images flow into `gen.Draft.ReferenceImages` and the review step.
- Clipboard and file access are best-effort and never fatal.
- Cross-platform (macOS, Linux X11/Wayland, Windows) with a graceful "unavailable" path.

**Non-Goals:**
- No reliance on the terminal's own paste to deliver image bytes.
- No `cmd+v` interception on macOS (not deliverable to a terminal app for image-only clipboards).
- No changes to `provider` or `gen` interfaces.
- No drag-and-drop (not meaningful in a terminal).
- No OSC 52 / remote-clipboard support; SSH and tmux degrade to "no image found" for clipboard paste, with file-path attach as the fallback.
- No persistence of attached images across sessions.

## Decisions

### D1: Trigger key — `ctrl+v` with `alt+v` fallback, not OS-aware `cmd+v`

Bind both `ctrl+v` and `alt+v` to the same "paste reference image" action in the compose phase. This matches Claude Code and Gemini CLI. Rationale:
- `ctrl+v` reaches a Bubble Tea app as `tea.KeyCtrlV` on macOS Terminal/iTerm2 and most Linux terminals.
- Windows Terminal reserves `ctrl+v` for its own paste; `alt+v` is the documented escape hatch there.
- True OS-aware `cmd+v` is not meaningful: v1 Bubble Tea (the pinned `v1.3.10`) does not expose the Super/Cmd modifier, and terminals intercept `cmd+v` anyway.

The paste key is only active when the current model's `AcceptsReferenceImages` is true; otherwise the keystroke falls through to the textarea unchanged (so it never swallows a legitimate paste in an incapable context). The on-screen hint names the key so discovery does not depend on OS detection.

**Alternative considered:** detect OS and bind `cmd+v` on macOS. Rejected — not deliverable through the terminal for image clipboards, and adds OS-detection complexity for no gain.

### D2: Clipboard access via a small internal seam, implementation `golang.design/x/clipboard`

Introduce a narrow TUI-local seam, an `imagePaster` interface (one method, roughly `ReadImage(ctx) ([]byte, error)`), injected into `Deps` so it stays testable with a fake — mirroring how `provider.Provider` is injected. The default implementation uses `golang.design/x/clipboard`:
- Cgo-free on desktop since v0.8.0 (no C toolchain, no libX11/libwayland at build time), so it does not complicate the Go-only build.
- `clipboard.Read(ctx, clipboard.FmtImage)` returns PNG bytes regardless of source format; `clipboard.Init()` failure (e.g. no display) is treated as "unavailable".

The read runs inside a `tea.Cmd` and returns a result message; `Update` never blocks on it.

**Alternative considered:** per-OS shell-outs (`osascript` / `wl-paste` / `xclip` / PowerShell), the Gemini CLI approach. Rejected as the default — more moving parts, runtime helper dependencies on Linux, stderr-leak hazards into the TUI, and the macOS PNGf/JPEG AppleScript dance. The injected seam keeps shell-out available as a future alternative implementation without touching callers.

**Alternative considered:** `atotto/clipboard` (already an indirect dep). Rejected — text-only; cannot read image bytes.

### D3: Attached images live on `Model`, validated before attaching

Add a field to the TUI `Model`, e.g. `refImages []provider.ReferenceImage`. On a successful clipboard read the bytes are validated with the standard library (`image.DecodeConfig` to confirm a supported image and `image.Decode`/sniff for the media type), checked against a size cap, then appended as a `provider.ReferenceImage{Data, MediaType}`. `generate()` changes to `gen.Draft{Prompt, Model, ReferenceImages: m.refImages}`. The review view lists attached-image count alongside the prompt.

Validation up front means the OpenRouter layer's existing capability guard stays a backstop, and a bad clipboard never reaches the network.

**Alternative considered:** attach raw bytes unvalidated and let the provider reject them. Rejected — worse UX (failure surfaces late, after a request) and risks sending garbage.

### D4: Indicator style — inline pills in the compose view (Claude Code style)

Render attached images as inline chips (e.g. `[Image 1] [Image 2]`) within the compose view, above or below the textarea, with a count. This is the pattern the user leaned toward and keeps the attachment visually tied to the prompt being composed. Removal is a dedicated key that drops the most-recently-added image (and the footer advertises it when ≥1 image is attached); repeated presses clear back to the no-image state.

**Alternative considered:** a separate side panel (Kiro style). Rejected for this TUI — the compose view is a single centered column; a side panel would complicate the layout for little benefit at the expected low image counts.

### D5: File-path attach via a sigil-prefixed prompt line, resolved at submit

Attach by path is expressed inline in the prompt rather than through a separate input mode: a line whose first non-space character is a reserved sigil (e.g. `@`) followed by a path marks a reference image (e.g. `@~/shots/ref.png`). At submit — the existing `handleComposeKey` `ctrl+s`/`alt+enter` path — the compose step scans the prompt for such lines, resolves each (trim, `~`/home expansion), reads and validates the file through the **same** D3 validation helper, appends the results to `m.refImages`, and removes those lines from the prompt text before building the draft. The provider therefore never sees the sigil lines.

Rationale:
- Reuses the existing textarea and submit flow — no new modal input, no extra phase, minimal `Update` surface.
- Reuses D3 validation and D2's `provider.ReferenceImage` construction verbatim; the only new logic is "read bytes from a path" versus "read bytes from the clipboard".
- Works where the clipboard cannot be read (SSH, tmux, no display), so it is the reliable fallback the proposal calls for.
- The attached-path images surface through the same D4 pills, so the indicator and removal behavior are shared.

Failure handling: a missing/unreadable path, or a file that fails validation, yields a transient non-fatal status and attaches nothing for that line. A path problem blocks the submit (the user is returned to compose to fix it) rather than silently sending a prompt that still contains the sigil line; a clipboard-paste failure, by contrast, is purely advisory since nothing was requested to send.

**Alternative considered (B vs A):** a dedicated one-line path-input mode toggled by a key (option A). Rejected in favor of the inline sigil — it keeps the single compose surface and avoids a new input phase, at the cost of reserving a sigil and detecting it at submit. Accidental triggering is mitigated by requiring the sigil to start the line.

**Trade-off noted:** the `@`-at-line-start convention means a prompt that legitimately needs a line beginning with `@` must be escaped or reworded. This is an accepted, low-frequency cost; the exact sigil and any escape are an implementation detail left to the task work, constrained by the spec's "on its own line, prefixed with a reserved sigil".

## Risks / Trade-offs

- **Terminal/OS variance in whether `ctrl+v` reaches the app** → Bind `alt+v` as a fallback and always show the key hint; document the Windows `alt+v` case. The feature degrades to "no image attached", never a crash.
- **Clipboard unavailable (no display, SSH, tmux)** → `clipboard.Init()` / read failure maps to a non-fatal "no image found / clipboard unavailable" status; generation proceeds without a reference image. Covered by a spec scenario.
- **New pre-1.0 dependency (`golang.design/x/clipboard`, v0.x API churn)** → Isolated behind the `imagePaster` seam, so a swap to shell-outs or another lib is a one-file change with no caller impact. Pin an exact version per AGENTS.md dependency guidance.
- **Large images bloat the request / memory** → Enforce a size cap at validation and reject oversized data with a status message.
- **Capable models vary in how many references they accept** → Out of scope to model precisely; allow multiple attachments and let the provider enforce its own limit. The existing `ErrReferenceImagesUnsupported` guard remains the backstop for incapable models.
- **`clipboard.Init()` must run once and may panic/err if unavailable** → Initialize lazily inside the paste command and treat any failure as "unavailable"; never call it in `init()` or `New()` where it could affect startup.

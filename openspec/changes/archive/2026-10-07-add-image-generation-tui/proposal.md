# Proposal

## Why

There is no pleasant, local way to generate images from a terminal against OpenRouter's image models. `genifer` fills that gap: a full-screen, visually polished Go TUI that turns a prompt (and optional reference images) into saved image files, remembers your last model, and resolves API credentials from env vars or external commands so secrets never live in config. It is built for OpenRouter now, with seams so a second provider and prompt templates drop in later without rework.

## What Changes

- New `genifer` Go binary: a full-screen Bubble Tea / lipgloss TUI with a gorgeous gradient header.
- Freeform prompt entry with a review step (submit or edit) before a generation is sent.
- Model picker populated from OpenRouter's image model list; the generate form adapts to the selected model's `supported_parameters` (aspect ratio, seed, reference-image support, etc.).
- Image generation via OpenRouter's dedicated `POST /api/v1/images` endpoint (base64 output decoded to a file; extension derived from `media_type`).
- Image-to-image support by attaching reference images (`input_references[]`) when the selected model allows it.
- Generated images are saved to an output directory; the TUI then offers to open the file, with an `auto_open` setting to open automatically.
- In-flight generation shows a spinner and can be cancelled with ESC (HTTP request cancelled via context).
- Hand-edited `config.yaml` separate from app-written `state.json` (which remembers the last selected model), so the app never clobbers user edits.
- Generic, lazily-evaluated value resolver: any config string may be a literal, `{env:VAR}`, or `{cmd:...}` (command run only when the value is needed).
- MVP locks generation to a single image (`n=1`).

## Capabilities

### New Capabilities
- `config`: Loading `config.yaml`, the separate app-written `state.json` for last-used selections, and the generic `{env:...}` / `{cmd:...}` lazy value resolver used for secrets and other settings.
- `image-provider`: The provider abstraction (narrow `Models` / `Generate` interface) and its OpenRouter implementation targeting the dedicated images API, including model discovery, per-model capability reporting, reference-image input, and error mapping (credit/failure).
- `generation-workflow`: Composing a prompt, reviewing it (submit/edit), running generation asynchronously with cancellation, saving the result to disk, and the open / auto-open behavior.
- `tui`: The full-screen Bubble Tea application shell — gradient header, model picker, capability-adaptive generate form, status/spinner, and keybindings — that drives the workflow.

### Modified Capabilities
<!-- None; this is a greenfield project with no existing specs. -->

## Impact

- New Go module and project scaffolding (`go.mod`, `main.go`, internal packages per capability).
- New runtime dependencies: Charm libraries (Bubble Tea, lipgloss, bubbles) as needed; a YAML parser for config.
- New external dependency at runtime: OpenRouter image API (`/api/v1/images`, `/api/v1/images/models`) over HTTPS; requires an OpenRouter API key and account credit.
- New files on the user's machine: `~/.config/genifer/config.yaml`, `~/.config/genifer/state.json`, and an output directory for generated images.
- No existing code or specs are affected (greenfield).

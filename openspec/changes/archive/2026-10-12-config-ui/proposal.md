# Proposal

## Why

Changing settings today means leaving the TUI and hand-editing
`~/.config/genifer/config.yaml`. There is no in-tool way to adjust the common
options — where images are saved, whether history is kept, and whether images
auto-open on completion. A lightweight in-tool config view lets users change
these from inside the app.

## What Changes

- Add a lightweight in-tool configuration view that acts as a graphical editor
  for the user's `config.yaml`, letting the user view and edit a small set of
  options:
  - **save location** (output directory for generated images)
  - **enable history** (on by default; see the prompt-history change)
  - **auto-open image on complete**
- Save edits by writing them back to `config.yaml`. The config view is a direct
  editor of that file — there is no separate override store and no precedence
  layering.
- Apply changed settings to the running session where safe (e.g. a new save
  location and auto-open preference take effect for the next generation without
  a restart).

## Capabilities

### New Capabilities
- `config-ui`: an in-tool graphical editor for the user's `config.yaml` covering
  save location, history enablement, and auto-open-on-complete, writing edits
  back to the file and applying them to the current session.

### Modified Capabilities
<!-- None: no pre-existing specs. -->

## Impact

- `internal/tui/model.go` + `view.go`: a new config phase/view with editable
  fields and a key to open it.
- `internal/config/`: a save path that writes the edited settings back to
  `config.yaml`. The `OutputDir`, `AutoOpen`, and history fields already exist
  in `Config`. This introduces the first app-driven write to `config.yaml`
  (previously only onboarding/starter wrote it), so the writer must preserve the
  rest of the user's file as much as practical.
- `main.go` / `internal/tui`: let a changed save location and auto-open take
  effect for the running session.
- Depends on the prompt-history change for the `history` config field; if applied
  first, the history toggle is deferred to that change.

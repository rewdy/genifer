# Configuration

genifer reads two files from your platform config directory
(`~/.config/genifer` on Linux/macOS):

- `config.yaml` — **you own this.** genifer never writes to it.
- `state.json` — **genifer owns this.** It stores last-used selections (e.g. the
  last model and last aspect ratio) and is rewritten freely. Don't hand-edit it.
- `pricing-cache.json` — **genifer owns this.** It caches per-model prices for
  24h so the picker doesn't refetch pricing on every launch. Safe to delete; it
  will be rebuilt. Don't hand-edit it.
- `history-cache.json` — **genifer owns this.** It records your past prompts and
  their parameters (see `history` below). It keeps every entry, so to clear your
  history just delete this file. Safe to delete; it will be recreated on the next
  generation. Don't hand-edit it.

If `config.yaml` is missing, genifer runs with defaults and tells you where it
looked. A malformed `config.yaml` is a hard error (it won't silently fall back).

## Editing the config

```sh
genifer config        # create a starter config.yaml (if missing), then open it
genifer config path   # print the config.yaml path and exit
```

`genifer config` opens the file in your editor, resolved as `$VISUAL`, then
`$EDITOR`, then an OS default (macOS `open -t`, Linux `xdg-open`, Windows
`notepad`). On first use it writes a commented starter `config.yaml`; it never
overwrites an existing one. `genifer config path` just prints the path (handy
for scripting) without creating or opening anything.

You can also edit the common settings from inside the TUI: press `c` on the
model picker to open the config view, which is a direct editor of your
`config.yaml`. It edits five keys — `output_dir` (save location), `auto_open`,
`history`, `smart_filenames`, and `filename_model` — writing them back to the
file while leaving your comments and other settings (such as the `api_key`
directive) intact. A changed save location, auto-open preference, and smart
file naming also take effect for the current session, so you don't need to
restart. Everything else in `config.yaml` stays file-/onboarding-managed.

New users don't need either command: on first launch with no config, genifer
walks you through creating one — see the first-run section of the
[README](../README.md).

## config.yaml

```yaml
# Active provider. Only "openrouter" is supported today.
provider: openrouter

openrouter:
  # The OpenRouter API key. See value directives below — avoid a literal key.
  api_key: "{env:OPENROUTER_API_KEY}"

# Where images are written. Options:
#   omit        -> ~/Pictures/genifer (default)
#   "." or "pwd" -> the directory you launched genifer from
#   relative     -> resolved against the launch directory
#   absolute     -> used as-is
output_dir: "."

# Open generated images automatically, with no prompt.
auto_open: false

# Override the per-OS command used to open images. Omit to use the OS default
# (macOS: open, Linux: xdg-open, Windows: start).
# open_command: "feh"

# Record successful generations to history-cache.json so you can recall and
# reuse past prompts. Enabled by default; set to false to turn it off (nothing
# is written and the history view reports that it's off). History keeps every
# entry — delete history-cache.json to clear it.
history: true

# Generate descriptive file names using a text model instead of timestamps.
# When enabled, each saved image is named from a short text completion of your
# prompt using filename_model (default: timestamps). Naming is best-effort:
# a failed or empty reply falls back to the timestamped name.
smart_filenames: false

# The OpenRouter text model used to generate descriptive file names when
# smart_filenames is enabled. Any model from https://openrouter.ai/models
# works; a small, cheap one (e.g. openai/gpt-4o-mini) is fine. May be a value
# directive. Omit or leave empty to keep timestamped names.
# filename_model: "openai/gpt-4o-mini"
```

## Value directives

Any string setting may be a literal, or one of two directives. Resolution is
**lazy** (a command runs only when its value is actually needed) and
**single-level** (a resolved value is not scanned again for directives).

| Form           | Meaning                                                        |
| -------------- | -------------------------------------------------------------- |
| `plain text`   | Used verbatim.                                                 |
| `{env:NAME}`   | The value of environment variable `NAME`. Error if unset.     |
| `{cmd:...}`    | The trimmed stdout of running `...` via the shell. Error if it exits non-zero. |

Examples:

```yaml
openrouter:
  # From an environment variable:
  api_key: "{env:OPENROUTER_API_KEY}"

  # Or fetched from a secrets manager at the moment it's needed:
  # api_key: "{cmd:op read op://vault/openrouter/key}"
```

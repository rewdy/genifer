# Configuration

genifer reads two files from your platform config directory
(`~/.config/genifer` on Linux/macOS):

- `config.yaml` — **you own this.** genifer never writes to it.
- `state.json` — **genifer owns this.** It stores last-used selections (e.g. the
  last model) and is rewritten freely. Don't hand-edit it.
- `pricing-cache.json` — **genifer owns this.** It caches per-model prices for
  24h so the picker doesn't refetch pricing on every launch. Safe to delete; it
  will be rebuilt. Don't hand-edit it.

If `config.yaml` is missing, genifer runs with defaults and tells you where it
looked. A malformed `config.yaml` is a hard error (it won't silently fall back).

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

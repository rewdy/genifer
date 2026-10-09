# Configuration

genifer reads these files from your platform config directory
(`~/.config/genifer` on Linux/macOS):

- `config.yaml` — **you own this.** genifer never writes to it during normal
  operation (the one exception is legacy recovery, below, which backs the old
  file up first).
- `state.json` — **genifer owns this.** It stores last-used selections (the
  last model together with its provider, and the last aspect ratio) and is
  rewritten freely. Don't hand-edit it.
- `pricing-cache.json` — **genifer owns this.** It caches per-model prices for
  24h so the picker doesn't refetch pricing on every launch. Safe to delete; it
  will be rebuilt. Don't hand-edit it.

If `config.yaml` is missing, genifer runs first-run onboarding to create one. A
malformed `config.yaml` is a hard error (it won't silently fall back).

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

New users don't need either command: on first launch with no config, genifer
walks you through creating one — see the first-run section of the
[README](../README.md).

## config.yaml

Providers are a **list** of keyed, typed instances under `providers:`. Each
entry has a unique `key` (its identity in the picker and in saved state), a
`type` (which selects the implementation), and the fields that type needs. Two
instances may share a `type` while differing by `key`.

```yaml
providers:
  # Hosted OpenRouter provider.
  - key: or
    type: openrouter
    # The OpenRouter API key. See value directives below — avoid a literal key.
    api_key: "{env:OPENROUTER_API_KEY}"
  # Local A1111-compatible WebUI (A1111, Forge, or ComfyUI with the shim).
  - key: local
    type: a1111
    base_url: "http://127.0.0.1:7860"

# Where images are written. Options:
#   omit        -> ~/Pictures/genifer (default)
#   "." or "pwd" -> the directory you launched genifer from
#   relative     -> resolved against the launch directory
#   absolute     -> used as-is
output_dir: "{env:HOME}/Pictures/genifer"

# Open generated images automatically, with no prompt.
auto_open: false

# Override the per-OS command used to open images. Omit to use the OS default
# (macOS: open, Linux: xdg-open, Windows: start).
# open_command: "feh"
```

The model picker merges the models of every configured provider, grouped by
`key`. A model is identified by `(key, model id)`, so two providers can offer
the same model id without colliding. A provider that is unreachable at launch
(e.g. a local WebUI that isn't running) is shown as offline — its models are
omitted while every reachable provider's models stay usable.

### Provider types

| `type`       | Fields it uses       | Notes                                            |
| ------------ | -------------------- | ------------------------------------------------ |
| `openrouter` | `api_key`, `base_url` (optional) | Hosted. See [OpenRouter notes](openrouter.md). |
| `a1111`      | `base_url`           | Local A1111-compatible WebUI. See [local provider](local-provider.md). |

### Adding more providers by hand

Onboarding writes a single instance. To run more than one, add entries to the
`providers:` list yourself — for example a hosted OpenRouter plus two local
boxes:

```yaml
providers:
  - key: or
    type: openrouter
    api_key: "{env:OPENROUTER_API_KEY}"
  - key: laptop
    type: a1111
    base_url: "http://127.0.0.1:7860"
  - key: gpu-box
    type: a1111
    base_url: "http://10.0.0.2:7860"
```

Each `key` must be unique and non-empty, and each `type` must be recognized;
otherwise genifer reports a clear error naming the offending instance.

## Legacy config recovery

Earlier genifer used a single top-level `provider:` string plus a type-named
block (e.g. `openrouter:`). That shape no longer loads. On launch genifer
detects it, renames the file to `config.yaml.bak` (choosing a non-colliding
name if a `.bak` already exists, so an existing backup is never overwritten),
and routes you into onboarding to rebuild it in the list shape. Your old file is
preserved in the backup — nothing is destroyed.

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
providers:
  - key: or
    type: openrouter
    # From an environment variable:
    api_key: "{env:OPENROUTER_API_KEY}"
    # Or fetched from a secrets manager at the moment it's needed:
    # api_key: "{cmd:op read op://vault/openrouter/key}"
```

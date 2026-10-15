# genifer

A cute, full-screen terminal UI for generating images with
[OpenRouter](https://openrouter.ai) image models.

![genifer demo](demo/genifer.gif)

- Freeform prompt with a review step before you send
- Model picker that remembers your last choice
- Per-model pricing shown in the picker (`$X/img`, `per token`, `free`), cached daily
- Form that adapts to each model's capabilities (aspect ratio, seed, image-to-image)
- Image-to-image via reference images, when the model supports it
- Saves generated images and offers to open them (or auto-open)
- Iterate from the result screen: try a new model, start fresh, or re-generate
- Auto-generates and shows a seed for seed-capable models, so you can reproduce
  or vary a result
- Remembers your past prompts (on-device, opt-out) so you can recall and reuse
  them
- Shows the actual cost of each generation
- Config values can come from env vars or external commands — secrets stay out of config

## Install

Requires Go 1.24+.

```sh
go install github.com/rewdy/genifer@latest
```

Or build from a clone:

```sh
git clone https://github.com/rewdy/genifer.git
cd genifer
go build -o genifer .
```

## Configure

Set your OpenRouter API key (the default config reads it from this variable):

```sh
export OPENROUTER_API_KEY="sk-or-..."
```

That's the only required setup. To customize, create
`~/.config/genifer/config.yaml`:

```yaml
provider: openrouter

openrouter:
  api_key: "{env:OPENROUTER_API_KEY}"

# Where images are written.
#   "." or "pwd" -> the directory you launched genifer from
#   omit         -> ~/Pictures/genifer (default)
#   also accepts a relative or absolute path
output_dir: "."

# Open each generated image automatically
auto_open: false
```

See [docs/config.md](docs/config.md) for the full schema and the
`{env:...}` / `{cmd:...}` value directives, and
[docs/openrouter.md](docs/openrouter.md) for provider details.

To create and edit the config without writing it by hand:

```sh
genifer config        # create a starter config.yaml (if missing), then open it
genifer config path   # print the config.yaml path
```

`genifer config` opens your editor (`$VISUAL`, then `$EDITOR`, then an OS
default) on a commented starter file. It never overwrites an existing config.

### First run

The very first time you launch genifer with no config, it greets you with a
`WELCOME` screen and asks how it should get your OpenRouter API key:

- **Environment variable** — reads it from a variable like `OPENROUTER_API_KEY`
  (recommended; written as `{env:NAME}`).
- **Command** — runs a command to fetch it, e.g. a secrets manager (written as
  `{cmd:...}`).
- **Paste a key** — stores the key directly; genifer warns first that it will be
  saved as plain text and asks you to confirm.

Your choice is written into a new `config.yaml` and genifer continues straight
into the model picker. On later launches the config exists, so onboarding is
skipped.

> Note: `config.yaml` is yours to edit. genifer stores its own state in
> `~/.config/genifer/state.json` (last model and last aspect ratio) and
> `~/.config/genifer/pricing-cache.json`
> (per-model prices, refreshed daily) — don't hand-edit those.

## Run

```sh
genifer
```

### Example session

1. genifer opens full-screen and loads the available image models.
2. Pick a model with `↑/↓` (press `/` to filter the list by name), then
   `enter` (your choice is remembered next time).
3. Type a prompt. If the model offers a choice of output sizes, press
   `ctrl+a` to open a dialog listing the available aspect ratios and pick one
   (your last choice is remembered). Then press `ctrl+s` to review it.
4. At the review screen, press `enter` to generate or `e` to edit.
5. While generating, press `esc` to cancel.
6. On success, the image is saved and you can press `o` to open it
   (or enable `auto_open` to skip the prompt). From here you can also `t` try
   the same prompt on a new model, `n` start a new image, or `g` re-generate.
   For seed-capable models genifer auto-generates a seed and shows it, so you
   can `g` reproduce the result with the same seed or `v` vary it with a fresh
   one.

### Keys

| Context    | Keys                                      |
| ---------- | ----------------------------------------- |
| Picker     | `↑/↓` move · `/` filter · `enter` select · `h` history · `c` settings · `q` quit |
| Compose    | `ctrl+s` review · `esc` back · `↑/↓` recall history · `ctrl+p` history list · `ctrl+a` aspect ratio (when offered) |
| Review     | `enter` generate · `e` edit               |
| History    | `↑/↓` move · `/` filter · `enter` reuse · `esc` back |
| Settings   | `↑/↓` move · `space` toggle · `enter` save · `esc` cancel |
| Generating | `esc` cancel                              |
| Result (success) | `o` open · `f` open folder · `p` edit prompt · `m` try new model · `n` new image · `g` re-generate (seed models: same seed) · `v` re-generate varying the seed · `q` quit |
| Result (failure) | `r` retry (when retryable) · `enter` new · `q` quit |

### History

genifer records each successful generation — prompt, model, aspect ratio, the
seed used, saved path, and timestamp — to `history-cache.json` in the config
dir. Press `h` from the picker or `ctrl+p` while composing to open the history
view, then `enter` to reuse a prompt (its aspect ratio and seed are reapplied
where the selected model supports them). The view shows your most-recent
prompts; the full record lives in the file, and the view points you to it when
you page past the loaded entries or filter to no matches. History is on-device
only, kept forever (delete the file to clear it), and can be turned off with
`history: false` in `config.yaml`.

You can also recall prompts inline while composing, like a shell: press `↑` to
step back through recent prompts and `↓` to come forward again. Recall only
kicks in at the edges of the text — `↑` on the first line, `↓` on the last —
so the arrows still move the cursor within a multi-line prompt. Pressing `↓`
past the newest entry restores whatever you were typing before you started
recalling.

## Account requirements

Image generation requires an OpenRouter account with more than $1 of credit.
See [docs/openrouter.md](docs/openrouter.md).

## Development

Common tasks run through [`just`](https://github.com/casey/just):

```sh
just          # list recipes
just check    # build + test + vet
```

### Cutting a release

Versions are SemVer tags with a `v` prefix (the reported version comes from
`git describe --tags`, so a release *is* a tag — there's no constant to bump).
`just release` enforces that:

```sh
just release v0.3.0 "config command and first-run onboarding"
```

It validates the version, refuses a dirty tree or an already-used tag, runs the
full build/test/vet gate, then creates an annotated tag (signed when a signing
key is configured). It does **not** push — on success it prints the
`git push origin <tag>` command to run when you're ready.

To bump from the latest tag instead of naming a version, use `just bump`, which
computes the next version and routes through `release` (same guards and gate):

```sh
just bump              # patch: v0.3.0 -> v0.3.1
just bump minor        # v0.3.1 -> v0.4.0
just bump major "big rewrite"
```

The level defaults to `patch`. If there are no tags yet, `bump` won't guess —
cut the first release explicitly with `just release v0.1.0`.

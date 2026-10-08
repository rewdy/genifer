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

> Note: `config.yaml` is yours to edit. genifer stores its own state in
> `~/.config/genifer/state.json` (last model) and `~/.config/genifer/pricing-cache.json`
> (per-model prices, refreshed daily) — don't hand-edit those.

## Run

```sh
genifer
```

### Example session

1. genifer opens full-screen and loads the available image models.
2. Pick a model with `↑/↓` (press `/` to filter the list by name), then
   `enter` (your choice is remembered next time).
3. Type a prompt, then press `ctrl+s` to review it.
4. At the review screen, press `enter` to generate or `e` to edit.
5. While generating, press `esc` to cancel.
6. On success, the image is saved and you can press `o` to open it
   (or enable `auto_open` to skip the prompt).

### Keys

| Context    | Keys                                      |
| ---------- | ----------------------------------------- |
| Picker     | `↑/↓` move · `/` filter · `enter` select · `q` quit |
| Compose    | `ctrl+s` review · `esc` back              |
| Review     | `enter` generate · `e` edit               |
| Generating | `esc` cancel                              |
| Result     | `o` open · `r` retry · `enter` new · `q` quit |

## Account requirements

Image generation requires an OpenRouter account with more than $1 of credit.
See [docs/openrouter.md](docs/openrouter.md).

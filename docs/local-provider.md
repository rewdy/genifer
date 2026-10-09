# Local provider (A1111-compatible)

genifer can generate images through a WebUI you run on your own machine, instead
of a hosted service. The local provider talks to the **A1111-compatible HTTP
API** (`/sdapi/v1/*`), so generation runs on your hardware and every local model
is free.

## Supported runtimes

The `/sdapi/v1/*` contract is a de-facto standard. These runtimes speak it:

- **AUTOMATIC1111** `stable-diffusion-webui` — the original, speaks it natively.
- **Forge** — a drop-in A1111 fork, speaks it natively.
- **ComfyUI** — speaks it via a community shim/extension that exposes the A1111
  endpoints. ComfyUI's native graph API is not used; install the shim if you
  want genifer to reach ComfyUI.

You must start the WebUI with its HTTP API enabled (for A1111/Forge, launch with
`--api`).

## Configuration

Add an `a1111` instance to the `providers:` list in `config.yaml`:

```yaml
providers:
  - key: local
    type: a1111
    base_url: "http://127.0.0.1:7860"
```

- `base_url` is the address of the running WebUI. The default offered during
  onboarding is `http://127.0.0.1:7860`.
- Run several by adding more `a1111` entries with distinct `key`s (e.g. a laptop
  and a GPU box). See [config.md](config.md#adding-more-providers-by-hand).

## What the local provider does

- **Model discovery** — lists the WebUI's installed checkpoints via
  `GET /sdapi/v1/sd-models`.
- **Generation** — posts the prompt to `POST /sdapi/v1/txt2img`, mapping the
  selected aspect ratio to output width/height, and decodes the returned image.
  Whatever checkpoint the WebUI currently has loaded is what generates.
- **Pricing** — always free.
- **Capabilities** — fixed for the first cut: seed supported, a fixed set of
  aspect ratios, and **reference images are not accepted** (img2img/ControlNet
  are out of scope). A request with reference images is rejected before any call
  is made.

## Offline handling

If the WebUI isn't running, genifer marks that instance **offline** at launch:
its models are omitted and it's noted in the status line, while every reachable
provider's models stay listed and usable. One unreachable local endpoint never
blanks the picker or errors the app.

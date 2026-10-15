# OpenRouter provider

genifer generates images through OpenRouter's dedicated images API
(`POST /api/v1/images`) and discovers models via `GET /api/v1/images/models`.

## Credentials

An OpenRouter API key is required. By default genifer resolves it from the
`OPENROUTER_API_KEY` environment variable:

```sh
export OPENROUTER_API_KEY="sk-or-..."
```

This matches the default `config.yaml`:

```yaml
openrouter:
  api_key: "{env:OPENROUTER_API_KEY}"
```

You can point `api_key` at any value directive (see [config.md](config.md)) —
for example, fetch it from a secrets manager at the moment it's needed:

```yaml
openrouter:
  api_key: "{cmd:op read op://vault/openrouter/key}"
```

The key is resolved **lazily** — only when a request is actually made — so a
command-backed key never runs on startup.

The key is sent as `Authorization: Bearer <key>`. An empty or rejected key
surfaces as an authentication error in the TUI.

## Account requirements

- Paid image generation requires more than $1 of account credit, otherwise the
  API returns 402 and genifer reports an "insufficient credit" error.
- Failed generations return 502 (and are not billed); genifer reports a
  retryable "generation failed" error.

## Attribution headers (optional)

OpenRouter accepts optional attribution headers. genifer can send
`HTTP-Referer` and `X-Title`; a title only counts when a referer is also set.
These are optional and off by default.

## Vector models and style references

Native vector models produce SVG files through the same generation workflow as
raster models. For example, Recraft V4 Vector supports a text-only prompt;
Recraft V4 Styles Vector requires style reference images. A style reference
teaches the model the visual style to use for a new image, rather than asking
it to edit the reference itself.

Model discovery supplies reference-count limits. The form shows whether
references are optional or required, the maximum when known, and the current
attachment count. A model advertising only SVG output shows `Output: SVG` in
compose and review. There is no output-format selector: generation uses the
model's native default.

For Recraft V4 Styles, Styles Pro, Styles Vector, and Styles Pro Vector:

- Attach 1–10 style images in PNG, JPEG, or WebP format.
- Each image must be at least 256 pixels wide and 256 pixels high.
- genifer also limits each reference to 20 MiB.

To attach a reference, paste an image with **ctrl+v** (or **alt+v**), or put its
local path on a separate prompt line prefixed by `@`:

```text
A flat vector illustration of a red bird
@~/Pictures/style-reference.png
```

Use **ctrl+s** (or **alt+enter**) to review, then **enter** to generate. Path
lines are read in the background and removed from the submitted prompt after
the batch loads successfully. If a path is invalid or an attachment exceeds the
model's limits, genifer preserves the draft and shows how to correct it. Remove
the last attachment with **ctrl+r**, replace it with a suitable image, or switch
to a model with optional references for text-only generation. References carried
between capable models stay attached, but must satisfy the new model's rules.

## Image API request and response

The relevant endpoints share the `https://openrouter.ai` host:

| Operation | Path |
| --- | --- |
| Discover models and advertised parameters | `GET /api/v1/images/models` |
| Inspect a model's endpoint capabilities and pricing | `GET /api/v1/images/models/{author}/{slug}/endpoints` |
| Generate one image | `POST /api/v1/images` |

The discovery descriptor for `input_references` includes `min` and `max` count
bounds when known. Model-level capabilities combine endpoints; per-endpoint
records are definitive, so a valid model-level request can still be rejected by
upstream routing. Format/dimension rules for the four Recraft Styles models are
currently documented on their model pages rather than included in discovery;
genifer supplements those models' capabilities in the OpenRouter adapter.

For example, the following Python script constructs a request using a local PNG
style reference (at least 256 by 256 pixels):

```python
import base64
import json
import os
import urllib.request
from pathlib import Path

reference = base64.b64encode(Path("style-reference.png").read_bytes()).decode("ascii")
payload = {
    "model": "recraft/recraft-v4-styles-vector",
    "prompt": "A flat vector illustration of a red bird",
    "n": 1,
    "aspect_ratio": "1:1",
    "input_references": [{
        "type": "image_url",
        "image_url": {"url": "data:image/png;base64," + reference},
    }],
}
request = urllib.request.Request(
    "https://openrouter.ai/api/v1/images",
    data=json.dumps(payload).encode(),
    headers={
        "Authorization": "Bearer " + os.environ["OPENROUTER_API_KEY"],
        "Content-Type": "application/json",
    },
)
# Running this submits a paid generation request.
with urllib.request.urlopen(request, timeout=120) as response:
    result = json.load(response)
image = result["data"][0]
assert image["media_type"] == "image/svg+xml"
Path("result.svg").write_bytes(base64.b64decode(image["b64_json"]))
```

No extra vector endpoint, style-creation request, or `output_format` parameter
is needed for this model. genifer keeps its default output and saves the decoded
SVG bytes unchanged with a `.svg` extension. Manual open and `auto_open` use the
same system viewer or configured opener as raster outputs. If the service omits
the media type, genifer retains the existing `.img` fallback.

The picker displays **output-image pricing**, which can exclude other request
fees. Recraft V4 Styles Vector currently has an additional $0.005 style-creation
fee per request, on top of $0.05 per output image ($0.055 for genifer's one-image
request). Other Styles variants have different image prices. The result screen
shows the actual total cost reported by the API; unavailable pricing never
blocks generation.

Sources (checked 2026-10-14): [Image API documentation](https://openrouter.ai/docs/guides/overview/multimodal/image-generation),
[Styles Vector model requirements](https://openrouter.ai/recraft/recraft-v4-styles-vector),
[Styles Vector endpoint metadata](https://openrouter.ai/api/v1/images/models/recraft/recraft-v4-styles-vector/endpoints).
The implementation was verified with local HTTP fixtures and SVG save/open
tests; no paid generation smoke test was required or run.

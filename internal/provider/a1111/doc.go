// Package a1111 implements provider.Provider against a running
// A1111-compatible WebUI over HTTP. The /sdapi/v1/* contract is honored by
// AUTOMATIC1111's stable-diffusion-webui, Forge, and ComfyUI (via a community
// shim), so one client reaches all three.
//
// Model discovery uses GET /sdapi/v1/sd-models; generation uses POST
// /sdapi/v1/txt2img. All models are reported free (generation runs on the
// user's own hardware) with hardcoded capabilities (seed supported, a fixed
// aspect-ratio-to-dimensions map, reference images not accepted). An
// unreachable endpoint maps to provider.ErrUnreachable so the app can mark the
// instance offline without aborting.
package a1111

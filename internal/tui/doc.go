// Package tui implements the full-screen Bubble Tea application shell: header,
// model picker, capability-adaptive generate form, prompt history, an in-tool
// settings editor, status, and keybindings.
//
// For reference-capable models the compose form attaches reference images two
// ways: a clipboard paste keypress (read out-of-band via the imagePaster seam)
// and sigil-prefixed file-path lines in the prompt, resolved asynchronously at
// submit. Results are committed only to the unchanged draft that requested them. Both
// paths validate the image, surface it as an inline pill, and feed
// gen.Draft.ReferenceImages. Clipboard and file access are best-effort and
// never fatal. Known reference count, format, and dimension requirements are
// enforced before generation; sole-SVG models show their native output format.
//
// A successful generation is appended to the app-owned history store (best-
// effort, gated by the history config option). The history view loads only the
// most-recent entries for responsiveness and points the user to the on-disk
// file for the full record; selecting an entry pre-fills compose, reapplying
// aspect ratio and seed where the active model supports them.
//
// Native result thumbnails are optional. Image preparation and terminal geometry
// run in commands; the output adapter owns protocol emission and cleanup while
// View lays out ordinary text and an opaque placement marker.
package tui

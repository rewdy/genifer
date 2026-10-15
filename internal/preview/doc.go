// Package preview prepares optional local image thumbnails and serializes native
// terminal graphics with the application's text renderer. Its Writer owns all
// graphics output; models only emit zero-width placement markers. Commands must
// never write image protocols directly to stdout. Raster decoding is serialized,
// limited to 64 MiB files and 32 megapixels, and checked against a five-second
// context deadline. Only fitted thumbnails are retained; SVG is omitted.
package preview

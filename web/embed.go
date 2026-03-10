package web

import "embed"

// DistFS embeds the frontend build output.
// The directory will be populated by the frontend build step.
// In development mode, this is not used (requests are proxied to Vite).
//
//go:embed all:frontend/build
var DistFS embed.FS

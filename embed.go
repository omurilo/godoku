package godoku

import "embed"

//go:embed static/*
var StaticFS embed.FS

//go:embed templates/*
var TemplatesFS embed.FS

// AppFS holds the React shell (App.tsx, components, index.css, tsconfig.json).
// The MDX builder materializes it to a temp dir at build time so esbuild can
// resolve its files on disk. `all:` ensures nested directories are embedded.
//
//go:embed all:app
var AppFS embed.FS

# Repository Guide

- This is a Go library for writing comic and manga chapters as CBZ, PDF, EPUB, or directories.
- Keep the public API small and idiomatic.
- Put format-specific behavior in the corresponding top-level file and metadata logic in `metadata/`.
- Format Go code with `gofmt`, but do not run or build the code unless the user asks.
- Run `go vet` whenever any `.go` file is changed.
- Do not add generated files, build artifacts, or unrelated dependency changes.

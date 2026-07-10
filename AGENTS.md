# comicfile

`comicfile` is a Go package for creating comic and manga chapters as CBZ, PDF, EPUB, or directories.

# Project Structure

```
.
├── cbz
├── dir
├── epub
├── examples
│   ├── cbz
│   ├── dir
│   ├── epub
│   └── pdf
├── internal
│   └── container
├── metadata
├── pdf
└── testdata
```

- `comicfile.go` and `doc.go`: public API and package documentation.
- `cbz/`, `pdf/`, `epub/`, `dir/`: format-specific readers and writers.
- `metadata/`: comic metadata types and logic.
- `internal/container/`: shared internal container code.
- `examples/`: runnable examples for each output format.
- `testdata/`: test fixtures.

# Development Guidelines

- Use `rg` instead of `grep` to search text or files.
- Keep the public API small and idiomatic.
- Format Go code with `gofmt`, but do not run or build the code unless the user asks.
- Run `go vet` whenever any `.go` file is changed.
- Do not add generated files, build artifacts, or unrelated dependency changes.

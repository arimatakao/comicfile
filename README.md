# comicfile

Go package for creating manga or comic chapter files from image pages. It provides a common container interface and can write output as CBZ, PDF, EPUB, or a plain directory, with optional comic metadata.

## Installation

```sh
go get github.com/arimatakao/comicfile
```

Runnable examples are grouped by format in the [examples](./examples)
directory. Each uses `-c` to create its `file.*` container and `-r` to read it.

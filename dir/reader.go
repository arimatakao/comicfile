package dir

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"

	"github.com/arimatakao/comicfile/internal/container"
	"github.com/arimatakao/comicfile/metadata"
)

type dirReader struct {
	pages    []image.Image
	errPages int
}

// Open creates a reader for the image files in path. Pages are
// ordered by filename, matching the naming convention used by dirContainer.
func Open(path string) (*dirReader, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	reader := &dirReader{pages: make([]image.Image, 0, len(entries))}
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}

		page, err := readImage(filepath.Join(path, entry.Name()))
		if err != nil {
			reader.errPages++
			continue
		}
		reader.pages = append(reader.pages, page)
	}

	return reader, nil
}

// TotalPages returns the number of readable images in the directory.
func (d *dirReader) TotalPages() int {
	return len(d.pages)
}

// ErrPages returns the number of pages that could not be read.
func (d *dirReader) ErrPages() int {
	return d.errPages
}

// Metadata returns the metadata stored in the directory container.
func (d *dirReader) Metadata() *metadata.Metadata {
	return nil
}

// Page returns the decoded image at index.
func (d *dirReader) Page(index int) (image.Image, error) {
	if index < 0 || index >= len(d.pages) {
		return nil, container.ErrPageIndexOutOfRange
	}

	return d.pages[index], nil
}

func readImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	page, _, err := image.Decode(file)
	return page, err
}

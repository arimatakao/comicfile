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
	pages    []string
	errPages int
}

// Open creates a reader for image files in path. It records valid page paths
// without decoding their pixels.
func Open(path string) (*dirReader, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	reader := &dirReader{pages: make([]string, 0, len(entries))}
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		pagePath := filepath.Join(path, entry.Name())
		if _, err := imageConfig(pagePath); err != nil {
			reader.errPages++
			continue
		}
		reader.pages = append(reader.pages, pagePath)
	}
	return reader, nil
}

func (d *dirReader) TotalPages() int              { return len(d.pages) }
func (d *dirReader) ErrPages() int                { return d.errPages }
func (d *dirReader) Metadata() *metadata.Metadata { return nil }

// Page decodes and returns one page on demand.
func (d *dirReader) Page(index int) (image.Image, error) {
	if index < 0 || index >= len(d.pages) {
		return nil, container.ErrPageIndexOutOfRange
	}
	return decodeImage(d.pages[index])
}

// Close implements comicfile.ContainerReader. Directory readers own no open
// resources between Page calls.
func (*dirReader) Close() error { return nil }

func imageConfig(path string) (image.Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return image.Config{}, err
	}
	defer file.Close()
	config, _, err := image.DecodeConfig(file)
	return config, err
}

func decodeImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	page, _, err := image.Decode(file)
	return page, err
}

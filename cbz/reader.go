package cbz

import (
	"archive/zip"
	"encoding/json"
	"encoding/xml"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"sync"

	"github.com/arimatakao/comicfile/internal/container"
	"github.com/arimatakao/comicfile/metadata"
)

type cbzReader struct {
	mu       sync.RWMutex
	archive  *zip.ReadCloser
	pages    []*zip.File
	errPages int
	metadata metadata.Metadata
	closed   bool
}

// Open creates a reader for image files in a CBZ archive. Page pixels are
// decoded lazily by Page.
func Open(path string) (*cbzReader, error) {
	archive, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}

	reader := &cbzReader{archive: archive, pages: make([]*zip.File, 0, len(archive.File))}
	if archive.Comment != "" {
		_ = json.Unmarshal([]byte(archive.Comment), &reader.metadata.CBI)
	}
	for _, file := range archive.File {
		if file.FileInfo().IsDir() {
			continue
		}
		if file.Name == "ComicInfo.xml" {
			_ = readComicInfo(file, &reader.metadata.CI)
			continue
		}
		if _, err := imageConfig(file); err != nil {
			reader.errPages++
			continue
		}
		reader.pages = append(reader.pages, file)
	}
	return reader, nil
}

func (c *cbzReader) TotalPages() int { return len(c.pages) }

func (c *cbzReader) ErrPages() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.errPages
}

func (c *cbzReader) Metadata() *metadata.Metadata { return &c.metadata }

// Page decodes and returns one page from the still-open archive.
func (c *cbzReader) Page(index int) (image.Image, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if index < 0 || index >= len(c.pages) {
		return nil, container.ErrPageIndexOutOfRange
	}
	if c.closed {
		return nil, os.ErrClosed
	}
	return decodeImageFile(c.pages[index])
}

// Close releases the open CBZ archive.
func (c *cbzReader) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	return c.archive.Close()
}

func imageConfig(file *zip.File) (image.Config, error) {
	reader, err := file.Open()
	if err != nil {
		return image.Config{}, err
	}
	defer reader.Close()
	config, _, err := image.DecodeConfig(reader)
	return config, err
}

func decodeImageFile(file *zip.File) (image.Image, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	page, _, err := image.Decode(reader)
	return page, err
}

func readComicInfo(file *zip.File, comicInfo *metadata.ComicInfoMetadata) error {
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()
	return xml.NewDecoder(reader).Decode(comicInfo)
}
